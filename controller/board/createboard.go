package board

// import (
// 	"context"
// 	"myapp/dto"
// 	"myapp/middleware"
// 	"myapp/model"
// 	"myapp/services"
// 	"net/http"
// 	"time"

// 	"cloud.google.com/go/firestore"
// 	"github.com/gin-gonic/gin"
// )

// func CreateBoardController(router *gin.Engine, firestoreClient *firestore.Client) {
// 	router.POST("/board", middleware.AccessTokenMiddleware(), func(c *gin.Context) {
// 		CreateBoard(c, firestoreClient)
// 	})
// }

// func CreateBoard(c *gin.Context, firestoreClient *firestore.Client) {
// 	userId := c.MustGet("userId").(string)
// 	var board dto.CreateBoardRequest
// 	if err := c.ShouldBindJSON(&board); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}

// 	// ค้นหาผู้ใช้จากฐานข้อมูล
// 	ctx := context.Background()
// 	docSnap, err := services.GetUserDataByUserid(ctx, firestoreClient, userId)
// 	if err != nil {
// 		c.JSON(404, gin.H{"error": err.Error()})
// 		return
// 	}
// 	// แปลงข้อมูลเป็น struct
// 	var user model.User
// 	if err := docSnap.DataTo(&user); err != nil {
// 		c.JSON(500, gin.H{"error": "Failed to parse user data"})
// 		return
// 	}

// 	newBoard := model.Board{
// 		BoardName: board.BoardName,
// 		CreatedBy: user.UserID,
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// }
