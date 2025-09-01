package service

import (
	"testing"
	"time"

	"stori-challenge/internal/domain"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCalculator_CalculateSummary(t *testing.T) {
	calculator := NewCalculator()

	tests := []struct {
		name         string
		accountID    string
		transactions []*domain.Transaction
		expected     expectedSummary
	}{
		{
			name:         "empty transactions",
			accountID:    "test-account",
			transactions: []*domain.Transaction{},
			expected: expectedSummary{
				totalBalance:      decimal.Zero,
				totalTransactions: 0,
				monthlyCount:      0,
			},
		},
		{
			name:      "mixed transactions from PDF example",
			accountID: "test-account",
			transactions: []*domain.Transaction{
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(60.5),
					Type:      domain.Credit,
				},
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(-10.3),
					Type:      domain.Debit,
				},
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.August, 2, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(-20.46),
					Type:      domain.Debit,
				},
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.August, 13, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(10),
					Type:      domain.Credit,
				},
			},
			expected: expectedSummary{
				totalBalance:       decimal.NewFromFloat(39.74),
				totalTransactions:  4,
				monthlyCount:       2, // July and August
				julyTransactions:   2,
				augustTransactions: 2,
			},
		},
		{
			name:      "single credit transaction",
			accountID: "test-account",
			transactions: []*domain.Transaction{
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(100.50),
					Type:      domain.Credit,
				},
			},
			expected: expectedSummary{
				totalBalance:      decimal.NewFromFloat(100.50),
				totalTransactions: 1,
				monthlyCount:      1,
			},
		},
		{
			name:      "single debit transaction",
			accountID: "test-account",
			transactions: []*domain.Transaction{
				{
					ID:        generateUUID(),
					AccountID: "test-account",
					Date:      time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
					Amount:    decimal.NewFromFloat(-50.25),
					Type:      domain.Debit,
				},
			},
			expected: expectedSummary{
				totalBalance:      decimal.NewFromFloat(-50.25),
				totalTransactions: 1,
				monthlyCount:      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculator.CalculateSummary(tt.accountID, tt.transactions)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.accountID, result.AccountID)
			assert.True(t, tt.expected.totalBalance.Equal(result.TotalBalance),
				"Expected balance %s, got %s", tt.expected.totalBalance, result.TotalBalance)
			assert.Equal(t, tt.expected.totalTransactions, result.TotalTransactions)
			assert.Equal(t, tt.expected.monthlyCount, len(result.MonthlySummaries))

			// Check specific monthly counts if provided
			if tt.expected.julyTransactions > 0 {
				julySummary, exists := result.MonthlySummaries["July"]
				assert.True(t, exists, "July summary should exist")
				assert.Equal(t, tt.expected.julyTransactions, julySummary.TransactionCount)
			}

			if tt.expected.augustTransactions > 0 {
				augustSummary, exists := result.MonthlySummaries["August"]
				assert.True(t, exists, "August summary should exist")
				assert.Equal(t, tt.expected.augustTransactions, augustSummary.TransactionCount)
			}
		})
	}
}

func TestCalculator_FormatSummaryForEmail(t *testing.T) {
	calculator := NewCalculator()

	// Create test summary matching PDF example
	summary := &domain.AccountSummary{
		AccountID:    "test-account",
		TotalBalance: decimal.NewFromFloat(39.74),
		MonthlySummaries: map[string]domain.MonthlySummary{
			"July": {
				Month:            time.July,
				Year:             2023,
				TransactionCount: 2,
				AverageCredit:    decimal.NewFromFloat(60.5),
				AverageDebit:     decimal.NewFromFloat(-10.3),
				TotalCredits:     decimal.NewFromFloat(60.5),
				TotalDebits:      decimal.NewFromFloat(-10.3),
				CreditCount:      1,
				DebitCount:       1,
			},
			"August": {
				Month:            time.August,
				Year:             2023,
				TransactionCount: 2,
				AverageCredit:    decimal.NewFromFloat(10),
				AverageDebit:     decimal.NewFromFloat(-20.46),
				TotalCredits:     decimal.NewFromFloat(10),
				TotalDebits:      decimal.NewFromFloat(-20.46),
				CreditCount:      1,
				DebitCount:       1,
			},
		},
	}

	result := calculator.FormatSummaryForEmail(summary)

	assert.Contains(t, result, "Total balance is 39.74")
	assert.Contains(t, result, "Number of transactions in July: 2")
	assert.Contains(t, result, "Number of transactions in August: 2")
	assert.Contains(t, result, "Average debit amount: -15.38")
	assert.Contains(t, result, "Average credit amount: 35.25")
}

func TestCalculator_ParseMonth(t *testing.T) {
	calculator := NewCalculator()

	tests := []struct {
		monthName string
		expected  time.Month
	}{
		{"January", time.January},
		{"February", time.February},
		{"July", time.July},
		{"August", time.August},
		{"December", time.December},
		{"Invalid", time.January}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.monthName, func(t *testing.T) {
			result := calculator.parseMonth(tt.monthName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMonthlySummary_GetMonthlyKey(t *testing.T) {
	summary := domain.MonthlySummary{
		Month: time.July,
	}

	result := summary.GetMonthlyKey()
	assert.Equal(t, "July", result)
}

func TestTransaction_Methods(t *testing.T) {
	creditTransaction := &domain.Transaction{
		Amount: decimal.NewFromFloat(100.50),
		Type:   domain.Credit,
	}

	debitTransaction := &domain.Transaction{
		Amount: decimal.NewFromFloat(-75.25),
		Type:   domain.Debit,
	}

	// Test IsCredit
	assert.True(t, creditTransaction.IsCredit())
	assert.False(t, debitTransaction.IsCredit())

	// Test IsDebit
	assert.False(t, creditTransaction.IsDebit())
	assert.True(t, debitTransaction.IsDebit())

	// Test GetAbsoluteAmount
	assert.True(t, decimal.NewFromFloat(100.50).Equal(creditTransaction.GetAbsoluteAmount()))
	assert.True(t, decimal.NewFromFloat(75.25).Equal(debitTransaction.GetAbsoluteAmount()))
}

type expectedSummary struct {
	totalBalance       decimal.Decimal
	totalTransactions  int
	monthlyCount       int
	julyTransactions   int
	augustTransactions int
}

func generateUUID() uuid.UUID {
	return uuid.New()
}
