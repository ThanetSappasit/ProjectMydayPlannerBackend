package connection

import (
	"log"
	auth "myapp/controller/auth"
	user "myapp/controller/user"

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

	auth.SignInController(router, fb)
	auth.SignUpController(router, fb)
	auth.OTPController(router, fb)
	auth.SignUpGetEmailController(router, fb)

	user.UserController(router, fb)

	router.Run()
}
