package email

import (
	"fmt"
	"html/template"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"stori-challenge/internal/config"

	"github.com/jordan-wright/email"
	"go.uber.org/zap"
)

type SMTPService struct {
	config config.Email
	logger *zap.Logger
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(cfg config.Email, logger *zap.Logger) *SMTPService {
	cfg.ConfigureSMTP()
	
	logger.Info("Initializing SMTP email service",
		zap.String("email_mode", cfg.EmailMode),
		zap.String("description", cfg.GetEmailModeDescription()),
		zap.String("smtp_host", cfg.SMTPHost),
		zap.Int("smtp_port", cfg.SMTPPort),
		zap.Bool("has_auth", cfg.SMTPUsername != ""),
	)
	
	return &SMTPService{
		config: cfg,
		logger: logger,
	}
}
func (s *SMTPService) SendSummaryEmail(subject, body string) error {
	s.logger.Info("Preparing to send summary email",
		zap.String("email_mode", s.config.EmailMode),
		zap.String("smtp_endpoint", fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)),
		zap.String("to", s.config.ToAddress),
		zap.String("subject", subject),
	)

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress)
	e.To = []string{s.config.ToAddress}
	e.Subject = subject

	e.Text = []byte(body)

	htmlBody, err := s.createHTMLBody(subject, body)
	if err != nil {
		s.logger.Warn("Failed to create HTML email body, using text only", zap.Error(err))
	} else {
		e.HTML = []byte(htmlBody)
		preview := htmlBody
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		s.logger.Info("Generated HTML email content", zap.String("preview", preview))
	}

	if s.config.SMTPPassword == "password2" {
		s.logger.Info("Test mode - email content generated but not sent")
		s.logger.Info("Full HTML email content:", zap.String("html", htmlBody))
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var sendErr error
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		sendErr = e.Send(addr, s.getSMTPAuth())
	} else {
		sendErr = e.Send(addr, nil)
	}

	if sendErr != nil {
		s.logger.Error("Failed to send email", zap.Error(sendErr))
		return fmt.Errorf("failed to send email: %w", sendErr)
	}

	s.logger.Info("Successfully sent summary email",
		zap.String("email_mode", s.config.EmailMode),
		zap.String("smtp_endpoint", fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)),
		zap.String("to", s.config.ToAddress),
		zap.String("subject", subject),
	)
	
	if s.config.IsMailHogMode() {
		s.logger.Info("Email captured by MailHog - view at http://localhost:8025")
	}

	return nil
}

// createHTMLBody creates an HTML version of the email body
func (s *SMTPService) createHTMLBody(subject, textBody string) (string, error) {
	return s.createGmailCompatibleHTML(subject, textBody)
}

// createGmailCompatibleHTML creates a Gmail-compatible HTML version using tables
func (s *SMTPService) createGmailCompatibleHTML(subject, textBody string) (string, error) {
	lines := strings.Split(textBody, "\n")
	var totalBalance, averageDebit, averageCredit string
	var transactions []string
	var totalTransactionCount int

	accountID := "transactions"
	if parts := strings.Split(subject, " - "); len(parts) > 1 {
		accountID = parts[1]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Total balance is") {
			totalBalance = line
		} else		if strings.HasPrefix(line, "Number of transactions in") {
			transactions = append(transactions, line)
			if parts := strings.Split(line, ": "); len(parts) > 1 {
				if count, err := strconv.Atoi(parts[1]); err == nil {
					totalTransactionCount += count
				}
			}
		} else if strings.HasPrefix(line, "Average debit amount:") {
			averageDebit = line
		} else if strings.HasPrefix(line, "Average credit amount:") {
			averageCredit = line
		}
	}

	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
</head>
<body style="margin:0;padding:0;font-family:Arial,sans-serif;background-color:#f8f9fa;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f8f9fa;min-height:100vh;">
        <tr>                    <td align="center" style="padding:20px;">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color:#ffffff;border-radius:16px;max-width:600px;margin:0 auto;">
                    
                    <tr>
                        <td style="background-color:#003A40;color:#ffffff;text-align:center;padding:40px 32px 24px;border-radius:16px 16px 0 0;">
                            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                <tr>
                                    <td align="center" style="padding-bottom:16px;">
                                        <img src="https://stori-challenge-manuuu-1756656646.s3.us-west-2.amazonaws.com/assets/images/logo-stori-60x32.png" 
                                             alt="Stori" 
                                             width="60" 
                                             height="32" 
                                             style="display:block;margin:0 auto;border:none;background-color:white;padding:8px;border-radius:6px;" />
                                    </td>
                                </tr>
                            </table>
                            
                            <h1 style="margin:0;font-size:28px;font-weight:700;color:#ffffff;line-height:1.2;">Account Transaction Summary</h1>
                            
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-top:16px;">
                                <tr>
                                    <td align="center">
                                        <div style="background-color:rgba(255,255,255,0.15);border:1px solid rgba(255,255,255,0.2);border-radius:8px;padding:12px 16px;display:inline-block;font-size:13px;font-weight:500;">
                                            Account: {{.AccountID}} • Generated: {{.GeneratedDate}}
                                        </div>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <tr>
                        <td style="background-color:#00d4aa;color:#ffffff;text-align:center;padding:48px 32px;">
                            <div style="font-size:14px;font-weight:600;opacity:0.9;margin-bottom:8px;text-transform:uppercase;letter-spacing:1px;">CURRENT BALANCE</div>
                            <div style="font-size:42px;font-weight:800;margin:8px 0;line-height:1.1;">{{.TotalBalance}}</div>
                            <div style="font-size:13px;opacity:0.85;margin-top:12px;font-weight:500;">Based on {{.TotalTransactions}} transactions across 12 months</div>
                        </td>
                    </tr>
                    
                    <tr>
                        <td style="padding:40px 20px;">
                            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                <tr>
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">📊</div>
                                        <div style="font-size:20px;font-weight:700;color:#003A40;margin:4px 0;">{{.TotalTransactions}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">TOTAL TRANSACTIONS</div>
                                    </td>
                                    <td width="2%"></td>
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">📈</div>
                                        <div style="font-size:20px;font-weight:700;color:#003A40;margin:4px 0;">{{.AverageCredit}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">AVG CREDIT</div>
                                    </td>
                                    <td width="2%"></td>
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">📉</div>
                                        <div style="font-size:20px;font-weight:700;color:#003A40;margin:4px 0;">{{.AverageDebit}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">AVG DEBIT</div>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    {{if .Transactions}}
                    <tr>
                        <td style="padding:0 32px 40px;">
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-bottom:20px;">
                                <tr>
                                    <td style="font-size:20px;font-weight:700;color:#003A40;padding:8px 0;">
                                        📅 Monthly Breakdown
                                    </td>
                                </tr>
                            </table>
                            
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#f8f9fa;border:2px solid #e9ecef;border-radius:8px;">
                                {{range .Transactions}}
                                <tr>
                                    <td style="padding:14px 18px;border-bottom:1px solid #dee2e6;font-size:14px;color:#003A40;font-weight:600;">
                                        {{.}}
                                    </td>
                                </tr>
                                {{end}}
                            </table>
                        </td>
                    </tr>
                    {{end}}
                    
                    <tr>
                        <td style="background-color:#003A40;color:#ffffff;text-align:center;padding:32px;">
                            <div style="font-size:16px;font-weight:500;margin:0 0 16px 0;">Need help or have questions?</div>
                            <table cellpadding="0" cellspacing="0" border="0" style="margin:0 auto;">
                                <tr>
                                    <td style="background-color:#00d4aa;border-radius:8px;">
                                        <a href="https://www.storicard.com/" style="display:block;color:#ffffff;text-decoration:none;padding:14px 28px;font-weight:700;font-size:14px;text-transform:uppercase;letter-spacing:0.5px;">VISIT STORICARD.COM</a>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <tr>
                        <td style="background-color:#f8f9fa;padding:32px;text-align:center;color:#6c757d;font-size:12px;line-height:1.6;border-radius:0 0 16px 16px;">
                            <div style="margin:0 0 16px 0;font-weight:500;">
                                This summary was generated automatically by Stori<br>
                                Your trusted financial partner
                            </div>
                            <div style="margin:0;">
                                <a href="https://www.storicard.com/privacy" style="color:#003A40;text-decoration:none;font-weight:600;">Privacy Policy</a> • 
                                <a href="https://www.storicard.com/terms" style="color:#003A40;text-decoration:none;font-weight:600;">Terms of Service</a> • 
                                <a href="https://www.storicard.com/support" style="color:#003A40;text-decoration:none;font-weight:600;">Support</a>
                            </div>
                        </td>
                    </tr>
                    
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

	// Parse template
	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	data := struct {
		Subject           string
		AccountID         string
		GeneratedDate     string
		TotalBalance      string
		TotalTransactions string
		Transactions      []string
		AverageDebit      string
		AverageCredit     string
	}{
		Subject:           subject,
		AccountID:         accountID,
		GeneratedDate:     time.Now().Format("January 2, 2006"),
		TotalBalance:      totalBalance,
		TotalTransactions: strconv.Itoa(totalTransactionCount),
		Transactions:      transactions,
		AverageDebit:      averageDebit,
		AverageCredit:     averageCredit,
	}

	var htmlBuffer strings.Builder
	if err := tmpl.Execute(&htmlBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return htmlBuffer.String(), nil
}

// getSMTPAuth returns SMTP authentication if credentials are provided
func (s *SMTPService) getSMTPAuth() smtp.Auth {
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		return smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}
	return nil
}
