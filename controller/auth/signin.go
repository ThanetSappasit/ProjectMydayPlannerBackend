package auth

import (
	"backend/dto"
	"backend/model"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func UserSignController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/auth")
	{
		routes.POST("/signin", func(c *gin.Context) {
			Signin(c, db, firestoreClient)
		})
		routes.POST("/signout", func(c *gin.Context) {
			Signout(c, db, firestoreClient)
		})
		// routes.POST("/verifyOTP", func(c *gin.Context) {
		// 	VerifyOTP(c, db, firestoreClient)
		// })
	}
}

func Signin(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var signinRequest dto.SigninRequest

	if err := c.ShouldBindJSON(&signinRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if signinRequest.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	// ค้นหาผู้ใช้ในฐานข้อมูล
	var user model.User
	result := db.Where("email = ?", signinRequest.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}
	// ตรวจสอบรหัสผ่าน
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(signinRequest.HashedPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// ตรวจสอบว่าผู้ใช้มีการยืนยันและเปิดใช้งานหรือไม่
	if user.IsActive != "1" {
		if condition := user.IsActive == "0"; condition {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is not active", "status": "0"})
			return
		} else if condition := user.IsActive == "2"; condition {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is delete", "status": "2"})
			return
		}
	}

	if user.IsVerify != "1" {
		c.JSON(http.StatusForbidden, gin.H{"error": "User account is not verified"})
		return
	}

	// เตรียมข้อมูลสำหรับบันทึกใน Firebase
	role := "user"
	isActive := "1"
	isVerify := "1"

	firebaseData := map[string]interface{}{
		"email":  signinRequest.Email,
		"active": isActive,
		"verify": isVerify,
		"login":  1,
		"role":   role,
	}

	// บันทึกหรืออัปเดตข้อมูลใน Firebase collection "usersLogin"
	_, err = firestoreClient.Collection("usersLogin").Doc(signinRequest.Email).Set(c, firebaseData, firestore.MergeAll)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update Firebase user data: " + err.Error()})
		return
	}

	// ส่งการตอบกลับที่สำเร็จ
	c.JSON(200, gin.H{
		"message": "Signin successful",
		"email":   signinRequest.Email,
		"role":    role,
	})
}

func Signout(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var signoutRequest dto.SignoutRequest

	if err := c.ShouldBindJSON(&signoutRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if signoutRequest.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	// ค้นหาผู้ใช้ในฐานข้อมูล
	var user model.User
	result := db.Where("email = ?", signoutRequest.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}

	// ลบข้อมูลใน Firebase collection "usersLogin"
	_, err := firestoreClient.Collection("usersLogin").Doc(signoutRequest.Email).Delete(c)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete Firebase user data: " + err.Error()})
		return
	}

	// ส่งการตอบกลับที่สำเร็จ
	c.JSON(200, gin.H{
		"message": "Signout successful",
		"email":   signoutRequest.Email,
	})

}
