package auth

import (
	"context"
	"myapp/dto"
	"myapp/model"
	"myapp/services"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func SignInController(router *gin.Engine, firestoreClient *firestore.Client) {
	router.POST("/auth/signin", func(c *gin.Context) {
		Signin(c, firestoreClient)
	})
}

func Signin(c *gin.Context, firestoreClient *firestore.Client) {
	var request dto.SigninRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ตรวจสอบข้อมูลที่จำเป็น
	if request.Email == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	// ค้นหาผู้ใช้จากฐานข้อมูล
	ctx := context.Background()
	docSnap, err := services.GetUserData(ctx, firestoreClient, request.Email)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	// แปลงข้อมูลเป็น struct
	var user model.User
	if err := docSnap.DataTo(&user); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse user data"})
		return
	}

	// ตรวจสอบรหัสผ่าน
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// ตรวจสอบสถานะบัญชีผู้ใช้
	switch user.Active {
	case "0":
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is not active", "status": "0"})
		return
	case "2":
		c.JSON(http.StatusBadRequest, gin.H{"error": "User account is deleted", "status": "2"})
		return
	}

	// ตรวจสอบการยืนยันบัญชี
	if user.Verify != "1" {
		c.JSON(http.StatusForbidden, gin.H{"error": "User account is not verified"})
		return
	}

	// สร้าง tokens
	accessToken, err := services.CreateAccessToken(user.UserID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
		return
	}

	refreshToken, err := services.CreateRefreshToken(user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refresh token"})
		return
	}

	// แฮช refresh token
	hashedRefreshToken, err := services.HashRefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash refresh token"})
		return
	}

	// กำหนดค่าเวลาสำหรับ token
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour).Unix()
	issuedAt := now.Unix()

	// สร้างข้อมูล refresh token
	refreshTokenData := model.TokenResponse{
		UserID:       user.UserID,
		RefreshToken: hashedRefreshToken,
		CreatedAt:    issuedAt,
		Revoked:      false,
		ExpiresIn:    expiresAt - issuedAt,
	}

	// บันทึก refresh token ใน Firestore
	if _, err := firestoreClient.Collection("refreshTokens").Doc(user.UserID).Set(c, refreshTokenData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store refresh token ", "detail": err.Error()})
		return
	}

	// กำหนดบทบาทผู้ใช้
	role := "user"
	if user.Role == "admin" {
		role = user.Role
	}

	// อัปเดตข้อมูลการเข้าสู่ระบบใน Firestore
	loginData := map[string]interface{}{
		"email":     request.Email,
		"active":    user.Active,
		"verify":    user.Verify,
		"login":     1,
		"role":      role,
		"updatedat": now,
	}

	// บันทึกข้อมูลการเข้าสู่ระบบใน Firestore
	if _, err := firestoreClient.Collection("Users").Doc(user.UserID).Set(c, loginData, firestore.MergeAll); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update login status"})
		return
	}

	// ส่งผลลัพธ์กลับ
	c.JSON(http.StatusOK, gin.H{
		"message": "Login Successfully",
		"token": gin.H{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
		},
	})
}
