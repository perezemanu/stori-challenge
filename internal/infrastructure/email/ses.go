package email

import (
	"context"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"stori-challenge/internal/aws"
	"stori-challenge/internal/domain"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type SESService struct {
	sesClient *aws.SESClient
	logger    *zap.Logger
	toEmail   string
}

func NewSESService(sesClient *aws.SESClient, logger *zap.Logger) *SESService {
	return &SESService{
		sesClient: sesClient,
		logger:    logger,
		toEmail:   "",
	}
}

func NewSESServiceWithRecipient(sesClient *aws.SESClient, logger *zap.Logger, toEmail string) *SESService {
	return &SESService{
		sesClient: sesClient,
		logger:    logger,
		toEmail:   toEmail,
	}
}
func (s *SESService) SendAccountSummary(ctx context.Context, email string, summary domain.AccountSummary) error {
	s.logger.Info("Preparing account summary email via SES",
		zap.String("to_email", email),
		zap.String("account_id", summary.AccountID),
		zap.Int("transaction_count", summary.TotalTransactions),
	)

	subject := fmt.Sprintf("Account Summary for %s", summary.AccountID)
	htmlBody, err := s.generateHTMLBody(summary)
	if err != nil {
		s.logger.Error("Failed to generate HTML email body",
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	textBody := s.generateTextBody(summary)

	err = s.sesClient.SendEmail(ctx, email, subject, htmlBody, textBody)
	if err != nil {
		s.logger.Error("Failed to send account summary email via SES",
			zap.String("to_email", email),
			zap.String("account_id", summary.AccountID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Successfully sent account summary email via SES",
		zap.String("to_email", email),
		zap.String("account_id", summary.AccountID),
	)

	return nil
}

func (s *SESService) generateHTMLBody(summary domain.AccountSummary) (string, error) {
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Stori Account Summary</title>
</head>
<body style="margin:0;padding:0;font-family:Arial,sans-serif;background-color:#f8f9fa;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f8f9fa;min-height:100vh;">
        <tr>
            <td align="center" style="padding:20px;">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color:#ffffff;border-radius:16px;max-width:600px;margin:0 auto;box-shadow:0 10px 30px rgba(0,0,0,0.1);">
                    
                    <!-- Header Section with Stori Branding -->
                    <tr>
                        <td style="background-color:#003A40;color:#ffffff;text-align:center;padding:40px 32px 24px;border-radius:16px 16px 0 0;">
                            <!-- Logo -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                <tr>
                                    <td align="center" style="padding-bottom:16px;">
                                        <!-- Stori Logo -->
                                        <img src="https://stori-challenge-manuuu-1756656646.s3.us-west-2.amazonaws.com/assets/images/logo-stori-60x32.png" 
                                             alt="Stori" 
                                             width="60" 
                                             height="32" 
                                             style="display:block;margin:0 auto;border:none;background-color:white;padding:8px;border-radius:6px;" />
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Title -->
                            <h1 style="margin:0;font-size:28px;font-weight:700;color:#ffffff;line-height:1.2;">Account Transaction Summary</h1>
                            
                            <!-- Account Info -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-top:16px;">
                                <tr>
                                    <td align="center">
                                        <div style="background-color:rgba(255,255,255,0.15);border:1px solid rgba(255,255,255,0.2);border-radius:8px;padding:12px 16px;display:inline-block;font-size:13px;font-weight:500;">
                                            Account: {{.AccountID}} • Generated: {{.ProcessedAt.Format "January 2, 2006"}}
                                        </div>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>
                    
                    <!-- Hero Balance Section -->
                    <tr>
                        <td style="background-color:#00d4aa;color:#ffffff;text-align:center;padding:48px 32px;">
                            <div style="font-size:14px;font-weight:600;opacity:0.9;margin-bottom:8px;text-transform:uppercase;letter-spacing:1px;">CURRENT BALANCE</div>
                            <div style="font-size:42px;font-weight:800;margin:8px 0;line-height:1.1;">${{.TotalBalance.StringFixed 2}}</div>
                            <div style="font-size:13px;opacity:0.85;margin-top:12px;font-weight:500;">Based on {{.TotalTransactions}} transactions across multiple months</div>
                        </td>
                    </tr>
                    
                    <!-- Metrics Section -->
                    <tr>
                        <td style="padding:40px 20px;">
                            <!-- First Row of Metrics -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-bottom:20px;">
                                <tr>
                                    <!-- Metric 1 -->
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">📊</div>
                                        <div style="font-size:20px;font-weight:700;color:#003A40;margin:4px 0;">{{.TotalTransactions}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">TOTAL TRANSACTIONS</div>
                                    </td>
                                    <td width="2%"></td>
                                    <!-- Metric 2 -->
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">{{if .TotalBalance.IsPositive}}📈{{else}}📉{{end}}</div>
                                        <div style="font-size:20px;font-weight:700;color:{{if .TotalBalance.IsPositive}}#28a745{{else}}#dc3545{{end}};margin:4px 0;">{{if .TotalBalance.IsPositive}}POSITIVE{{else}}NEGATIVE{{end}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">BALANCE STATUS</div>
                                    </td>
                                    <td width="2%"></td>
                                    <!-- Metric 3 -->
                                    <td width="32%" style="text-align:center;padding:20px 12px;background:#ffffff;border:2px solid #e9ecef;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">🗓️</div>
                                        <div style="font-size:20px;font-weight:700;color:#003A40;margin:4px 0;">{{len .MonthlySummaries}}</div>
                                        <div style="font-size:10px;color:#6c757d;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">ACTIVE MONTHS</div>
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Second Row of Metrics - Overall Averages -->
                            {{range $month, $summary := .MonthlySummaries}}
                            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                <tr>
                                    <!-- Average Credit -->
                                    <td width="49%" style="text-align:center;padding:20px 12px;background:#e8f5e8;border:2px solid #28a745;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">💰</div>
                                        <div style="font-size:18px;font-weight:700;color:#28a745;margin:4px 0;">
                                            {{if $summary.AverageCredit.IsPositive}}
                                                ${{$summary.AverageCredit.StringFixed 2}}
                                            {{else}}
                                                $0.00
                                            {{end}}
                                        </div>
                                        <div style="font-size:10px;color:#155724;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">AVERAGE CREDIT</div>
                                    </td>
                                    <td width="2%"></td>
                                    <!-- Average Debit -->
                                    <td width="49%" style="text-align:center;padding:20px 12px;background:#f8e8e8;border:2px solid #dc3545;border-radius:8px;">
                                        <div style="font-size:24px;margin-bottom:8px;">💸</div>
                                        <div style="font-size:18px;font-weight:700;color:#dc3545;margin:4px 0;">
                                            {{if not $summary.AverageDebit.IsZero}}
                                                ${{$summary.AverageDebit.StringFixed 2}}
                                            {{else}}
                                                $0.00
                                            {{end}}
                                        </div>
                                        <div style="font-size:10px;color:#721c24;text-transform:uppercase;letter-spacing:0.5px;font-weight:600;">AVERAGE DEBIT</div>
                                    </td>
                                </tr>
                            </table>
                            {{break}}
                            {{end}}
                        </td>
                    </tr>
                    
                    <!-- Monthly Breakdown Section -->
                    {{if .MonthlySummaries}}
                    <tr>
                        <td style="padding:0 32px 40px;">
                            <!-- Section Title -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-bottom:20px;">
                                <tr>
                                    <td style="font-size:20px;font-weight:700;color:#003A40;padding:8px 0;">
                                        📅 Monthly Transaction Breakdown
                                    </td>
                                </tr>
                            </table>
                            
                            <!-- Transactions Table -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#f8f9fa;border:2px solid #e9ecef;border-radius:8px;">
                                {{range $month, $summary := .MonthlySummaries}}
                                <tr>
                                    <td style="padding:14px 18px;border-bottom:1px solid #dee2e6;font-size:14px;color:#003A40;font-weight:600;">
                                        <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                            <tr>
                                                <td style="font-weight:700;color:#003A40;">{{$month}}</td>
                                                <td align="right" style="color:#6c757d;">{{$summary.TransactionCount}} transactions</td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                                {{end}}
                            </table>
                        </td>
                    </tr>
                    {{end}}
                    
                    <!-- CTA Section -->
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
                    
                    <!-- Footer -->
                    <tr>
                        <td style="background-color:#f8f9fa;padding:32px;text-align:center;color:#6c757d;font-size:12px;line-height:1.6;border-radius:0 0 16px 16px;">
                            <div style="margin:0 0 16px 0;font-weight:500;">
                                This summary was generated automatically by Stori<br>
                                Your trusted financial partner
                            </div>
                            <div style="margin:0 0 16px 0;">
                                Processed on {{.ProcessedAt.Format "January 2, 2006 at 3:04 PM"}}
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

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, summary); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

func (s *SESService) SendSummaryEmail(subject, body string) error {
	ctx := context.Background()

	toEmail := s.toEmail
	if toEmail == "" {
		toEmail = "perezetchegaraymanuel@gmail.com"
	}

	s.logger.Info("Sending summary email via SES adapter",
		zap.String("subject", subject),
		zap.String("to_email", toEmail),
	)

	accountSummary, err := s.parseTextBodyToSummary(subject, body)
	if err != nil {
		s.logger.Warn("Failed to parse summary data, falling back to plain text email", zap.Error(err))
		err := s.sesClient.SendEmail(ctx, toEmail, subject, body, body)
		if err != nil {
			return fmt.Errorf("failed to send email via SES: %w", err)
		}
		return nil
	}

	htmlBody, err := s.generateHTMLBody(*accountSummary)
	if err != nil {
		s.logger.Warn("Failed to generate HTML email, falling back to plain text", zap.Error(err))
		err := s.sesClient.SendEmail(ctx, toEmail, subject, body, body)
		if err != nil {
			return fmt.Errorf("failed to send email via SES: %w", err)
		}
		return nil
	}

	err = s.sesClient.SendEmail(ctx, toEmail, subject, htmlBody, body)
	if err != nil {
		s.logger.Error("Failed to send summary email via SES adapter",
			zap.String("subject", subject),
			zap.String("to_email", toEmail),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	s.logger.Info("Successfully sent summary email via SES adapter",
		zap.String("subject", subject),
		zap.String("to_email", toEmail),
	)

	return nil
}

func (s *SESService) generateTextBody(summary domain.AccountSummary) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("ACCOUNT SUMMARY FOR %s\n", summary.AccountID))
	builder.WriteString("=" + strings.Repeat("=", len(summary.AccountID)+20) + "\n\n")

	builder.WriteString("Dear Valued Customer,\n\n")
	builder.WriteString(fmt.Sprintf("Here's your account summary for %s:\n\n", summary.AccountID))

	builder.WriteString("SUMMARY:\n")
	builder.WriteString("---------\n")
	builder.WriteString(fmt.Sprintf("Total Balance: $%.2f\n", summary.TotalBalance.InexactFloat64()))
	builder.WriteString(fmt.Sprintf("Total Transactions: %d\n\n", summary.TotalTransactions))

	if len(summary.MonthlySummaries) > 0 {
		builder.WriteString("MONTHLY SUMMARIES:\n")
		builder.WriteString("------------------\n")
		for month, monthlySummary := range summary.MonthlySummaries {
			builder.WriteString(fmt.Sprintf("%s: %d transactions\n", month, monthlySummary.TransactionCount))
		}
		builder.WriteString("\n")
	}

	builder.WriteString(fmt.Sprintf("This summary was processed on %s.\n\n",
		summary.ProcessedAt.Format("January 2, 2006 at 3:04 PM")))
	builder.WriteString("Thank you for banking with us!\n")
	builder.WriteString("If you have any questions, please don't hesitate to contact our support team.\n")

	return builder.String()
}

func (s *SESService) parseTextBodyToSummary(subject, textBody string) (*domain.AccountSummary, error) {
	accountID := "unknown"
	if parts := strings.Split(subject, " - "); len(parts) > 1 {
		accountID = parts[1]
	}

	balanceRegex := regexp.MustCompile(`Total balance is ([+-]?[\d,.]+)`)
	balanceMatch := balanceRegex.FindStringSubmatch(textBody)
	if len(balanceMatch) < 2 {
		return nil, fmt.Errorf("could not parse total balance from text body")
	}

	balanceStr := strings.Replace(balanceMatch[1], ",", "", -1)
	totalBalance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse balance as decimal: %w", err)
	}

	monthlySummaries := make(map[string]domain.MonthlySummary)
	transactionRegex := regexp.MustCompile(`Number of transactions in ([A-Za-z]+): (\d+)`)
	transactionMatches := transactionRegex.FindAllStringSubmatch(textBody, -1)

	totalTransactions := 0
	for _, match := range transactionMatches {
		if len(match) >= 3 {
			month := match[1]
			countStr := match[2]
			count, err := strconv.Atoi(countStr)
			if err == nil {
				totalTransactions += count
				monthlySummaries[month] = domain.MonthlySummary{
					TransactionCount: count,
					Month:            s.parseMonth(month),
					Year:             time.Now().Year(), // Default to current year
				}
			}
		}
	}

	var averageDebit, averageCredit decimal.Decimal
	debitRegex := regexp.MustCompile(`Average debit amount: ([+-]?[\d,.]+)`)
	creditRegex := regexp.MustCompile(`Average credit amount: ([+-]?[\d,.]+)`)

	if debitMatch := debitRegex.FindStringSubmatch(textBody); len(debitMatch) >= 2 {
		debitStr := strings.Replace(debitMatch[1], ",", "", -1)
		if parsed, err := decimal.NewFromString(debitStr); err == nil {
			averageDebit = parsed
		}
	}

	if creditMatch := creditRegex.FindStringSubmatch(textBody); len(creditMatch) >= 2 {
		creditStr := strings.Replace(creditMatch[1], ",", "", -1)
		if parsed, err := decimal.NewFromString(creditStr); err == nil {
			averageCredit = parsed
		}
	}

	for month := range monthlySummaries {
		summary := monthlySummaries[month]
		summary.AverageDebit = averageDebit
		summary.AverageCredit = averageCredit
		monthlySummaries[month] = summary
	}

	return &domain.AccountSummary{
		AccountID:         accountID,
		TotalBalance:      totalBalance,
		TotalTransactions: totalTransactions,
		MonthlySummaries:  monthlySummaries,
		ProcessedAt:       time.Now(),
	}, nil
}

func (s *SESService) parseMonth(monthName string) time.Month {
	switch strings.ToLower(monthName) {
	case "january", "enero":
		return time.January
	case "february", "febrero":
		return time.February
	case "march", "marzo":
		return time.March
	case "april", "abril":
		return time.April
	case "may", "mayo":
		return time.May
	case "june", "junio":
		return time.June
	case "july", "julio":
		return time.July
	case "august", "agosto":
		return time.August
	case "september", "septiembre":
		return time.September
	case "october", "octubre":
		return time.October
	case "november", "noviembre":
		return time.November
	case "december", "diciembre":
		return time.December
	default:
		return time.January
	}
}
