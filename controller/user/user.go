package user

import (
	"context"
	"errors"
	"fmt"
	"myapp/dto"
	"myapp/middleware"
	"myapp/model"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UserController(router *gin.Engine, firestoreClient *firestore.Client) {
	routes := router.Group("/user", middleware.AccessTokenMiddleware())
	{
		routes.POST("/search", func(c *gin.Context) {
			SearchUser(c, firestoreClient)
		})
		routes.PUT("/profile", func(c *gin.Context) {
			UpdateProfileUser(c, firestoreClient)
		})
		routes.DELETE("/account", func(c *gin.Context) {
			DeleteUser(c, firestoreClient)
		})
	}
}

func SearchUser(c *gin.Context, fb *firestore.Client) {
	var emailReq dto.SearchEmailRequest
	if err := c.ShouldBindJSON(&emailReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	ctx := context.Background()
	searchText := emailReq.Email

	iter := fb.Collection("Users").
		Where("email", ">=", searchText).
		Where("email", "<=", searchText+"\uf8ff").
		Documents(ctx)

	var userResponses []dto.UserResponse
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var user model.User
		if err := doc.DataTo(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user data"})
			return
		}

		userResp := dto.UserResponse{
			UserID:    user.UserID,
			Name:      user.Name,
			Email:     user.Email,
			Profile:   user.Profile,
			Role:      user.Role,
			IsVerify:  user.Verify,
			IsActive:  user.Active,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		}

		userResponses = append(userResponses, userResp)
	}

	if userResponses == nil {
		userResponses = []dto.UserResponse{}
	}

	c.JSON(http.StatusOK, userResponses)
}

func UpdateProfileUser(c *gin.Context, firestoreClient *firestore.Client) {
	userId := c.MustGet("userId").(string)

	var updateProfile dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&updateProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate if there's anything to update
	if updateProfile.Name == "" && updateProfile.Password == "" && updateProfile.Profile == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data to update"})
		return
	}

	// Validate input data
	if updateProfile.Name != "" {
		updateProfile.Name = strings.TrimSpace(updateProfile.Name)
		if len(updateProfile.Name) < 2 || len(updateProfile.Name) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be between 2 and 100 characters"})
			return
		}
	}

	if updateProfile.Profile != "" {
		updateProfile.Profile = strings.TrimSpace(updateProfile.Profile)
		if len(updateProfile.Profile) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Profile description must not exceed 500 characters"})
			return
		}
	}

	ctx := context.Background()

	// Reference to the user document
	userDocRef := firestoreClient.Collection("Users").Doc(userId)

	// Build update map efficiently
	updateMap := make(map[string]interface{})

	// Only add non-empty fields
	if updateProfile.Name != "" {
		updateMap["name"] = updateProfile.Name
	}
	if updateProfile.Profile != "" {
		updateMap["profile"] = updateProfile.Profile
	}

	// Handle password hashing if password is provided
	if updateProfile.Password != "" {
		// Hash password synchronously
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateProfile.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}
		updateMap["password"] = string(hashedPassword)
	}

	// Add updated timestamp
	updateMap["updatedat"] = time.Now()

	// Start Firestore transaction
	err := firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Check if user exists before updating
		userDoc, err := tx.Get(userDocRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return errors.New("user not found")
			}
			return errors.New("failed to retrieve user")
		}

		// Verify document exists and has data
		if !userDoc.Exists() {
			return errors.New("user not found")
		}

		// Create firestore updates only for fields that are being updated
		var updates []firestore.Update
		for field, value := range updateMap {
			updates = append(updates, firestore.Update{
				Path:  field,
				Value: value,
			})
		}

		// Update user profile in transaction
		return tx.Update(userDocRef, updates)
	})

	// Handle transaction errors
	if err != nil {
		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		case "failed to process password":
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		case "failed to retrieve user":
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		}
		return
	}

	// Prepare response data (without sensitive information)
	responseData := gin.H{
		"message": "Profile updated successfully",
		"userid":  userId,
	}

	c.JSON(http.StatusOK, responseData)
}

func DeleteUser(c *gin.Context, firestoreClient *firestore.Client) {
	userId := c.MustGet("userId").(string)
	ctx := context.Background()

	// Use channels for concurrent checking
	type checkResult struct {
		hasAssociations bool
		err             error
	}

	checkChan := make(chan checkResult, 1)

	// Check associations in Firestore concurrently
	go func() {
		defer func() {
			if r := recover(); r != nil {
				checkChan <- checkResult{hasAssociations: false, err: fmt.Errorf("panic in goroutine: %v", r)}
			}
		}()

		// Check Boards collection (where userId is creator or member)
		boardsQuery := firestoreClient.Collection("Boards").Where("userid", "==", userId).Limit(1)
		boardsSnapshot, err := boardsQuery.Documents(ctx).GetAll()
		if err != nil {
			checkChan <- checkResult{hasAssociations: false, err: fmt.Errorf("failed to check Boards: %w", err)}
			return
		}

		if len(boardsSnapshot) > 0 {
			checkChan <- checkResult{hasAssociations: true, err: nil}
			return
		}

		// Check BoardUser collection
		boardUserQuery := firestoreClient.Collection("BoardUser").Where("userid", "==", userId).Limit(1)
		boardUserSnapshot, err := boardUserQuery.Documents(ctx).GetAll()
		if err != nil {
			checkChan <- checkResult{hasAssociations: false, err: fmt.Errorf("failed to check BoardUser: %w", err)}
			return
		}

		if len(boardUserSnapshot) > 0 {
			checkChan <- checkResult{hasAssociations: true, err: nil}
			return
		}

		// Check Tasks collection
		tasksQuery := firestoreClient.Collection("Tasks").Where("userid", "==", userId).Limit(1)
		tasksSnapshot, err := tasksQuery.Documents(ctx).GetAll()
		if err != nil {
			checkChan <- checkResult{hasAssociations: false, err: fmt.Errorf("failed to check Tasks: %w", err)}
			return
		}

		hasAssociations := len(tasksSnapshot) > 0
		checkChan <- checkResult{hasAssociations: hasAssociations, err: nil}
	}()

	// Get result from goroutine
	result := <-checkChan
	if result.err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user associations"})
		return
	}

	// Get user document reference
	userDocRef := firestoreClient.Collection("Users").Doc(userId)

	if result.hasAssociations {
		// Deactivate user by updating active field
		_, err := userDocRef.Update(ctx, []firestore.Update{
			{Path: "active", Value: "2"}, // หรือ false หากเป็น boolean
		})
		if err != nil {
			// Check if document doesn't exist
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
	} else {
		// Delete user document
		_, err := userDocRef.Delete(ctx)
		if err != nil {
			// Check if document doesn't exist
			if status.Code(err) == codes.NotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
	}
}
