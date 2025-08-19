package auth

import (
	"context"
	"errors"
	"myapp/dto"
	"myapp/model"
	"myapp/services"
	"net"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func SignUpController(router *gin.Engine, firestoreClient *firestore.Client) {
	router.POST("/auth/signup", func(c *gin.Context) {
		Signup(c, firestoreClient)
	})
}

func Signup(c *gin.Context, firestoreClient *firestore.Client) {
	var request dto.SignupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := isValidEmail(request.Email); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, request.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if exists {
		c.JSON(400, gin.H{"error": "Email is already registered"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}

	docid := uuid.New().String()

	newUser := model.User{
		UserID:    docid,
		Name:      request.Name,
		Email:     request.Email,
		Password:  string(hashedPassword),
		Profile:   "none-url",
		Role:      "user",
		Verify:    "0",
		Active:    "1",
		CreatedAt: time.Now(),
	}

	// ประกาศ docRef ก่อนใช้
	_, err = firestoreClient.Collection("Users").Doc(docid).Set(ctx, newUser)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(201, gin.H{
		"message": "User registered successfully",
		"docID":   docid,
	})
}

func isValidEmail(email string) error {
	// Check email format with regex
	const emailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	if !re.MatchString(email) {
		return errors.New("invalid email format")
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email structure")
	}
	domain := parts[1]

	// Check for MX records
	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		return errors.New("email domain does not have valid MX records")
	}

	return nil
}
