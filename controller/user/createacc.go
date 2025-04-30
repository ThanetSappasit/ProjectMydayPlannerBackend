package user

import (
	"backend/dto"
	"backend/model"
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func CreateUserController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/user")
	{
		routes.POST("/createaccount", func(c *gin.Context) {
			CreateAccUser(c, db, firestoreClient)
		})

	}
}

// ฟังก์ชันสำหรับตรวจสอบอีเมลที่มีอยู่แล้ว
func checkExistingEmail(db *gorm.DB, email string) (*model.User, error) {
	var user model.User
	result := db.Where("email = ?", email).First(&user)
	if result.RowsAffected > 0 {
		return &user, fmt.Errorf("email already exists")
	}
	return nil, nil
}

// ฟังก์ชันสำหรับสร้างข้อมูลผู้ใช้ในฐานข้อมูล
func createUserInDB(db *gorm.DB, name, email, hashedPassword, profile, role, isActive, isVerify string) error {
	sql := `
		INSERT INTO user (name, email, profile, hashed_password, role, is_active, is_verify, create_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result := db.Exec(sql,
		name,
		email,
		profile,
		hashedPassword,
		role,
		isActive,
		isVerify,
		time.Now().UTC())

	return result.Error
}

// ฟังก์ชันสำหรับสร้างข้อมูลผู้ใช้ใน Firestore
func createUserInFirestore(firestoreClient *firestore.Client, email, isActive, isVerify, role string) error {
	firebasedata := map[string]interface{}{
		"email":  email,
		"active": isActive,
		"verify": isVerify,
		"login":  1,
		"role":   role,
	}

	_, err := firestoreClient.Collection("usersLogin").Doc(email).Set(context.Background(), firebasedata)
	return err
}

// ฟังก์ชันสำหรับแฮชรหัสผ่าน
func hashPassword(password string) (string, error) {
	if password == "" {
		return "-", nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// ฟังก์ชันหลักสำหรับสร้างผู้ใช้
func CreateAccUser(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var userRequest dto.CreateAccUserRequest
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// ตรวจสอบอีเมลที่มีอยู่แล้ว
	_, err := checkExistingEmail(db, userRequest.Email)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// กำหนดค่าเริ่มต้น
	role := "user"
	isActive := "1"
	isVerify := "0"
	profile := "none-url"

	// แฮชรหัสผ่าน
	hashedPasswordValue, err := hashPassword(userRequest.HashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// สร้างผู้ใช้ในฐานข้อมูล
	err = createUserInDB(db, userRequest.Name, userRequest.Email, hashedPasswordValue, profile, role, isActive, isVerify)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// สร้างผู้ใช้ใน Firestore
	err = createUserInFirestore(firestoreClient, userRequest.Email, isActive, isVerify, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user in Firestore"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "User created successfully"})
}
