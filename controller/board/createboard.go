package board

import (
	"context"
	"encoding/base64"
	"myapp/dto"
	"myapp/middleware"
	"myapp/model"
	"myapp/services"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateBoardController(router *gin.Engine, firestoreClient *firestore.Client) {
	router.POST("/board", middleware.AccessTokenMiddleware(), func(c *gin.Context) {
		CreateBoard(c, firestoreClient)
	})
}

func CreateBoard(c *gin.Context, firestoreClient *firestore.Client) {
	userId := c.MustGet("userId").(string)
	var board dto.CreateBoardRequest
	if err := c.ShouldBindJSON(&board); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// ค้นหาผู้ใช้จากฐานข้อมูล
	ctx := context.Background()
	docSnap, err := services.GetUserDataByUserid(ctx, firestoreClient, userId)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	// แปลงข้อมูลเป็น struct
	var user model.User
	if err := docSnap.DataTo(&user); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse user data"})
		return
	}

	var grouptype string
	switch board.Is_group {
	case "0":
		grouptype = "private"
	case "1":
		grouptype = "group"
	default:
		grouptype = "unknown" // กันกรณีค่าไม่ใช่ 0 หรือ 1
	}

	boardid := uuid.New().String()

	newBoard := model.Board{
		BoardID:   boardid,
		BoardName: board.BoardName,
		CreatedBy: user.UserID,
		BoardType: grouptype,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var deepLink string
	if grouptype == "group" {
		// สร้าง share token
		expireAt := time.Now().Add(7 * 24 * time.Hour)
		params := url.Values{}
		params.Add("boardId", boardid)
		params.Add("expire", strconv.FormatInt(expireAt.Unix(), 10))

		encodedParams := base64.URLEncoding.EncodeToString([]byte(params.Encode()))

		deepLink = encodedParams
		// เพิ่ม deepLink ใน newBoard
		newBoard.DeepLink = deepLink
	}

	// บันทึกข้อมูล Board ลง Firestore
	_, err = firestoreClient.Collection("Boards").Doc(boardid).Set(ctx, newBoard)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create board"})
		return
	}

	// ส่งข้อมูลกลับไปยัง client
	response := gin.H{
		"boardId": boardid,
		"message": "Board created successfully",
	}

	// เพิ่ม deepLink ใน response หากเป็น group
	if grouptype == "group" {
		response["deep_link"] = deepLink
	}

	c.JSON(http.StatusCreated, response)
}
