package services

import (
	"crypto/sha256"
	"myapp/model"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func CreateAccessToken(userID string, role string) (string, error) {
	hmacSampleSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
	claims := &model.AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mydayplanner",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(hmacSampleSecret)
}

func CreateRefreshToken(userID string) (string, error) {
	refreshTokenSecret := []byte(os.Getenv("JWT_REFRESH_SECRET_KEY"))
	claims := &model.AccessRefresh{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mydayplanner",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Longer-lived token (7 days)
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshTokenSecret)
}

func HashRefreshToken(token string) (string, error) {
	// ใช้ SHA-256 เพื่อลดความยาวของ token ก่อนส่งเข้า bcrypt
	// SHA-256 จะผลิต hash ที่มีความยาวแน่นอนเป็น 32 bytes (256 bits)
	hash := sha256.Sum256([]byte(token))

	// เอา hash ที่ได้จาก SHA-256 ที่มีความยาวแน่นอนแล้วไปเข้า bcrypt
	hashedToken, err := bcrypt.GenerateFromPassword(hash[:], bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedToken), nil
}
