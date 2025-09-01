package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	Credit TransactionType = "credit"
	Debit  TransactionType = "debit"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID          uuid.UUID       `json:"id" gorm:"type:varchar(36);primary_key"`
	AccountID   string          `json:"account_id" gorm:"not null;index" validate:"required"`
	Date        time.Time       `json:"date" gorm:"column:transaction_date;not null" validate:"required"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(15,2);not null" validate:"required"`
	Type        TransactionType `json:"type" gorm:"not null" validate:"required"`
	Description string          `json:"description" gorm:"size:255"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Transaction) TableName() string {
	return "transactions"
}

// IsCredit returns true if the transaction is a credit
func (t *Transaction) IsCredit() bool {
	return t.Type == Credit
}

// IsDebit returns true if the transaction is a debit
func (t *Transaction) IsDebit() bool {
	return t.Type == Debit
}

// GetAbsoluteAmount returns the absolute value of the amount
func (t *Transaction) GetAbsoluteAmount() decimal.Decimal {
	return t.Amount.Abs()
}

// BeforeCreate generates a UUID for new transactions
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == (uuid.UUID{}) {
		t.ID = uuid.New()
	}
	return nil
}

// MonthlySummary represents transaction summary for a specific month
type MonthlySummary struct {
	Month            time.Month      `json:"month"`
	Year             int             `json:"year"`
	TransactionCount int             `json:"transaction_count"`
	AverageCredit    decimal.Decimal `json:"average_credit"`
	AverageDebit     decimal.Decimal `json:"average_debit"`
	TotalCredits     decimal.Decimal `json:"total_credits"`
	TotalDebits      decimal.Decimal `json:"total_debits"`
	CreditCount      int             `json:"credit_count"`
	DebitCount       int             `json:"debit_count"`
}

// AccountSummary represents the complete summary of an account
type AccountSummary struct {
	AccountID         string                    `json:"account_id"`
	TotalBalance      decimal.Decimal           `json:"total_balance"`
	MonthlySummaries  map[string]MonthlySummary `json:"monthly_summaries"`
	ProcessedAt       time.Time                 `json:"processed_at"`
	TotalTransactions int                       `json:"total_transactions"`
}

// GetMonthlyKey returns a formatted string for the monthly summary key
func (ms *MonthlySummary) GetMonthlyKey() string {
	return ms.Month.String()
}

// Account represents a financial account
type Account struct {
	ID              string          `json:"id" gorm:"primary_key"`
	Name            string          `json:"name" gorm:"size:255"`
	Email           string          `json:"email" gorm:"not null;size:255"`
	CurrentBalance  decimal.Decimal `json:"current_balance" gorm:"type:decimal(15,2);default:0"`
	CreatedAt       time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	LastProcessedAt *time.Time      `json:"last_processed_at"`

	// Relationship with transactions
	Transactions []Transaction `json:"transactions" gorm:"foreignKey:AccountID"`
}

// TableName specifies the table name for GORM
func (Account) TableName() string {
	return "accounts"
}

// CSVRecord represents the structure of a CSV record
type CSVRecord struct {
	ID          string `csv:"Id"`
	Date        string `csv:"Date"`
	Transaction string `csv:"Transaction"`
}
