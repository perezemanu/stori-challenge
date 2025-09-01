package csv

import (
	"context"
	"os"
	"testing"
	"time"

	"stori-challenge/internal/domain"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestReader_parseAmount(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger)

	tests := []struct {
		name           string
		amountStr      string
		expectedAmount decimal.Decimal
		expectedType   domain.TransactionType
		expectError    bool
	}{
		{
			name:           "positive amount with + sign",
			amountStr:      "+100.50",
			expectedAmount: decimal.NewFromFloat(100.50),
			expectedType:   domain.Credit,
			expectError:    false,
		},
		{
			name:           "negative amount with - sign",
			amountStr:      "-75.25",
			expectedAmount: decimal.NewFromFloat(-75.25),
			expectedType:   domain.Debit,
			expectError:    false,
		},
		{
			name:           "positive amount without sign",
			amountStr:      "50.75",
			expectedAmount: decimal.NewFromFloat(50.75),
			expectedType:   domain.Credit,
			expectError:    false,
		},
		{
			name:           "integer amount",
			amountStr:      "100",
			expectedAmount: decimal.NewFromFloat(100),
			expectedType:   domain.Credit,
			expectError:    false,
		},
		{
			name:           "amount with two decimal places",
			amountStr:      "123.45",
			expectedAmount: decimal.NewFromFloat(123.45),
			expectedType:   domain.Credit,
			expectError:    false,
		},
		{
			name:        "empty amount",
			amountStr:   "",
			expectError: true,
		},
		{
			name:        "non-numeric amount",
			amountStr:   "abc",
			expectError: true,
		},
		{
			name:        "amount exceeding maximum limit",
			amountStr:   "1000000.01",
			expectError: true,
		},
		{
			name:        "amount with too many decimal places",
			amountStr:   "100.123",
			expectError: true,
		},
		{
			name:           "amount with whitespace",
			amountStr:      "  +50.25  ",
			expectedAmount: decimal.NewFromFloat(50.25),
			expectedType:   domain.Credit,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, txType, err := reader.parseAmount(tt.amountStr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.expectedAmount.Equal(amount),
					"Expected amount %s, got %s", tt.expectedAmount, amount)
				assert.Equal(t, tt.expectedType, txType)
			}
		})
	}
}

func TestReader_parseDate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger)

	tests := []struct {
		name        string
		dateStr     string
		expected    time.Time
		expectError bool
	}{
		{
			name:        "MM/DD format current year",
			dateStr:     "07/15",
			expected:    time.Date(time.Now().Year(), 7, 15, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "M/D format current year",
			dateStr:     "7/5",
			expected:    time.Date(time.Now().Year(), 7, 5, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "MM/DD/YYYY format",
			dateStr:     "07/15/2023",
			expected:    time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "M/D/YYYY format",
			dateStr:     "7/5/2023",
			expected:    time.Date(2023, 7, 5, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "YYYY-MM-DD format",
			dateStr:     "2023-07-15",
			expected:    time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "YYYY/MM/DD format",
			dateStr:     "2023/07/15",
			expected:    time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "invalid date format",
			dateStr:     "invalid-date",
			expectError: true,
		},
		{
			name:        "empty date string",
			dateStr:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reader.parseDate(tt.dateStr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Year(), result.Year())
				assert.Equal(t, tt.expected.Month(), result.Month())
				assert.Equal(t, tt.expected.Day(), result.Day())
			}
		})
	}
}

func TestReader_parseTransaction(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger)
	accountID := "test-account"

	tests := []struct {
		name        string
		record      []string
		expectError bool
	}{
		{
			name:        "valid transaction record",
			record:      []string{"1", "7/15", "+60.5"},
			expectError: false,
		},
		{
			name:        "valid debit transaction",
			record:      []string{"2", "7/28", "-10.3"},
			expectError: false,
		},
		{
			name:        "record with too few fields",
			record:      []string{"1", "7/15"},
			expectError: true,
		},
		{
			name:        "record with too many fields",
			record:      []string{"1", "7/15", "+60.5", "extra"},
			expectError: true,
		},
		{
			name:        "record with empty ID",
			record:      []string{"", "7/15", "+60.5"},
			expectError: true,
		},
		{
			name:        "record with invalid date",
			record:      []string{"1", "invalid", "+60.5"},
			expectError: true,
		},
		{
			name:        "record with invalid amount",
			record:      []string{"1", "7/15", "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transaction, err := reader.parseTransaction(tt.record, accountID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, transaction)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transaction)
				assert.Equal(t, accountID, transaction.AccountID)
				assert.NotEqual(t, "", transaction.ID.String())
			}
		})
	}
}

func TestReader_isHeaderRow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger)

	tests := []struct {
		name     string
		record   []string
		expected bool
	}{
		{
			name:     "header with ID column",
			record:   []string{"ID", "Date", "Amount"},
			expected: true,
		},
		{
			name:     "header with transaction_id column",
			record:   []string{"transaction_id", "date", "transaction"},
			expected: true,
		},
		{
			name:     "header with date column",
			record:   []string{"col1", "date", "col3"},
			expected: true,
		},
		{
			name:     "header with amount column",
			record:   []string{"col1", "col2", "amount"},
			expected: true,
		},
		{
			name:     "data row with numeric ID",
			record:   []string{"1", "7/15", "+60.5"},
			expected: false,
		},
		{
			name:     "data row with text ID",
			record:   []string{"TXN001", "7/15", "+60.5"},
			expected: false,
		},
		{
			name:     "wrong number of columns",
			record:   []string{"ID", "Date"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.isHeaderRow(tt.record)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReader_ReadTransactions_LocalFile tests reading from a local CSV string
func TestReader_ReadTransactions_LocalFile(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger)

	// Create a temporary CSV content
	csvContent := `ID,Date,Transaction
1,7/15,+60.5
2,7/28,-10.3
3,8/2,-20.46
4,8/13,+10`

	// Create a temporary file with CSV content
	tmpFile := "/tmp/test_transactions.csv"
	err := writeToFile(tmpFile, csvContent)
	assert.NoError(t, err)

	// Clean up after test
	defer func() {
		_ = removeFile(tmpFile)
	}()

	ctx := context.Background()
	accountID := "test-account"

	transactions, err := reader.ReadTransactions(ctx, tmpFile, accountID)

	assert.NoError(t, err)
	assert.NotNil(t, transactions)
	assert.Equal(t, 4, len(transactions))

	// Check first transaction
	tx1 := transactions[0]
	assert.Equal(t, accountID, tx1.AccountID)
	assert.True(t, decimal.NewFromFloat(60.5).Equal(tx1.Amount))
	assert.Equal(t, domain.Credit, tx1.Type)

	// Check second transaction (debit)
	tx2 := transactions[1]
	assert.True(t, decimal.NewFromFloat(-10.3).Equal(tx2.Amount))
	assert.Equal(t, domain.Debit, tx2.Type)
}

// Helper functions for file operations
func writeToFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

func removeFile(filename string) error {
	return os.Remove(filename)
}

func TestReader_ReadTransactions_S3Path_NilClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	reader := NewReader(nil, logger) // No S3 client

	ctx := context.Background()
	s3Path := "s3://test-bucket/input/transactions.csv"
	accountID := "test-account"

	transactions, err := reader.ReadTransactions(ctx, s3Path, accountID)

	assert.Error(t, err)
	assert.Nil(t, transactions)
	assert.Contains(t, err.Error(), "S3 client not configured")
}
