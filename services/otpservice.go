package services

import (
	"context"
	"fmt"
	"math/rand"
	"myapp/model"
	"net/smtp"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// ฟังก์ชันตรวจสอบว่าอีเมลถูกบล็อกหรือไม่
func IsEmailBlocked(c context.Context, firestoreClient *firestore.Client, email string, recordfirebase string) (bool, error) {
	// เข้าถึง document ของ email ใน collection หลัก
	mainDoc := firestoreClient.Collection("EmailBlocked").Doc(email)
	subCollection := mainDoc.Collection(fmt.Sprintf("EmailBlocked_%s", recordfirebase))
	blockedRef := subCollection.Doc(email)

	// ดึงข้อมูลจาก Firestore
	blockedDoc, err := blockedRef.Get(c)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}

	// ตรวจสอบข้อมูล
	if blockedDoc.Exists() {
		blockData := blockedDoc.Data()
		expiresAt, ok := blockData["expiresAt"].(time.Time)
		if ok {
			if time.Now().Before(expiresAt) {
				return true, nil
			}

			// ลบบันทึกถ้าหมดเวลาแล้ว
			_, err = blockedRef.Delete(c)
			if err != nil {
				return false, err
			}
		}
	}

	return false, nil
}

// ฟังก์ชันตรวจสอบจำนวนครั้งที่ขอ OTP และบล็อกถ้าเกินกำหนด
func CheckAndBlockIfNeeded(c context.Context, firestoreClient *firestore.Client, email string, record string) (bool, error) {
	// เข้าถึง subcollection ใน document ของ email
	mainDoc := firestoreClient.Collection("OTPRecords").Doc(email)
	subCollection := mainDoc.Collection(fmt.Sprintf("OTPRecords_%s", record))

	// อ่านทุก document ใน subcollection
	iter := subCollection.Documents(c)
	defer iter.Stop()

	var otpCount int
	currentTime := time.Now()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, err
		}

		data := doc.Data()
		expiresAt, ok := data["expiresAt"].(time.Time)
		if !ok {
			otpCount++
			continue
		}

		if currentTime.Before(expiresAt) {
			otpCount++
		}
	}

	if otpCount >= 3 {
		err := BlockEmail(c, firestoreClient, email, record)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// ฟังก์ชันบล็อกอีเมล
func BlockEmail(c context.Context, firestoreClient *firestore.Client, email string, record string) error {
	blockTime := time.Now()
	expireTime := blockTime.Add(10 * time.Minute)

	blockData := map[string]interface{}{
		"email":     email,
		"createdAt": blockTime,
		"expiresAt": expireTime,
	}

	mainDoc := firestoreClient.Collection("EmailBlocked").Doc(email)
	subCollection := mainDoc.Collection(fmt.Sprintf("EmailBlocked_%s", record))
	_, err := subCollection.Doc(email).Set(c, blockData)

	return err

}

// ฟังก์ชันบันทึกข้อมูล OTP ลงใน Firebase
func SaveOTPRecord(c context.Context, firestoreClient *firestore.Client, email, otp, ref string, record string) error {
	expirationTime := time.Now().Add(15 * time.Minute)
	otpData := map[string]interface{}{
		"email":     email,
		"otp":       otp,
		"reference": ref,
		"is_used":   "0",
		"createdAt": time.Now(),
		"expiresAt": expirationTime,
	}

	mainDoc := firestoreClient.Collection("OTPRecords").Doc(email)
	subCollection := mainDoc.Collection(fmt.Sprintf("OTPRecords_%s", record))
	_, err := subCollection.Doc(ref).Set(c, otpData)
	return err
}

func GenerateOTP(length int) (string, error) {
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

func GenerateREF(length int) string {
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

func GenerateEmailContent(OTP string, REF string) string {
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

func SendingEmail(to, subject, body string) error {
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
