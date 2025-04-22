package tasks

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TasksController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/tasks")
	{
		routes.POST("/createtasks", func(c *gin.Context) {
			CreateTasks(c, db, firestoreClient)
		})
		// routes.POST("/signout", func(c *gin.Context) {
		// 	SignOut(c, db, firestoreClient)
		// })
		// routes.POST("/googlesignin", func(c *gin.Context) {
		// 	googleSignIn(c, db, firestoreClient)
		// })
	}
}

func CreateTasks(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// Implement the function to create tasks
	c.JSON(200, gin.H{
		"message": "Create Tasks",
	})
}
