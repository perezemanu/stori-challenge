package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"stori-challenge/internal/aws"
	"stori-challenge/internal/infrastructure/csv"
	"stori-challenge/internal/repository"

	"go.uber.org/zap"
)

// EmailService defines the interface for email operations
type EmailService interface {
	SendSummaryEmail(subject, body string) error
}

// Processor orchestrates the transaction processing workflow
type Processor struct {
	transactionRepo repository.TransactionRepository
	emailService    EmailService
	calculator      *Calculator
	csvReader       *csv.Reader
	logger          *zap.Logger
	recipientEmail  string
}

// NewProcessor creates a new processor instance
func NewProcessor(
	transactionRepo repository.TransactionRepository,
	emailService EmailService,
	calculator *Calculator,
	s3Client *aws.S3Client,
	recipientEmail string,
	logger *zap.Logger,
) *Processor {
	return &Processor{
		transactionRepo: transactionRepo,
		emailService:    emailService,
		calculator:      calculator,
		csvReader:       csv.NewReader(s3Client, logger),
		logger:          logger,
		recipientEmail:  recipientEmail,
	}
}

// ProcessFile processes a single CSV file
func (p *Processor) ProcessFile(ctx context.Context, filePath string) error {
	p.logger.Info("Starting file processing", zap.String("file", filePath))

	// Extract account ID from filename (or use default)
	accountID := p.extractAccountIDFromFilename(filePath)

	// Read transactions from CSV
	transactions, err := p.csvReader.ReadTransactions(ctx, filePath, accountID)
	if err != nil {
		return fmt.Errorf("failed to read transactions from CSV: %w", err)
	}

	p.logger.Info("Read transactions from CSV",
		zap.String("file", filePath),
		zap.Int("count", len(transactions)),
		zap.String("account_id", accountID),
	)

	// Store transactions in database (only in local development)
	var totalInDB int64
	if p.transactionRepo != nil {
		if err := p.transactionRepo.CreateBatch(transactions, p.recipientEmail); err != nil {
			return fmt.Errorf("failed to store transactions in database: %w", err)
		}

		// Count actual transactions in database for this account
		count, err := p.transactionRepo.Count(accountID)
		if err != nil {
			p.logger.Warn("Failed to count transactions in database", zap.Error(err))
			totalInDB = 0
		} else {
			totalInDB = count
		}

		p.logger.Info("Processed transactions with database",
			zap.Int("csv_transactions", len(transactions)),
			zap.Int64("total_in_db", totalInDB),
		)
	} else {
		p.logger.Info("Processed transactions in memory (serverless mode)",
			zap.Int("csv_transactions", len(transactions)),
		)
	}

	// Calculate summary
	summary, err := p.calculator.CalculateSummary(accountID, transactions)
	if err != nil {
		return fmt.Errorf("failed to calculate summary: %w", err)
	}

	// Format summary for email
	emailBody := p.calculator.FormatSummaryForEmail(summary)

	// Send summary email
	subject := fmt.Sprintf("Stori Account Summary - %s", accountID)
	if err := p.emailService.SendSummaryEmail(subject, emailBody); err != nil {
		return fmt.Errorf("failed to send summary email: %w", err)
	}

	p.logger.Info("Successfully processed file and sent summary email",
		zap.String("file", filePath),
		zap.String("account_id", accountID),
		zap.Int("transactions", len(transactions)),
		zap.String("balance", summary.TotalBalance.String()),
	)

	return nil
}

// ProcessBatch processes multiple transactions in batches
func (p *Processor) ProcessBatch(ctx context.Context, filePaths []string) error {
	for _, filePath := range filePaths {
		if err := p.ProcessFile(ctx, filePath); err != nil {
			return fmt.Errorf("failed to process file %s: %w", filePath, err)
		}
	}
	return nil
}

// extractAccountIDFromFilename extracts account ID from filename
func (p *Processor) extractAccountIDFromFilename(filePath string) string {
	filename := filepath.Base(filePath)

	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Use filename as account ID, or default if empty
	if name == "" || name == "transactions" {
		return "default-account"
	}

	return name
}
