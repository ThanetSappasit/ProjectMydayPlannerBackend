package user

import (
	"backend/model"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AllDataUserController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/user")
	{
		routes.GET("/emailalldatauser/:userID", func(c *gin.Context) {
			GetUserAllData(c, db, firestoreClient)
		})
	}
}

func GetUserAllData(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// Get userID from URL params
	userID := c.Param("userID")

	// Check if userID is provided
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UserID is required"})
		return
	}

	// Create a struct to hold the complete user data result
	type CompleteUserData struct {
		User      model.User
		Boards    []model.Board
		BoardUser []model.BoardUser
		Tasks     []model.Task
	}

	var result CompleteUserData

	// First, get the user by userID
	if err := db.Where("user_id = ?", userID).First(&result.User).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Use wait group to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(3) // We have 3 concurrent operations

	// Channel for error handling
	errChan := make(chan error, 3)

	// Get boards created by this user (concurrently)
	go func() {
		defer wg.Done()
		if err := db.Where("create_by = ?", userID).Find(&result.Boards).Error; err != nil {
			errChan <- fmt.Errorf("error fetching boards: %w", err)
		}
	}()

	// Get board memberships for this user (concurrently)
	go func() {
		defer wg.Done()
		if err := db.Where("user_id = ?", userID).Find(&result.BoardUser).Error; err != nil {
			errChan <- fmt.Errorf("error fetching board memberships: %w", err)
		}
	}()

	// Get tasks created by or assigned to this user (concurrently)
	go func() {
		defer wg.Done()
		if err := db.Where("create_by = ? OR assigned_to = ?", userID, userID).
			Find(&result.Tasks).Error; err != nil {
			errChan <- fmt.Errorf("error fetching tasks: %w", err)
		}
	}()

	// Close error channel when all goroutines complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for any errors
	for err := range errChan {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Return all data
	c.JSON(http.StatusOK, gin.H{
		"message": "User data retrieved successfully",
		"data":    result,
	})
}
