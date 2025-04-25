package auth

import (
	"backend/dto"
	"backend/model"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func VerifyController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/auth")
	{
		routes.POST("/verifyOTP", func(c *gin.Context) {
			VerifyOTP(c, db, firestoreClient)
		})
	}
}

func VerifyOTP(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	// รับข้อมูลจาก request
	var verifyRequest dto.VerifyRequest

	if err := c.ShouldBindJSON(&verifyRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่า input ไม่เป็นค่าว่าง
	if verifyRequest.Record == "" || verifyRequest.Reference == "" || verifyRequest.OTP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record, Reference and OTP are required"})
		return
	}

	// ดึงข้อมูล OTP จาก Firebase โดยใช้ reference
	ctx := c.Request.Context() // ใช้ context จาก request แทนการส่ง c ไปโดยตรง
	collectionName := fmt.Sprintf("OTPRecords_%s", verifyRequest.Record)
	docRef := firestoreClient.Collection(collectionName).Doc(verifyRequest.Reference)
	docSnap, err := docRef.Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid reference code"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve OTP record"})
			log.Printf("Firestore error: %v", err) // บันทึก error ที่เกิดขึ้นโดยไม่แสดงให้ user เห็น
		}
		return
	}

	// แปลงข้อมูลจาก Firestore เป็นโครงสร้างข้อมูลที่ใช้งานได้
	var otpRecord model.OTPRecord

	if err := docSnap.DataTo(&otpRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OTP record"})
		log.Printf("Data parsing error: %v", err)
		return
	}

	// ตรวจสอบว่า OTP ถูกใช้ไปแล้วหรือไม่
	if otpRecord.Is_used == "1" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP has already been used"})
		return
	}

	// ตรวจสอบว่า OTP หมดอายุหรือยัง
	currentTime := time.Now()
	if currentTime.After(otpRecord.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP has expired"})
		return
	}

	// ตรวจสอบว่า OTP ตรงกันหรือไม่
	if otpRecord.OTP != verifyRequest.OTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OTP"})
		return
	}

	// เริ่ม transaction สำหรับการอัปเดต SQL database (ถ้าจำเป็น)
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		log.Printf("Transaction error: %v", err)
		return
	}

	// อัปเดตสถานะ OTP ว่าถูกใช้แล้ว
	_, err = firestoreClient.Collection(collectionName).Doc(verifyRequest.Reference).Update(ctx, []firestore.Update{
		{Path: "is_used", Value: "1"},
	})

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OTP status"})
		log.Printf("Firestore update error: %v", err)
		return
	}

	// เงื่อนไขพิเศษสำหรับ OTPRecords_verify
	if collectionName == "OTPRecords_verify" {
		// อัปเดตคอลัมน์ is_verify เป็น 1 ในตาราง user ของ SQL database
		result := tx.Model(&model.User{}).Where("email = ?", otpRecord.Email).Update("is_verify", 1)

		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user verification status"})
			log.Printf("DB update error: %v", result.Error)
			return
		}

		if result.RowsAffected == 0 {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// เฉพาะกรณี verify เท่านั้นที่จะบันทึกข้อมูลลงใน usersLogin
		// เตรียมข้อมูลสำหรับบันทึกหรืออัปเดตใน Firebase
		role := "user"
		isActive := "1"
		isVerify := "1"

		// บันทึกหรืออัปเดตข้อมูลใน Firebase collection "usersLogin"
		_, err = firestoreClient.Collection("usersLogin").Doc(otpRecord.Email).Set(ctx, map[string]interface{}{
			"email":      otpRecord.Email,
			"active":     isActive,
			"verify":     isVerify,
			"login":      1,
			"role":       role,
			"updated_at": time.Now(),
		}, firestore.MergeAll)

		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Firebase user data"})
			log.Printf("Firestore set error: %v", err)
			return
		}
	}

	// commit transaction หากทุกอย่างเรียบร้อย
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		log.Printf("Transaction commit error: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP verified successfully",
		"email":   otpRecord.Email,
	})
}
