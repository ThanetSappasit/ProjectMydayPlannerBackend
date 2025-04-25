package auth

import (
	"backend/dto"
	"backend/model"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func UserSignController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/auth")
	{
		routes.POST("/signin", func(c *gin.Context) {
			SignIn(c, db, firestoreClient)
		})
		routes.POST("/signout", func(c *gin.Context) {
			SignOut(c, db, firestoreClient)
		})
		routes.POST("/googlesignin", func(c *gin.Context) {
			googleSignIn(c, db, firestoreClient)
		})
	}
}

func SignIn(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
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

	role := "user"
	if user.Role == "admin" {
		role = user.Role
	}
	// เตรียมข้อมูลสำหรับบันทึกใน Firebase
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

func SignOut(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
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

	// อัปเดตเฉพาะฟิลด์ login เป็น 0 ใน Firebase โดยไม่กระทบข้อมูลอื่น
	_, err := firestoreClient.Collection("usersLogin").Doc(user.Email).Update(c, []firestore.Update{
		{Path: "login", Value: 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update login status"})
		return
	}

	// ส่งการตอบกลับที่สำเร็จ
	c.JSON(200, gin.H{
		"message": "Signout successful",
		"email":   signoutRequest.Email,
	})

}

func googleSignIn(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var googleSignInRequest dto.GoogleSignInRequest
	if err := c.ShouldBindJSON(&googleSignInRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// ค้นหาผู้ใช้ในฐานข้อมูล
	var user model.User
	result := db.Where("email = ?", googleSignInRequest.Email).First(&user)

	var firebaseData map[string]interface{}
	role := "user"
	if user.Role == "admin" {
		role = user.Role
	}

	// เงื่อนไขที่ 1: ไม่เจอข้อมูลในระบบ
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// กรณีไม่พบผู้ใช้ในระบบ
			isActive := "1"
			isVerify := "0"
			createAt := time.Now().UTC()

			var sql = `
				INSERT INTO user (name, email, profile, hashed_password, role, is_active, is_verify, create_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`
			// ต้องhash - ก่อนไหม ค่อยเอาไปเช็คที่หน้าบ้าน??
			hashedPasswordValue := "-"

			result := db.Exec(sql,
				googleSignInRequest.Name,
				googleSignInRequest.Email,
				googleSignInRequest.Profile,
				hashedPasswordValue,
				role,
				isActive,
				isVerify,
				createAt)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
			firebaseData = map[string]interface{}{
				"email":  googleSignInRequest.Email,
				"active": isActive,
				"verify": isVerify,
				"login":  0,
				"role":   role,
			}

			// บันทึกหรืออัปเดตข้อมูลใน Firebase collection "usersLogin"
			_, err := firestoreClient.Collection("usersLogin").Doc(googleSignInRequest.Email).Set(c, firebaseData, firestore.MergeAll)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to update Firebase user data: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "not_found",
				"message": "User not found in system but registered in Firebase",
				"email":   googleSignInRequest.Email,
			})
		} else {
			// กรณีเกิด error อื่นๆ
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}

	// เงื่อนไขที่ 2: เจอข้อมูลในระบบ
	// เตรียมข้อมูลสำหรับบันทึกใน Firebase
	isActive := "1"
	isVerify := "1"

	firebaseData = map[string]interface{}{
		"email":  googleSignInRequest.Email,
		"active": isActive,
		"verify": isVerify,
		"login":  1,
		"role":   role,
	}

	// บันทึกหรืออัปเดตข้อมูลใน Firebase collection "usersLogin"
	_, err := firestoreClient.Collection("usersLogin").Doc(googleSignInRequest.Email).Set(c, firebaseData, firestore.MergeAll)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update Firebase user data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login successful",
		"user": gin.H{
			"id":    user.UserID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}
