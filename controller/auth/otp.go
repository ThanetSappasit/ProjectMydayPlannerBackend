package auth

import (
	"context"
	"fmt"
	"myapp/dto"
	"myapp/model"
	"myapp/services"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func OTPController(router *gin.Engine, firestoreClient *firestore.Client) {
	routes := router.Group("/auth")
	{
		routes.POST("/IdentityOTP", func(c *gin.Context) {
			IdentityOTP(c, firestoreClient)
		})
		routes.POST("/resetpasswordOTP", func(c *gin.Context) {
			ResetpasswordOTP(c, firestoreClient)
		})
		routes.POST("/sendemail", func(c *gin.Context) {
			Sendemail(c, firestoreClient)
		})
		routes.POST("/resendotp", func(c *gin.Context) {
			ResendOTP(c, firestoreClient)
		})
		routes.POST("/verifyOTP", func(c *gin.Context) {
			VerifyOTP(c, firestoreClient)
		})
	}
}

func IdentityOTP(c *gin.Context, firestoreClient *firestore.Client) {
	var req dto.IdentityOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่ในระบบหรือไม่
	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if !exists {
		c.JSON(400, gin.H{"error": "Email is not already registered"})
		return
	}

	// ตรวจสอบว่าอีเมลถูกบล็อกหรือไม่
	blocked, err := services.IsEmailBlocked(c, firestoreClient, req.Email, "verify")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check email status"})
		return
	}
	if blocked {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Please try again later."})
		return
	}

	// ตรวจสอบจำนวนครั้งที่ขอ OTP และบล็อกถ้าเกินกำหนด
	shouldBlock, err := services.CheckAndBlockIfNeeded(c, firestoreClient, req.Email, "verify")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check OTP request count"})
		return
	}
	if shouldBlock {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Your email has been blocked temporarily."})
		return
	}

	ref := services.GenerateREF(10)

	c.JSON(200, gin.H{
		"message": "OTP has been sent to your email identity",
		"ref":     ref,
	})
}

func ResetpasswordOTP(c *gin.Context, firestoreClient *firestore.Client) {
	var req dto.IdentityOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่ในระบบหรือไม่
	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if !exists {
		c.JSON(400, gin.H{"error": "Email is not already registered"})
		return
	}

	// ตรวจสอบว่าอีเมลถูกบล็อกหรือไม่
	blocked, err := services.IsEmailBlocked(c, firestoreClient, req.Email, "resetpassword")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check email status"})
		return
	}
	if blocked {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Please try again later."})
		return
	}

	// ตรวจสอบจำนวนครั้งที่ขอ OTP และบล็อกถ้าเกินกำหนด
	shouldBlock, err := services.CheckAndBlockIfNeeded(c, firestoreClient, req.Email, "resetpassword")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check OTP request count"})
		return
	}
	if shouldBlock {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Your email has been blocked temporarily."})
		return
	}

	ref := services.GenerateREF(10)

	c.JSON(200, gin.H{
		"message": "OTP has been sent to your email resetpassword",
		"ref":     ref,
	})
}

func Sendemail(c *gin.Context, firestoreClient *firestore.Client) {
	var req dto.SendemailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่ในระบบหรือไม่
	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if !exists {
		c.JSON(400, gin.H{"error": "Email is not already registered"})
		return
	}

	// สร้าง OTP และ REF
	otp, err := services.GenerateOTP(6)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// สร้างเนื้อหาอีเมล
	emailContent := services.GenerateEmailContent(otp, req.Reference)

	var recordemail string
	var recordfirebase string
	// ส่งอีเมล

	switch req.Record {
	case "1":
		recordemail = "รหัส OTP สำหรับยืนยันตัวตนบัญชีอีเมล"
		recordfirebase = "verify"
	case "2":
		recordemail = "รหัส OTP สำหรับรีเซ็ตรหัสผ่าน"
		recordfirebase = "resetpassword"
	}
	err = services.SendingEmail(req.Email, recordemail, emailContent)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send email: " + err.Error()})
		return
	}

	// บันทึกข้อมูล OTP ลงใน Firebase
	err = services.SaveOTPRecord(c, firestoreClient, req.Email, otp, req.Reference, recordfirebase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to save OTP record: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("OTP %s has been sent to your email", recordfirebase),
	})
}

func ResendOTP(c *gin.Context, firestoreClient *firestore.Client) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่ในระบบหรือไม่
	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if !exists {
		c.JSON(400, gin.H{"error": "Email is not already registered"})
		return
	}

	var recordemail string
	var recordfirebase string
	// ส่งอีเมล

	switch req.Record {
	case "1":
		recordemail = "รหัส OTP สำหรับยืนยันตัวตนบัญชีอีเมล"
		recordfirebase = "verify"
	case "2":
		recordemail = "รหัส OTP สำหรับรีเซ็ตรหัสผ่าน"
		recordfirebase = "resetpassword"
	}

	// ตรวจสอบว่าอีเมลถูกบล็อกหรือไม่
	blocked, err := services.IsEmailBlocked(c, firestoreClient, req.Email, recordfirebase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check email status"})
		return
	}
	if blocked {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Please try again later."})
		return
	}

	// ตรวจสอบจำนวนครั้งที่ขอ OTP และบล็อกถ้าเกินกำหนด
	shouldBlock, err := services.CheckAndBlockIfNeeded(c, firestoreClient, req.Email, recordfirebase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check OTP request count"})
		return
	}
	if shouldBlock {
		c.JSON(403, gin.H{"error": "Too many OTP requests. Your email has been blocked temporarily."})
		return
	}

	// สร้าง OTP และ REF ใหม่
	otp, err := services.GenerateOTP(6)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate OTP"})
		return
	}

	ref := services.GenerateREF(10)

	// สร้างเนื้อหาอีเมล
	emailContent := services.GenerateEmailContent(otp, ref)

	err = services.SendingEmail(req.Email, recordemail, emailContent)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send email: " + err.Error()})
		return
	}

	err = services.SaveOTPRecord(c, firestoreClient, req.Email, otp, ref, recordfirebase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to save OTP record: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "OTP has been sent to your email",
		"ref":     ref,
	})
}

func VerifyOTP(c *gin.Context, firestoreClient *firestore.Client) {
	// รับข้อมูลจาก request
	var verifyRequest dto.VerifyRequest
	if err := c.ShouldBindJSON(&verifyRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่ามีผู้ใช้ในระบบไหม
	ctx := context.Background()
	exists, err := services.UserExist(ctx, firestoreClient, verifyRequest.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing email"})
		return
	}
	if !exists {
		c.JSON(400, gin.H{"error": "Email is not already registered"})
		return
	}

	// ตรวจสอบว่า input ไม่เป็นค่าว่าง
	if verifyRequest.Record == "" || verifyRequest.Reference == "" || verifyRequest.OTP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record, Reference and OTP are required"})
		return
	}

	var recordfirebase string
	// ส่งอีเมล

	switch verifyRequest.Record {
	case "1":
		recordfirebase = "verify"
	case "2":
		recordfirebase = "resetpassword"
	}

	// ดึงข้อมูล OTP จาก Firebase โดยใช้ reference
	mainDoc := firestoreClient.Collection("OTPRecords").Doc(verifyRequest.Email)
	subCollection := mainDoc.Collection(fmt.Sprintf("OTPRecords_%s", recordfirebase))
	docRef := subCollection.Doc(verifyRequest.Reference)

	docSnap, err := docRef.Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid reference code"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve OTP record"})
			fmt.Printf("Firestore error: %v", err) // บันทึก error ที่เกิดขึ้นโดยไม่แสดงให้ user เห็น
		}
		return
	}

	// แปลงข้อมูลจาก Firestore เป็นโครงสร้างข้อมูลที่ใช้งานได้
	var otpRecord model.OTPRecord

	if err := docSnap.DataTo(&otpRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OTP record"})
		fmt.Printf("Data parsing error: %v", err)
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

	// อัปเดตสถานะ OTP ว่าถูกใช้แล้ว
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "is_used", Value: "1"},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OTP status"})
		fmt.Printf("Firestore update error: %v", err)
		return
	}

	// ตัวแปรสำหรับเก็บข้อมูลที่จะส่งกลับ
	responseData := gin.H{
		"message": "OTP verified successfully",
	}

	// เงื่อนไขพิเศษสำหรับ OTPRecords_verify
	if recordfirebase == "verify" {
		docRef, err = services.GetUserExist(ctx, firestoreClient, verifyRequest.Email)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		// อัปเดทฟิลด์ verify
		_, err = docRef.Update(ctx, []firestore.Update{
			{Path: "verify", Value: "1"},
			{Path: "login", Value: "1"},
		})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to update verify field"})
			return
		}

		docSnap, err := services.GetUserData(ctx, firestoreClient, verifyRequest.Email)
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

		// สร้าง tokens
		accessToken, err := services.CreateAccessToken(user.UserID, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
			return
		}

		refreshToken, err := services.CreateRefreshToken(user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refresh token"})
			return
		}

		// แฮช refresh token
		hashedRefreshToken, err := services.HashRefreshToken(refreshToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash refresh token"})
			return
		}

		// กำหนดค่าเวลาสำหรับ token
		now := time.Now()
		expiresAt := now.Add(7 * 24 * time.Hour).Unix() // เปลี่ยนเป็น 7 วันแทน 2 นาที
		issuedAt := now.Unix()

		// สร้างข้อมูล refresh token
		refreshTokenData := model.TokenResponse{
			UserID:       user.UserID,
			RefreshToken: hashedRefreshToken,
			CreatedAt:    issuedAt,
			Revoked:      false,
			ExpiresIn:    expiresAt - issuedAt,
		}

		// บันทึก refresh token ใน Firestore
		if _, err := firestoreClient.Collection("refreshTokens").Doc(user.UserID).Set(ctx, refreshTokenData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store refresh token"})
			return
		}

		// เพิ่ม token ในข้อมูลที่จะส่งกลับ
		responseData["accessToken"] = accessToken
		responseData["refreshToken"] = refreshToken
	}

	// ส่งข้อมูลตอบกลับ (ทำเพียงครั้งเดียว)
	c.JSON(http.StatusOK, responseData)
}
