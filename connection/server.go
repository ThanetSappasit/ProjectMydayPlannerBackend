package connection

import (
	"backend/controller/admin"
	"backend/controller/attachments"
	"backend/controller/auth"
	"backend/controller/boards"
	"backend/controller/tasks"
	"backend/controller/user"
	"log"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	router := gin.Default()

	db, err := DBConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	firestoreClient, err := InitFirestoreClient()
	if err != nil {
		log.Fatalf("Failed to initialize Firestore client: %v", err)
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Api is running!"})
	})

	user.UserController(router, db, firestoreClient)
	auth.UserAuthController(router, db, firestoreClient)
	auth.UserSignController(router, db, firestoreClient)
	attachments.AttachmentsController(router, db, firestoreClient)
	admin.AdminEditController(router, db, firestoreClient)
	boards.BoardsController(router, db, firestoreClient)
	tasks.TasksController(router, db, firestoreClient)
	auth.CaptchaController(router, db, firestoreClient)

	router.Run()
}
