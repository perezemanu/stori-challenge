package repository

import (
	"stori-challenge/internal/domain"
	"time"

	"github.com/shopspring/decimal"
)

// AccountRepository defines the interface for account data access
type AccountRepository interface {
	// Create creates a new account
	Create(account *domain.Account) error

	// GetByID retrieves an account by its ID
	GetByID(id string) (*domain.Account, error)

	// Update updates an existing account
	Update(account *domain.Account) error

	// CreateOrUpdate creates a new account or updates existing one
	CreateOrUpdate(account *domain.Account) error

	// UpdateBalance updates the current balance of an account
	UpdateBalance(accountID string, balance decimal.Decimal) error
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	// Create inserts a new transaction
	Create(transaction *domain.Transaction) error

	// CreateBatch inserts multiple transactions in a single operation, with account management
	CreateBatch(transactions []*domain.Transaction, accountEmail string) error

	// GetByAccountID retrieves all transactions for a specific account
	GetByAccountID(accountID string) ([]*domain.Transaction, error)

	// GetByAccountIDAndDateRange retrieves transactions for an account within a date range
	GetByAccountIDAndDateRange(accountID string, startDate, endDate time.Time) ([]*domain.Transaction, error)

	// GetByID retrieves a transaction by its ID
	GetByID(id string) (*domain.Transaction, error)

	// Update updates an existing transaction
	Update(transaction *domain.Transaction) error

	// Delete deletes a transaction by ID
	Delete(id string) error

	// GetTransactionsByMonth retrieves transactions grouped by month for an account
	GetTransactionsByMonth(accountID string) (map[string][]*domain.Transaction, error)

	// Count returns the total number of transactions for an account
	Count(accountID string) (int64, error)

	// DeleteByAccountID deletes all transactions for an account (useful for testing)
	DeleteByAccountID(accountID string) error
}
