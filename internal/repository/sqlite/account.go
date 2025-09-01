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

// accountRepository implements the AccountRepository interface using SQLite
type accountRepository struct {
	db *gorm.DB
}

// LOCAL DEVELOPMENT ONLY - AWS Lambda processes in memory
func NewAccountRepository(db *gorm.DB) repository.AccountRepository {
	return &accountRepository{
		db: db,
	}
}

func (r *accountRepository) Create(account *domain.Account) error {
	if err := r.db.Create(account).Error; err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (r *accountRepository) GetByID(id string) (*domain.Account, error) {
	var account domain.Account

	err := r.db.Where("id = ?", id).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("account not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get account by ID: %w", err)
	}

	return &account, nil
}

// Update updates an existing account
func (r *accountRepository) Update(account *domain.Account) error {
	err := r.db.Save(account).Error
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	return nil
}

// CreateOrUpdate creates a new account or updates existing one
func (r *accountRepository) CreateOrUpdate(account *domain.Account) error {
	var existingAccount domain.Account

	err := r.db.Where("id = ?", account.ID).First(&existingAccount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Account doesn't exist, create it
			return r.Create(account)
		}
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	// Account exists, update it
	existingAccount.Email = account.Email
	existingAccount.Name = account.Name
	existingAccount.CurrentBalance = account.CurrentBalance
	existingAccount.LastProcessedAt = account.LastProcessedAt

	return r.Update(&existingAccount)
}

// UpdateBalance updates the current balance of an account
func (r *accountRepository) UpdateBalance(accountID string, balance decimal.Decimal) error {
	now := time.Now()

	err := r.db.Model(&domain.Account{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"current_balance":   balance,
			"last_processed_at": &now,
			"updated_at":        now,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	return nil
}