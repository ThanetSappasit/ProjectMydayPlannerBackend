package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AccessTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Request.Header.Get("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header is missing"})
			return
		}

		tokenString := strings.Replace(header, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบว่า token ใช้ algorithm HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			hmacSampleSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
			return hmacSampleSecret, nil
		})

		if err != nil {
			c.AbortWithStatusJSON(403, gin.H{"error": "Token is expired or invalid: " + err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// เก็บ claims ทั้งหมด
			c.Set("claims", claims)

			// ดึงค่า userID เป็น string
			if userID, ok := claims["userId"].(string); ok {
				c.Set("userId", userID)
			} else {
				c.AbortWithStatusJSON(401, gin.H{"error": "Invalid userId in token claims"})
				return
			}

			c.Next()
		} else {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token claims"})
			return
		}
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsValue, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Claims not found"})
			return
		}

		claims, ok := claimsValue.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims format"})
			return
		}

		// แก้ไขให้ตรวจสอบ field "Role" แทน "role" เพื่อให้สอดคล้องกับการสร้าง token
		role, ok := claims["Role"].(string)
		if !ok || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		c.Next()
	}
}

func RefreshTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// รับ refresh token จาก Header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Refresh token is missing"})
			c.Abort()
			return
		}

		// ตรวจสอบรูปแบบของ token
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		refreshToken := bearerToken[1]

		// Decode และตรวจสอบ token โดยใช้ JWT key
		hmacSampleSecret := []byte(os.Getenv("JWT_REFRESH_SECRET_KEY"))
		token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบว่า token ใช้ algorithm HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return hmacSampleSecret, nil
		})

		if err != nil {
			c.JSON(403, gin.H{"error": "Invalid refresh token: " + err.Error()})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// ตรวจสอบว่า token หมดอายุหรือไม่ (JWT library จะจัดการ exp claim อัตโนมัติ)
			// ไม่จำเป็นต้องตรวจสอบ expiresAt แยกต่างหาก

			// ดึง userID จาก claims เป็น string
			var userID string
			var found bool

			// ตรวจสอบ field "UserID" (ให้สอดคล้องกับการสร้าง refresh token)
			if userID, found = claims["userId"].(string); !found {
				c.JSON(401, gin.H{"error": "Invalid token claims: UserID not found"})
				c.Abort()
				return
			}

			// เก็บข้อมูลที่จำเป็นไว้ใน context เพื่อให้ handler สามารถเข้าถึงได้
			c.Set("userID", userID)
			c.Set("refreshToken", refreshToken)

			// ดำเนินการต่อไปยัง handler
			c.Next()
		} else {
			c.JSON(401, gin.H{"error": "Invalid refresh token claims"})
			c.Abort()
			return
		}
	}
}
