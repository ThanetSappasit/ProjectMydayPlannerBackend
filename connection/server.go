package connection

import (
	"log"
	auth "myapp/controller/auth"
	board "myapp/controller/board"
	task "myapp/controller/task"
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
	auth.CaptchaController(router, fb)
	auth.SignUpGetEmailController(router, fb)

	user.UserController(router, fb)

	board.CreateBoardController(router, fb)

	task.CreateTaskController(router, fb)

	router.Run()
}
