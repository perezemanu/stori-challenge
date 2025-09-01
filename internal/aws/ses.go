package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"go.uber.org/zap"
)

// SESClient provides operations for sending emails via AWS SES
type SESClient struct {
	client    *sesv2.Client
	logger    *zap.Logger
	fromEmail string
}

// NewSESClient creates a new SES client
func NewSESClient(ctx context.Context, fromEmail string, logger *zap.Logger) (*SESClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sesv2.NewFromConfig(cfg)

	return &SESClient{
		client:    client,
		logger:    logger,
		fromEmail: fromEmail,
	}, nil
}

// SendEmail sends an email using SES
func (c *SESClient) SendEmail(ctx context.Context, toEmail, subject, htmlBody, textBody string) error {
	c.logger.Info("Sending email via SES",
		zap.String("to", toEmail),
		zap.String("from", c.fromEmail),
		zap.String("subject", subject),
	)

	// Prepare the email content
	content := &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{},
		},
	}

	// Add HTML body if provided
	if htmlBody != "" {
		content.Simple.Body.Html = &types.Content{
			Data: aws.String(htmlBody),
		}
	}

	// Add text body if provided
	if textBody != "" {
		content.Simple.Body.Text = &types.Content{
			Data: aws.String(textBody),
		}
	}

	// If no text body provided, create a simple one
	if textBody == "" && htmlBody != "" {
		content.Simple.Body.Text = &types.Content{
			Data: aws.String("This email contains HTML content. Please view it in an HTML-capable email client."),
		}
	}

	// Prepare the send email input
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(c.fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: content,
	}

	// Send the email
	result, err := c.client.SendEmail(ctx, input)
	if err != nil {
		c.logger.Error("Failed to send email via SES",
			zap.String("to", toEmail),
			zap.String("from", c.fromEmail),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	c.logger.Info("Successfully sent email via SES",
		zap.String("to", toEmail),
		zap.String("from", c.fromEmail),
		zap.String("message_id", *result.MessageId),
	)

	return nil
}

// SendTemplatedEmail sends an email using a SES template
func (c *SESClient) SendTemplatedEmail(ctx context.Context, toEmail, templateName string, templateData map[string]interface{}) error {
	c.logger.Info("Sending templated email via SES",
		zap.String("to", toEmail),
		zap.String("from", c.fromEmail),
		zap.String("template", templateName),
	)

	// Convert template data to string map
	templateDataStr := make(map[string]string)
	for k, v := range templateData {
		templateDataStr[k] = fmt.Sprintf("%v", v)
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(c.fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: aws.String(templateName),
				TemplateData: aws.String(mapToJSONString(templateDataStr)),
			},
		},
	}

	result, err := c.client.SendEmail(ctx, input)
	if err != nil {
		c.logger.Error("Failed to send templated email via SES",
			zap.String("to", toEmail),
			zap.String("template", templateName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send templated email: %w", err)
	}

	c.logger.Info("Successfully sent templated email via SES",
		zap.String("to", toEmail),
		zap.String("template", templateName),
		zap.String("message_id", *result.MessageId),
	)

	return nil
}

// VerifyEmailAddress verifies an email address with SES
func (c *SESClient) VerifyEmailAddress(ctx context.Context, email string) error {
	c.logger.Info("Verifying email address with SES", zap.String("email", email))

	input := &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(email),
	}

	_, err := c.client.CreateEmailIdentity(ctx, input)
	if err != nil {
		c.logger.Error("Failed to verify email address",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to verify email address: %w", err)
	}

	c.logger.Info("Successfully initiated email verification",
		zap.String("email", email),
	)

	return nil
}

// Note: GetSendingQuota is not available in SES v2 API.
// Use AWS Console or CLI for quota information.

// mapToJSONString converts a string map to JSON string format for SES templates
func mapToJSONString(data map[string]string) string {
	if len(data) == 0 {
		return "{}"
	}

	jsonStr := "{"
	first := true
	for k, v := range data {
		if !first {
			jsonStr += ","
		}
		jsonStr += fmt.Sprintf("\"%s\":\"%s\"", k, v)
		first = false
	}
	jsonStr += "}"

	return jsonStr
}
