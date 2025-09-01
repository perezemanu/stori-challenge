package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"stori-challenge/internal/aws"
	"stori-challenge/internal/domain"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Reader handles CSV file processing
type Reader struct {
	s3Client *aws.S3Client
	logger   *zap.Logger
}

func NewReader(s3Client *aws.S3Client, logger *zap.Logger) *Reader {
	return &Reader{
		s3Client: s3Client,
		logger:   logger,
	}
}

// ReadTransactions reads transactions from a CSV file or S3 object
func (r *Reader) ReadTransactions(ctx context.Context, filePath, accountID string) ([]*domain.Transaction, error) {
	var reader io.ReadCloser
	var err error

	// Check if it's an S3 URL
	if strings.HasPrefix(filePath, "s3://") {
		if r.s3Client == nil {
			return nil, fmt.Errorf("S3 client not configured for S3 path: %s", filePath)
		}

		bucket, key, err := aws.ParseS3URL(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse S3 URL: %w", err)
		}

		if r.logger != nil {
			r.logger.Info("Reading transactions from S3",
				zap.String("bucket", bucket),
				zap.String("key", key),
				zap.String("account_id", accountID),
			)
		}

		reader, err = r.s3Client.GetObject(ctx, bucket, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get S3 object: %w", err)
		}
	} else {
		if r.logger != nil {
			r.logger.Info("Reading transactions from local file",
				zap.String("file_path", filePath),
				zap.String("account_id", accountID),
			)
		}

		reader, err = os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open CSV file: %w", err)
		}
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = 3 // Enforce exactly 3 fields per record

	var transactions []*domain.Transaction
	lineNumber := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record at line %d: %w", lineNumber+1, err)
		}

		lineNumber++

		// Skip header row
		if lineNumber == 1 && r.isHeaderRow(record) {
			continue
		}

		transaction, err := r.parseTransaction(record, accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction at line %d: %w", lineNumber, err)
		}

		transactions = append(transactions, transaction)
	}

	if len(transactions) == 0 {
		return nil, fmt.Errorf("no valid transactions found in CSV file")
	}

	return transactions, nil
}

// parseTransaction converts a CSV record to a Transaction
func (r *Reader) parseTransaction(record []string, accountID string) (*domain.Transaction, error) {
	if len(record) != 3 {
		return nil, fmt.Errorf("invalid CSV record format, expected 3 fields but got %d", len(record))
	}

	idStr := strings.TrimSpace(record[0])
	if idStr == "" {
		return nil, fmt.Errorf("transaction ID cannot be empty")
	}

	dateStr := strings.TrimSpace(record[1])
	transactionDate, err := r.parseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format '%s': %w", dateStr, err)
	}

	amountStr := strings.TrimSpace(record[2])
	amount, transactionType, err := r.parseAmount(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount format '%s': %w", amountStr, err)
	}

	transaction := &domain.Transaction{
		ID:          uuid.New(),
		AccountID:   accountID,
		Date:        transactionDate,
		Amount:      amount,
		Type:        transactionType,
		Description: fmt.Sprintf("Transaction %s", idStr),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	return transaction, nil
}

func (r *Reader) parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"1/2",        // M/D (assuming current year)
		"1/02",       // M/DD (assuming current year)
		"01/2",       // MM/D (assuming current year)
		"01/02",      // MM/DD (assuming current year)
		"1/2/2006",   // M/D/YYYY
		"1/02/2006",  // M/DD/YYYY
		"01/2/2006",  // MM/D/YYYY
		"01/02/2006", // MM/DD/YYYY
		"2006-01-02", // YYYY-MM-DD
		"2006/01/02", // YYYY/MM/DD
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			// If year is not specified, assume current year
			if date.Year() == 0 {
				currentYear := time.Now().Year()
				date = date.AddDate(currentYear, 0, 0)
			}
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseAmount parses the transaction amount and determines if it's credit or debit
func (r *Reader) parseAmount(amountStr string) (decimal.Decimal, domain.TransactionType, error) {
	amountStr = strings.TrimSpace(amountStr)

	if amountStr == "" {
		return decimal.Zero, "", fmt.Errorf("amount cannot be empty")
	}

	// Determine if it's a credit (+) or debit (-)
	var transactionType domain.TransactionType
	var numericStr string

	if strings.HasPrefix(amountStr, "+") {
		transactionType = domain.Credit
		numericStr = strings.TrimPrefix(amountStr, "+")
	} else if strings.HasPrefix(amountStr, "-") {
		transactionType = domain.Debit
		numericStr = strings.TrimPrefix(amountStr, "-")
	} else {
		// If no sign, treat as positive (credit)
		transactionType = domain.Credit
		numericStr = amountStr
	}

	amount, err := decimal.NewFromString(numericStr)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid numeric amount: %s", numericStr)
	}

	amount = amount.Abs()
	if transactionType == domain.Debit {
		amount = amount.Neg()
	}

	// Validate reasonable amount limits (between -$1M and +$1M)
	maxAmount := decimal.NewFromFloat(1000000)
	if amount.Abs().GreaterThan(maxAmount) {
		return decimal.Zero, "", fmt.Errorf("amount exceeds maximum limit: %s", amountStr)
	}

	if amount.Exponent() < -2 {
		return decimal.Zero, "", fmt.Errorf("amount has too many decimal places (max 2): %s", amountStr)
	}

	return amount, transactionType, nil
}

func (r *Reader) isHeaderRow(record []string) bool {
	if len(record) != 3 {
		return false
	}

	firstCol := strings.ToLower(strings.TrimSpace(record[0]))
	if firstCol == "id" || firstCol == "transaction_id" {
		return true
	}

	secondCol := strings.ToLower(strings.TrimSpace(record[1]))
	if secondCol == "date" || secondCol == "transaction_date" {
		return true
	}

	thirdCol := strings.ToLower(strings.TrimSpace(record[2]))
	if thirdCol == "amount" || thirdCol == "transaction" || thirdCol == "value" {
		return true
	}

	if _, err := strconv.Atoi(strings.TrimSpace(record[0])); err == nil {
		return false
	}

	return false
}

func (r *Reader) ValidateFile(ctx context.Context, filePath string) error {
	maxSize := int64(10 * 1024 * 1024) // 10MB

	if strings.HasPrefix(filePath, "s3://") {
		if r.s3Client == nil {
			return fmt.Errorf("S3 client not configured for S3 path: %s", filePath)
		}

		bucket, key, err := aws.ParseS3URL(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse S3 URL: %w", err)
		}

		size, err := r.s3Client.GetObjectSize(ctx, bucket, key)
		if err != nil {
			return fmt.Errorf("S3 object does not exist or cannot be accessed: %s", filePath)
		}

		if size > maxSize {
			return fmt.Errorf("S3 object is too large: %d bytes (max %d bytes)", size, maxSize)
		}

		reader, err := r.s3Client.GetObject(ctx, bucket, key)
		if err != nil {
			return fmt.Errorf("cannot access S3 object: %w", err)
		}
		defer reader.Close()

		csvReader := csv.NewReader(reader)
		_, err = csvReader.Read()
		if err != nil {
			return fmt.Errorf("invalid CSV format in S3 object: %w", err)
		}
	} else {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("CSV file does not exist: %s", filePath)
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		if fileInfo.Size() > maxSize {
			return fmt.Errorf("CSV file is too large: %d bytes (max %d bytes)", fileInfo.Size(), maxSize)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("cannot open CSV file: %w", err)
		}
		defer file.Close()

		csvReader := csv.NewReader(file)
		_, err = csvReader.Read()
		if err != nil {
			return fmt.Errorf("invalid CSV format: %w", err)
		}
	}

	return nil
}
