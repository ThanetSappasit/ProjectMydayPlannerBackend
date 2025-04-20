package main

import (
	"backend/connection"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	connection.StartServer()
}
