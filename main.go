package main

import (
	"myapp/connection"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	// go scheduler.StartScheduler()
	connection.StartServer()
}
