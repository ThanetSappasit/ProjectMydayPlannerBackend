package attachments

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Maximum file size: 64MB (same as Node.js implementation)
const maxFileSize = 64 * 1024 * 1024

func AttachmentsController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/attachments")
	{
		routes.POST("/fileupload", func(c *gin.Context) {
			FileUpload(c, db, firestoreClient)
		})
	}
}

func FileUpload(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// Set maximum file size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	// Get file from the request
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
		return
	}
	defer file.Close()

	// Create a unique filename (same logic as Node.js implementation)
	uniqueSuffix := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))
	fileExt := filepath.Ext(fileHeader.Filename)
	if fileExt != "" {
		fileExt = fileExt[1:] // Remove the dot from extension
	} else {
		// Extract extension from the original filename
		parts := strings.Split(fileHeader.Filename, ".")
		if len(parts) > 1 {
			fileExt = parts[len(parts)-1]
		}
	}
	filename := uniqueSuffix + "." + fileExt

	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join(".", "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// Save the file
	dst := filepath.Join(uploadsDir, filename)
	if err := c.SaveUploadedFile(fileHeader, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Return the file path, same as Node.js implementation
	c.JSON(http.StatusOK, gin.H{"filename": "/uploads/" + filename})
}
