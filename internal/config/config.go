package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/viper"
)

type Config struct {
	Environment string   `envconfig:"ENVIRONMENT" default:"local"`
	DataPath    string   `envconfig:"DATA_PATH" default:"/app/data"`
	Database    Database `envconfig:",squash"`
	Email       Email    `envconfig:",squash"`
	Security    Security `envconfig:",squash"`
}

// Database configuration for local development (SQLite)
// AWS Lambda doesn't use database - processes transactions in memory
type Database struct {
	Driver string `envconfig:"DB_DRIVER" default:"sqlite"`
}

type Email struct {
	// Email Mode Selection: development (MailHog), gmail (real Gmail), production (AWS SES)
	EmailMode string `envconfig:"EMAIL_MODE" default:"development"`

	// MailHog Configuration (for development mode)
	MailHogHost string `envconfig:"MAILHOG_HOST" default:"mailhog"`
	MailHogPort int    `envconfig:"MAILHOG_PORT" default:"1025"`

	// Gmail Configuration (for demo/testing mode)
	GmailHost     string `envconfig:"GMAIL_HOST" default:"smtp.gmail.com"`
	GmailPort     int    `envconfig:"GMAIL_PORT" default:"587"`
	GmailUsername string `envconfig:"GMAIL_USERNAME"`
	GmailPassword string `envconfig:"GMAIL_PASSWORD"`

	// Legacy SMTP Configuration (for backward compatibility)
	SMTPHost     string `envconfig:"SMTP_HOST"`
	SMTPPort     int    `envconfig:"SMTP_PORT"`
	SMTPUsername string `envconfig:"SMTP_USERNAME"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD"`

	// General Email Configuration
	FromAddress string `envconfig:"FROM_ADDRESS" default:"noreply@stori.com"`
	FromName    string `envconfig:"FROM_NAME" default:"Stori Account Summary"`
	ToAddress   string `envconfig:"TO_ADDRESS" required:"true"`
	ToName      string `envconfig:"TO_NAME" default:"Account Holder"`

	// SES Configuration (for AWS)
	SESFromEmail string `envconfig:"SES_FROM_EMAIL" default:"noreply@stori.com"`
	SESRegion    string `envconfig:"SES_REGION" default:"us-west-2"`
}

type Security struct {
	MaxFileSize       int64         `envconfig:"MAX_FILE_SIZE" default:"10485760"`
	AllowedExtensions []string      `envconfig:"ALLOWED_EXTENSIONS" default:"csv"`
	ProcessTimeout    time.Duration `envconfig:"PROCESS_TIMEOUT" default:"30s"`
	MaxTransactions   int           `envconfig:"MAX_TRANSACTIONS" default:"100000"`
	MaxAmountLimit    float64       `envconfig:"MAX_AMOUNT_LIMIT" default:"1000000.00"`
}

func LoadConfig() (*Config, error) {
	var cfg Config

	// Set up Viper for configuration file support
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/app/config")

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return &cfg, nil
}

// IsServerless returns true if running in AWS Lambda or serverless environment
func (c *Config) IsServerless() bool {
	return c.Environment == "aws" || c.Environment == "lambda" || c.Environment == "serverless"
}

// IsLocal returns true if running in local development environment
func (c *Config) IsLocal() bool {
	return c.Environment == "local" || c.Environment == "development"
}

// GetDSN returns the SQLite database path for local development
func (d Database) GetDSN() string {
	// Always SQLite for local development
	return "/app/db/transactions.db"
}

// ConfigureSMTP automatically configures SMTP settings based on EmailMode
// This method sets the actual SMTP connection parameters used by the email service
func (e *Email) ConfigureSMTP() {
	switch e.EmailMode {
	case "development", "mailhog":
		// Use MailHog for development (no authentication required)
		e.SMTPHost = e.MailHogHost
		e.SMTPPort = e.MailHogPort
		e.SMTPUsername = ""
		e.SMTPPassword = ""
	case "gmail", "production":
		// Use Gmail for real email delivery
		e.SMTPHost = e.GmailHost
		e.SMTPPort = e.GmailPort
		e.SMTPUsername = e.GmailUsername
		e.SMTPPassword = e.GmailPassword
	default:
		// Keep legacy configuration if mode not recognized
		// This provides backward compatibility with existing deployments
	}
}

// GetEmailModeDescription returns a human-readable description of the current email mode
func (e Email) GetEmailModeDescription() string {
	switch e.EmailMode {
	case "development", "mailhog":
		return fmt.Sprintf("Development Mode (MailHog at %s:%d)", e.MailHogHost, e.MailHogPort)
	case "gmail":
		return fmt.Sprintf("Gmail Mode (SMTP at %s:%d)", e.GmailHost, e.GmailPort)
	case "production":
		return "Production Mode (AWS SES)"
	default:
		return fmt.Sprintf("Legacy Mode (SMTP at %s:%d)", e.SMTPHost, e.SMTPPort)
	}
}

// IsMailHogMode returns true if using MailHog for email testing
func (e Email) IsMailHogMode() bool {
	return e.EmailMode == "development" || e.EmailMode == "mailhog"
}

// IsGmailMode returns true if using Gmail for real email delivery
func (e Email) IsGmailMode() bool {
	return e.EmailMode == "gmail"
}
