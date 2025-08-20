package auth

import (
	"context"
	"myapp/dto"
	"myapp/model"
	"myapp/services"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GoogleSignInController(router *gin.Engine, firestoreClient *firestore.Client) {
	router.POST("/auth/googlelogin", func(c *gin.Context) {
		GoogleSignIn(c, firestoreClient)
	})
}

func GoogleSignIn(c *gin.Context, firestoreClient *firestore.Client) {
	// รับและตรวจสอบข้อมูลจาก Request
	var req dto.GoogleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "กรุณาระบุข้อมูลให้ครบถ้วนและถูกต้อง",
		})
		return
	}

	// ตรวจสอบข้อมูลที่จำเป็น
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "กรุณาระบุอีเมล",
		})
		return
	}

	// ค้นหาผู้ใช้จากฐานข้อมูล
	ctx := context.Background()
	usersCollection := firestoreClient.Collection("Users")
	query := usersCollection.Where("email", "==", req.Email).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "เกิดข้อผิดพลาดในการค้นหาผู้ใช้",
			"error":   err.Error(),
		})
		return
	}

	var user model.User
	role := "user"
	isNewUser := false

	if len(docs) > 0 {
		// ✅ เจอผู้ใช้
		if err := docs[0].DataTo(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "ไม่สามารถอ่านข้อมูลผู้ใช้ได้",
				"error":   err.Error(),
			})
			return
		}

		// อัพเดท verify status ใน Users collection (ไม่ใช่ OTPRecords)
		if user.Verify == "0" {
			userDocRef := firestoreClient.Collection("Users").Doc(user.UserID)
			_, err = userDocRef.Update(ctx, []firestore.Update{
				{Path: "verify", Value: "1"},
			})
			if err != nil {
				// Log error แต่ไม่หยุดการทำงาน
				// สามารถใช้ logger ที่เหมาะสมได้
			} else {
				user.Verify = "1" // อัพเดทค่าใน memory
			}
		}

		// ตรวจสอบสถานะบัญชี
		switch user.Active {
		case "0":
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "บัญชีผู้ใช้ไม่ได้เปิดใช้งาน",
				"status":  "0",
			})
			return
		case "2":
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "บัญชีผู้ใช้ถูกลบแล้ว",
				"status":  "2",
			})
			return
		}

		// ถ้าผู้ใช้เป็น admin ให้คงสถานะไว้
		if user.Role == "admin" {
			role = "admin"
		}
	} else {
		// สร้างผู้ใช้ใหม่
		docid := uuid.New().String()
		user = model.User{
			UserID:    docid,
			Name:      req.Name,
			Email:     req.Email,
			Password:  "-",
			Profile:   "none-url",
			Role:      role,
			Verify:    "1", // Google user ถือว่า verified แล้ว
			Active:    "1",
			CreatedAt: time.Now(),
		}

		_, err = firestoreClient.Collection("Users").Doc(docid).Set(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "ไม่สามารถสร้างบัญชีผู้ใช้ได้",
				"error":   err.Error(),
			})
			return
		}
		isNewUser = true
	}

	// สร้าง tokens
	accessToken, err := services.CreateAccessToken(user.UserID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "ไม่สามารถสร้าง access token ได้",
			"error":   err.Error(),
		})
		return
	}

	refreshToken, err := services.CreateRefreshToken(user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "ไม่สามารถสร้าง refresh token ได้",
			"error":   err.Error(),
		})
		return
	}

	// แฮช refresh token
	hashedRefreshToken, err := services.HashRefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "ไม่สามารถสร้าง hashed token ได้",
			"error":   err.Error(),
		})
		return
	}

	// กำหนดค่าเวลาสำหรับ token
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour).Unix()
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "ไม่สามารถบันทึก refresh token ได้",
			"error":   err.Error(),
		})
		return
	}

	// กำหนด response message
	message := "เข้าสู่ระบบสำเร็จ"
	if isNewUser {
		message = "สร้างบัญชีและเข้าสู่ระบบสำเร็จ"
	}

	// ส่งผลลัพธ์กลับ
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"status": func() string {
			return "success"
		}(),
		"user": gin.H{
			"id":    user.UserID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
		"token": gin.H{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"expiresIn":    expiresAt - issuedAt,
		},
	})
}
