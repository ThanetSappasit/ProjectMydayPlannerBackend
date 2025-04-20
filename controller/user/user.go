package user

import (
	"backend/model"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func UserController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/user")
	{
		routes.GET("/getalluser", func(c *gin.Context) {
			GetAllUser(c, db, firestoreClient)
		})
		// routes.POST("/getemail", func(c *gin.Context) {
		// 	GetUserByEmail(c, db, firestoreClient)
		// })
		// routes.POST("/createaccount", func(c *gin.Context) {
		// 	CreateAccUser(c, db, firestoreClient)
		// })
		// routes.POST("/resetpassword", func(c *gin.Context) {
		// 	ResetPassword(c, db, firestoreClient)
		// })
		// routes.DELETE("/deleteuser", func(c *gin.Context) {
		// 	DeleteUser(c, db, firestoreClient)
		// })
		// routes.PUT("/updateprofile", func(c *gin.Context) {
		// 	UpdateProfileUser(c, db, firestoreClient)
		// })
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
