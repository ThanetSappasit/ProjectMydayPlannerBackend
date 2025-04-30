package user

import (
	"backend/dto"
	"backend/model"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func UserController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/user")
	{
		routes.GET("/getalluser", func(c *gin.Context) {
			GetAllUser(c, db, firestoreClient)
		})
		routes.POST("/getemail", func(c *gin.Context) {
			GetUserByEmail(c, db, firestoreClient)
		})
		routes.POST("/resetpassword", func(c *gin.Context) {
			ResetPassword(c, db, firestoreClient)
		})
		routes.DELETE("/deleteuser", func(c *gin.Context) {
			DeleteUser(c, db, firestoreClient)
		})
		routes.PUT("/updateprofile", func(c *gin.Context) {
			UpdateProfileUser(c, db, firestoreClient)
		})
	}
}

func GetAllUser(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var user []model.User
	result := db.Find(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func GetUserByEmail(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var emailrequest dto.GetUserByEmail
	if err := c.ShouldBindJSON(&emailrequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if emailrequest.Email == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	var user model.User
	result := db.Where("email = ?", *emailrequest.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
	}
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var email dto.DeleteUserRequest
	if err := c.ShouldBindJSON(&email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	const checkSql = `
        SELECT DISTINCT *
        FROM user
        LEFT JOIN board ON user.user_id = board.create_by
        LEFT JOIN board_user ON user.user_id = board_user.user_id
        WHERE user.email = ?
            AND (board.board_id IS NOT NULL OR board_user.board_id IS NOT NULL)
    `
	var results []map[string]interface{}
	if err := db.Raw(checkSql, email.Email).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user associations"})
		return
	}

	//เช็คอีเมลก่อนว่ามีบอร์ดงานไหมถ้ามีไม่ให้ลบ ถ้าไม่มีลบเลย
	if len(results) > 0 {
		const updateSql = `
                UPDATE user
                SET is_active = "2"
                WHERE email = ?;`
		if err := db.Exec(updateSql, email.Email).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
	} else {
		const deleteSql = `
                DELETE FROM user
                WHERE email = ?;`
		if err := db.Exec(deleteSql, email.Email).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
	}
}

func UpdateProfileUser(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var updateProfile dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&updateProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var user model.User
	result := db.Where("email = ?", updateProfile.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}
	if updateProfile.ProfileData.HashedPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateProfile.ProfileData.HashedPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updateProfile.ProfileData.HashedPassword = string(hashedPassword)
	}

	updates := map[string]interface{}{
		"name":            updateProfile.ProfileData.Name,
		"hashed_password": updateProfile.ProfileData.HashedPassword,
		"profile":         updateProfile.ProfileData.Profile,
	}

	updateMap := make(map[string]interface{})
	for key, value := range updates {
		if value != "" {
			updateMap[key] = value
		}
	}

	if err := db.Model(&user).Updates(updateMap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func ResetPassword(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var resetPassword dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&resetPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user model.User
	result := db.Where("email = ?", resetPassword.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(resetPassword.HashedPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := db.Model(&user).Update("hashed_password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
