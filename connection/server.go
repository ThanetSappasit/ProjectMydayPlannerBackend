package connection

import (
	"log"
	controller "myapp/controller/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func StartServer() {
	router := gin.Default()

	fb, err := FBConnection()
	if err != nil {
		log.Fatalf("Failed to initialize Firestore client: %v", err)
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Api is running!"})
	})

	router.Use(cors.Default())

	controller.SignInController(router, fb)
	controller.SignUpController(router, fb)
	controller.OTPController(router, fb)

	router.Run()
}
