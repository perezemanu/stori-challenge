// SQLite implementation for local development
// AWS Lambda doesn't use repositories - processes transactions in memory
package sqlite

import (
	"fmt"
	"time"

	"stori-challenge/internal/domain"
	"stori-challenge/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// transactionRepository implements the TransactionRepository interface using SQLite
type transactionRepository struct {
	db          *gorm.DB
	accountRepo repository.AccountRepository
}

// LOCAL DEVELOPMENT ONLY - AWS Lambda processes in memory
func NewTransactionRepository(db *gorm.DB) repository.TransactionRepository {
	accountRepo := NewAccountRepository(db)
	return &transactionRepository{
		db:          db,
		accountRepo: accountRepo,
	}
}

func (r *transactionRepository) Create(transaction *domain.Transaction) error {
	if err := r.db.Create(transaction).Error; err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

func (r *transactionRepository) CreateBatch(transactions []*domain.Transaction, accountEmail string) error {
	if len(transactions) == 0 {
		return nil
	}

	accountID := transactions[0].AccountID

	account := &domain.Account{
		ID:    accountID,
		Email: accountEmail,
		Name:  fmt.Sprintf("Account %s", accountID),
	}

	var totalBalance decimal.Decimal
	for _, tx := range transactions {
		totalBalance = totalBalance.Add(tx.Amount)
	}
	account.CurrentBalance = totalBalance
	now := time.Now()
	account.LastProcessedAt = &now

	// Create or update account
	if err := r.accountRepo.CreateOrUpdate(account); err != nil {
		return fmt.Errorf("failed to create/update account: %w", err)
	}

	// Delete existing transactions for this account (clean slate)
	if err := r.db.Where("account_id = ?", accountID).Delete(&domain.Transaction{}).Error; err != nil {
		return fmt.Errorf("failed to delete existing transactions: %w", err)
	}

	// Insert all new transactions in batches
	batchSize := 100
	for i := 0; i < len(transactions); i += batchSize {
		end := i + batchSize
		if end > len(transactions) {
			end = len(transactions)
		}

		batch := transactions[i:end]
		if err := r.db.CreateInBatches(batch, batchSize).Error; err != nil {
			return fmt.Errorf("failed to create transaction batch: %w", err)
		}
	}

	return nil
}

func (r *transactionRepository) GetByAccountID(accountID string) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.Where("account_id = ?", accountID).
		Order("transaction_date ASC, created_at ASC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by account ID: %w", err)
	}

	return transactions, nil
}

func (r *transactionRepository) GetByAccountIDAndDateRange(accountID string, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.Where("account_id = ? AND transaction_date BETWEEN ? AND ?", accountID, startDate, endDate).
		Order("transaction_date ASC, created_at ASC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by date range: %w", err)
	}

	return transactions, nil
}

// GetByID retrieves a transaction by its ID
func (r *transactionRepository) GetByID(id string) (*domain.Transaction, error) {
	var transaction domain.Transaction

	err := r.db.Where("id = ?", id).First(&transaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}

	return &transaction, nil
}

func (r *transactionRepository) Update(transaction *domain.Transaction) error {
	err := r.db.Save(transaction).Error
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

func (r *transactionRepository) Delete(id string) error {
	err := r.db.Where("id = ?", id).Delete(&domain.Transaction{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	return nil
}

func (r *transactionRepository) GetTransactionsByMonth(accountID string) (map[string][]*domain.Transaction, error) {
	var transactions []*domain.Transaction

	err := r.db.Where("account_id = ?", accountID).
		Order("transaction_date ASC").
		Find(&transactions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by month: %w", err)
	}

	monthlyTransactions := make(map[string][]*domain.Transaction)
	for _, tx := range transactions {
		monthKey := tx.Date.Format("January")
		monthlyTransactions[monthKey] = append(monthlyTransactions[monthKey], tx)
	}

	return monthlyTransactions, nil
}

func (r *transactionRepository) Count(accountID string) (int64, error) {
	var count int64

	err := r.db.Model(&domain.Transaction{}).
		Where("account_id = ?", accountID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

func (r *transactionRepository) DeleteByAccountID(accountID string) error {
	err := r.db.Where("account_id = ?", accountID).Delete(&domain.Transaction{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete transactions by account ID: %w", err)
	}
	return nil
}