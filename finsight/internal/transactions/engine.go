package transactions

import (
	"context"
	"sync"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Engine processes and analyzes financial transactions
type Engine struct {
	config       *config.TransactionsConfig
	transactions map[string]*models.Transaction
	accounts     map[string]*models.Account
	categorizer  *Categorizer
	aggregator   *Aggregator
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

// NewEngine creates a new transaction engine
func NewEngine(cfg *config.TransactionsConfig) *Engine {
	return &Engine{
		config:       cfg,
		transactions: make(map[string]*models.Transaction),
		accounts:     make(map[string]*models.Account),
		categorizer:  NewCategorizer(),
		aggregator:   NewAggregator(),
		stopCh:       make(chan struct{}),
	}
}

// Start starts the transaction engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.processLoop(ctx)
	return nil
}

// Stop stops the transaction engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

func (e *Engine) processLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.processPendingTransactions()
		}
	}
}

func (e *Engine) processPendingTransactions() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, txn := range e.transactions {
		if txn.Status == models.TransactionStatusPending {
			// Auto-categorize if enabled
			if e.config.CategorizationEnabled && txn.Category == "" {
				txn.Category = e.categorizer.Categorize(txn)
			}

			// Update aggregates
			e.aggregator.Add(txn)
		}
	}
}

// ProcessTransaction processes a single transaction
func (e *Engine) ProcessTransaction(ctx context.Context, txn *models.Transaction) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Auto-categorize
	if e.config.CategorizationEnabled && txn.Category == "" {
		txn.Category = e.categorizer.Categorize(txn)
	}

	// Store transaction
	e.transactions[txn.ID] = txn

	// Update aggregates
	e.aggregator.Add(txn)

	// Update account balances
	if err := e.updateAccountBalances(txn); err != nil {
		return err
	}

	return nil
}

// GetTransaction retrieves a transaction by ID
func (e *Engine) GetTransaction(id string) (*models.Transaction, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	txn, ok := e.transactions[id]
	return txn, ok
}

// GetTransactions retrieves transactions with filters
func (e *Engine) GetTransactions(filter TransactionFilter) []*models.Transaction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*models.Transaction
	for _, txn := range e.transactions {
		if e.matchesFilter(txn, filter) {
			results = append(results, txn)
		}
	}
	return results
}

// TransactionFilter defines filters for transaction queries
type TransactionFilter struct {
	AccountID    string
	Type         models.TransactionType
	Status       models.TransactionStatus
	Category     string
	MinAmount    *decimal.Decimal
	MaxAmount    *decimal.Decimal
	StartDate    *time.Time
	EndDate      *time.Time
	MerchantID   string
	Limit        int
	Offset       int
}

func (e *Engine) matchesFilter(txn *models.Transaction, filter TransactionFilter) bool {
	if filter.AccountID != "" && txn.SourceAccount != filter.AccountID && txn.DestAccount != filter.AccountID {
		return false
	}
	if filter.Type != "" && txn.Type != filter.Type {
		return false
	}
	if filter.Status != "" && txn.Status != filter.Status {
		return false
	}
	if filter.Category != "" && txn.Category != filter.Category {
		return false
	}
	if filter.MinAmount != nil && txn.Amount.LessThan(*filter.MinAmount) {
		return false
	}
	if filter.MaxAmount != nil && txn.Amount.GreaterThan(*filter.MaxAmount) {
		return false
	}
	if filter.StartDate != nil && txn.CreatedAt.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && txn.CreatedAt.After(*filter.EndDate) {
		return false
	}
	if filter.MerchantID != "" && (txn.Merchant == nil || txn.Merchant.ID != filter.MerchantID) {
		return false
	}
	return true
}

func (e *Engine) updateAccountBalances(txn *models.Transaction) error {
	switch txn.Type {
	case models.TransactionTypeDebit:
		if acc, ok := e.accounts[txn.SourceAccount]; ok {
			acc.Balance = acc.Balance.Sub(txn.Amount)
			acc.AvailableBal = acc.AvailableBal.Sub(txn.Amount)
			acc.UpdatedAt = time.Now()
		}
	case models.TransactionTypeCredit:
		if acc, ok := e.accounts[txn.SourceAccount]; ok {
			acc.Balance = acc.Balance.Add(txn.Amount)
			acc.AvailableBal = acc.AvailableBal.Add(txn.Amount)
			acc.UpdatedAt = time.Now()
		}
	case models.TransactionTypeTransfer:
		if src, ok := e.accounts[txn.SourceAccount]; ok {
			src.Balance = src.Balance.Sub(txn.Amount)
			src.AvailableBal = src.AvailableBal.Sub(txn.Amount)
			src.UpdatedAt = time.Now()
		}
		if dst, ok := e.accounts[txn.DestAccount]; ok {
			dst.Balance = dst.Balance.Add(txn.Amount)
			dst.AvailableBal = dst.AvailableBal.Add(txn.Amount)
			dst.UpdatedAt = time.Now()
		}
	}
	return nil
}

// GetAccount retrieves an account by ID
func (e *Engine) GetAccount(id string) (*models.Account, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	acc, ok := e.accounts[id]
	return acc, ok
}

// CreateAccount creates a new account
func (e *Engine) CreateAccount(acc *models.Account) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	acc.CreatedAt = time.Now()
	acc.UpdatedAt = time.Now()
	e.accounts[acc.ID] = acc
	return nil
}

// GetAccountTransactions gets transactions for an account
func (e *Engine) GetAccountTransactions(accountID string, limit int) []*models.Transaction {
	return e.GetTransactions(TransactionFilter{
		AccountID: accountID,
		Limit:     limit,
	})
}

// GetStats returns transaction statistics
func (e *Engine) GetStats() *TransactionStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.aggregator.GetStats()
}

// TransactionStats contains transaction statistics
type TransactionStats struct {
	TotalCount       int                            `json:"total_count"`
	TotalVolume      decimal.Decimal                `json:"total_volume"`
	ByType           map[string]*TypeStats          `json:"by_type"`
	ByCategory       map[string]*CategoryStats      `json:"by_category"`
	ByStatus         map[string]int                 `json:"by_status"`
	DailyVolume      map[string]decimal.Decimal     `json:"daily_volume"`
	AverageAmount    decimal.Decimal                `json:"average_amount"`
	PeakHour         int                            `json:"peak_hour"`
}

// TypeStats contains stats by transaction type
type TypeStats struct {
	Count   int             `json:"count"`
	Volume  decimal.Decimal `json:"volume"`
	Average decimal.Decimal `json:"average"`
}

// CategoryStats contains stats by category
type CategoryStats struct {
	Count   int             `json:"count"`
	Volume  decimal.Decimal `json:"volume"`
}
