package boards

import (
	"backend/dto"
	"backend/model"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func BoardsController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/boards")
	{
		routes.POST("/getboards", func(c *gin.Context) {
			GetBoards(c, db, firestoreClient)
		})
		routes.POST("/createboard", func(c *gin.Context) {
			CreateBoards(c, db, firestoreClient)
		})
	}
}

func CreateBoards(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// Create a new board
	var board dto.CreateBoardRequest
	if err := c.ShouldBindJSON(&board); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user model.User
	if err := db.Where("email = ?", board.CreatedBy).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Start a transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Using GORM models instead of raw SQL
	newBoard := model.Board{
		BoardName: board.BoardName,
		CreatedBy: user.UserID,
		CreatedAt: time.Now(),
	}

	// Create the board
	if err := tx.Create(&newBoard).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create board"})
		return
	}

	// If it's a group board (or for all boards based on your logic), add the creator as a member
	// Assuming Is_group is a string representation of a boolean or number
	if board.Is_group == "1" || board.Is_group == "true" {
		boardUser := model.BoardUser{
			BoardID: newBoard.BoardID, // Assuming ID is the primary key field in your Board model
			UserID:  user.UserID,
			AddedAt: time.Now(),
		}

		if err := tx.Create(&boardUser).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to board"})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Board created successfully",
		"boardID": newBoard.BoardID,
	})
}

func GetBoards(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// Get all boards for a user
	var boards dto.GetBoardsRequest
	if err := c.ShouldBindJSON(&boards); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user model.User
	if err := db.Where("email = ?", boards.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	userID := user.UserID
	if boards.Group == "1" || boards.Group == "true" {
		// Fetch boards where the user is a member (shared boards)
		var sharedBoards []model.Board

		if err := db.Joins("JOIN board_user ON board.board_id = board_user.board_id").
			Where("board_user.user_id = ?", userID).
			Preload("Creator"). // Load the creator user data
			Find(&sharedBoards).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared boards"})
			return
		}

		c.JSON(http.StatusOK, sharedBoards)
		return
	} else {
		// Fetch boards created by the user but not shared with others
		var personalBoards []model.Board

		if err := db.Where("create_by = ?", userID).
			Where("board_id NOT IN (SELECT board_id FROM board_user)").
			Preload("Creator"). // Load the creator user data
			Find(&personalBoards).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch personal boards"})
			return
		}

		c.JSON(http.StatusOK, personalBoards)
		return
	}
}
