package user

import (
	"backend/dto"
	"backend/model"
	"fmt"
	"math/rand"
	"net/smtp"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func UserAuthController(router *gin.Engine, db *gorm.DB, firestoreClient *firestore.Client) {
	routes := router.Group("/user")
	{
		routes.POST("/requestOTP", func(c *gin.Context) {
			RequestOTP(c, db, firestoreClient)
		})
	}
}

func RequestOTP(c *gin.Context, db *gorm.DB, firestoreClient *firestore.Client) {
	var emailRequest dto.EmailOTPRequest
	if err := c.ShouldBindJSON(&emailRequest); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่ในระบบหรือไม่
	var user model.User
	result := db.Where("email = ?", emailRequest.Email).First(&user)
	if result.Error != nil {
		c.JSON(404, gin.H{"error": "Email not found"})
		return
	}

	// สร้าง OTP และ REF
	otp, err := generateOTP(6)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate OTP"})
		return
	}
	ref := generateREF(10)

	// สร้างเนื้อหาอีเมล
	emailContent := generateEmailContent(otp, ref)

	// ส่งอีเมล
	err = sendEmail(emailRequest.Email, "รหัส OTP สำหรับรีเซ็ตรหัสผ่าน", emailContent)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "OTP has been sent to your email",
		"otp":     otp,
		"ref":     ref,
	})
}

func LoadEmailConfig() (*model.EmailConfig, error) {
	// โหลด .env เฉพาะตอนรัน local (เมื่อ ENV "RENDER" ว่าง)
	if os.Getenv("RENDER") == "" {
		if err := godotenv.Load(); err != nil {
			fmt.Println("Warning: .env file not loaded, fallback to OS env vars")
		}
	}

	config := &model.EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}

	if config.Host == "" || config.Port == "" || config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("missing required SMTP environment variables")
	}

	fmt.Printf("SMTP Config: Host=%s, Port=%s, Username=%s\n", config.Host, config.Port, config.Username)
	return config, nil
}

func generateOTP(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}

	// In Go 1.20+, you don't need to call rand.Seed anymore
	var otp strings.Builder
	for i := 0; i < length; i++ {
		otp.WriteString(string(rune('0' + rand.Intn(10)))) // Random digit 0-9
	}

	return otp.String(), nil
}

func generateREF(length int) string {
	// Define the character set for REF
	const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	// In Go 1.20+, you don't need to call rand.Seed anymore
	var ref strings.Builder
	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(characters))
		ref.WriteByte(characters[randomIndex])
	}

	return ref.String()
}

func generateEmailContent(OTP string, REF string) string {
	// สร้าง HTML template สำหรับอีเมล
	emailTemplate := `
        <table width="680px" cellpadding="0" cellspacing="0" border="0">
                            <tbody>
                              <tr>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="80%" bgcolor="#eeeeee" align="center"><h1>Myday-Planner</h1></td>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="20" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="72" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="72" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="72" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="72" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="72" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="80%" bgcolor="#ffffff" align="center" valign="top" style="line-height:24px"><font color="#333333" face="Arial"><span style="font-size:20px">สวัสดี!</span></font><br><font color="#333333" face="Arial"><span style="font-size:16px">กรุณานำรหัส <span class="il">OTP</span> ด้านล่าง ไปกรอกในหน้ายืนยัน.</span></font><br></td>
                                <td width="5%" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="42" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="42" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="42" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="42" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="42" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="72" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="72" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="72" bgcolor="#ffffff" align="center" valign="top">
                                  <table width="100%" height="72" cellpadding="0" cellspacing="0" border="0">
                                    <tbody><tr>
                                      <td width="10%" height="1" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="5%" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="*" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="5%" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="1" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="10%" height="1" bgcolor="#ffffff" style="font-size:0"></td>
                                    </tr>
                                    <tr>
                                      <td width="10%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="5%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="*" height="20" bgcolor="#ffffff" align="center" valign="middle" style="font-size:18px;color:#c00;font-family:Arial">
                                        <span class="il">OTP</span> : <strong style="color:#000">` + OTP + `</strong></td>
                                      <td width="5%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="10%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                    </tr>
                                    <tr>
                                      <td width="10%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="5%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="*" height="20" bgcolor="#ffffff" align="center" valign="middle" style="font-size:18px;color:#c00;font-family:Arial">
                                        <span class="il">Ref</span> : <strong style="color:#000">` + REF + `</strong></td>
                                      <td width="5%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="10%" height="20" bgcolor="#ffffff" style="font-size:0"></td>
                                    </tr>
                                    <tr>
                                      <td width="10%" height="1" bgcolor="#ffffff" style="font-size:0"></td>
                                      <td width="1" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="5%" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="*" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="5%" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="1" height="1" bgcolor="#cc0000" style="font-size:0"></td>
                                      <td width="10%" height="1" bgcolor="#ffffff" style="font-size:0"></td>
                                    </tr>
                                  </tbody></table>
                                </td>
                                <td width="5%" height="72" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="72" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>	
                              <tr>
                                <td width="5%" height="78" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="78" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="78" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="78" bgcolor="#ffffff" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="78" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="54" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="54" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="54" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="54" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="54" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                              <tr>
                                <td width="5%" height="24" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="24" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="80%" height="24" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="24" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                                <td width="5%" height="24" bgcolor="#eeeeee" style="font-size:0">&nbsp;</td>
                              </tr>
                            </tbody>
                          </table>
                          `
	// ไม่ต้องใช้ fmt.Sprintf อีกต่อไปเพราะเราแทรกค่าโดยตรงในแบบ string concatenation
	return emailTemplate
}

func sendEmail(to, subject, body string) error {
	// Load SMTP configuration
	config, err := LoadEmailConfig()
	if err != nil {
		return fmt.Errorf("config loading error: %w", err)
	}

	// Validate SMTP configuration
	if config.Host == "" || config.Port == "" || config.Username == "" || config.Password == "" {
		return fmt.Errorf("incomplete SMTP configuration: host=%q, port=%q, username=%q",
			config.Host, config.Port, config.Username)
	}

	// Set up authentication and server address
	addr := config.Host + ":" + config.Port
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Create email message
	from := config.Username
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	message := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		mime + "\n" +
		body

	// Send email with better error handling
	fmt.Printf("Sending email to %s via %s...\n", to, addr)
	err = smtp.SendMail(addr, auth, from, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("SMTP send error: %w", err)
	}

	fmt.Println("Email sent successfully")
	return nil
}
