package model

// EmailConfig holds the SMTP configuration
type EmailConfig struct {
	Host     string `yaml:"host" gorm:"column:host"`
	Port     string `yaml:"port" gorm:"column:port"`
	Username string `yaml:"username" gorm:"column:username"`
	Password string `yaml:"password" gorm:"column:password"`
}

// TableName specifies the database table name for GORM
func (EmailConfig) TableName() string {
	return "OTPconfig"
}
