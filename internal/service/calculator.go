package service

import (
	"fmt"
	"time"

	"stori-challenge/internal/domain"

	"github.com/shopspring/decimal"
)

// Calculator handles financial calculations
type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) CalculateSummary(accountID string, transactions []*domain.Transaction) (*domain.AccountSummary, error) {
	if len(transactions) == 0 {
		return &domain.AccountSummary{
			AccountID:         accountID,
			TotalBalance:      decimal.Zero,
			MonthlySummaries:  make(map[string]domain.MonthlySummary),
			ProcessedAt:       time.Now().UTC(),
			TotalTransactions: 0,
		}, nil
	}

	totalBalance := decimal.Zero
	for _, tx := range transactions {
		totalBalance = totalBalance.Add(tx.Amount)
	}

	monthlyData := c.groupTransactionsByMonth(transactions)
	monthlySummaries := make(map[string]domain.MonthlySummary)

	for monthKey, monthTransactions := range monthlyData {
		summary := c.calculateMonthlySummary(monthKey, monthTransactions)
		monthlySummaries[monthKey] = summary
	}

	return &domain.AccountSummary{
		AccountID:         accountID,
		TotalBalance:      totalBalance,
		MonthlySummaries:  monthlySummaries,
		ProcessedAt:       time.Now().UTC(),
		TotalTransactions: len(transactions),
	}, nil
}

// groupTransactionsByMonth groups transactions by month name
func (c *Calculator) groupTransactionsByMonth(transactions []*domain.Transaction) map[string][]*domain.Transaction {
	monthlyData := make(map[string][]*domain.Transaction)

	for _, tx := range transactions {
		monthKey := tx.Date.Format("January") // e.g., "July", "August"
		monthlyData[monthKey] = append(monthlyData[monthKey], tx)
	}

	return monthlyData
}

// calculateMonthlySummary calculates summary statistics for a month
func (c *Calculator) calculateMonthlySummary(monthKey string, transactions []*domain.Transaction) domain.MonthlySummary {
	if len(transactions) == 0 {
		return domain.MonthlySummary{
			Month:            c.parseMonth(monthKey),
			Year:             time.Now().Year(),
			TransactionCount: 0,
			AverageCredit:    decimal.Zero,
			AverageDebit:     decimal.Zero,
			TotalCredits:     decimal.Zero,
			TotalDebits:      decimal.Zero,
			CreditCount:      0,
			DebitCount:       0,
		}
	}

	var (
		totalCredits = decimal.Zero
		totalDebits  = decimal.Zero
		creditCount  = 0
		debitCount   = 0
		sampleDate   = transactions[0].Date
	)

	for _, tx := range transactions {
		if tx.IsCredit() {
			totalCredits = totalCredits.Add(tx.Amount)
			creditCount++
		} else if tx.IsDebit() {
			totalDebits = totalDebits.Add(tx.Amount)
			debitCount++
		}
	}

	// Calculate averages
	var averageCredit, averageDebit decimal.Decimal

	if creditCount > 0 {
		averageCredit = totalCredits.Div(decimal.NewFromInt(int64(creditCount)))
	}

	if debitCount > 0 {
		averageDebit = totalDebits.Div(decimal.NewFromInt(int64(debitCount)))
	}

	return domain.MonthlySummary{
		Month:            sampleDate.Month(),
		Year:             sampleDate.Year(),
		TransactionCount: len(transactions),
		AverageCredit:    averageCredit,
		AverageDebit:     averageDebit,
		TotalCredits:     totalCredits,
		TotalDebits:      totalDebits,
		CreditCount:      creditCount,
		DebitCount:       debitCount,
	}
}

// parseMonth converts month name string to time.Month
func (c *Calculator) parseMonth(monthName string) time.Month {
	switch monthName {
	case "January":
		return time.January
	case "February":
		return time.February
	case "March":
		return time.March
	case "April":
		return time.April
	case "May":
		return time.May
	case "June":
		return time.June
	case "July":
		return time.July
	case "August":
		return time.August
	case "September":
		return time.September
	case "October":
		return time.October
	case "November":
		return time.November
	case "December":
		return time.December
	default:
		return time.January
	}
}

// FormatSummaryForEmail formats the summary for email display
func (c *Calculator) FormatSummaryForEmail(summary *domain.AccountSummary) string {
	var result string

	// Total balance
	result += fmt.Sprintf("Total balance is %s\n", summary.TotalBalance.StringFixed(2))

	// Monthly transaction counts
	for monthName, monthlySummary := range summary.MonthlySummaries {
		result += fmt.Sprintf("Number of transactions in %s: %d\n",
			monthName, monthlySummary.TransactionCount)
	}

	// Calculate overall averages across all months
	totalCredits := decimal.Zero
	totalDebits := decimal.Zero
	totalCreditCount := 0
	totalDebitCount := 0

	for _, monthlySummary := range summary.MonthlySummaries {
		totalCredits = totalCredits.Add(monthlySummary.TotalCredits)
		totalDebits = totalDebits.Add(monthlySummary.TotalDebits)
		totalCreditCount += monthlySummary.CreditCount
		totalDebitCount += monthlySummary.DebitCount
	}

	// Average debit amount (should be negative)
	if totalDebitCount > 0 {
		averageDebit := totalDebits.Div(decimal.NewFromInt(int64(totalDebitCount)))
		result += fmt.Sprintf("Average debit amount: %s\n", averageDebit.StringFixed(2))
	}

	// Average credit amount (should be positive)
	if totalCreditCount > 0 {
		averageCredit := totalCredits.Div(decimal.NewFromInt(int64(totalCreditCount)))
		result += fmt.Sprintf("Average credit amount: %s", averageCredit.StringFixed(2))
	}

	return result
}
