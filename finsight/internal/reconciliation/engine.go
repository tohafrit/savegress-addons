package reconciliation

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Engine handles transaction reconciliation
type Engine struct {
	config     *config.ReconciliationConfig
	batches    map[string]*models.ReconciliationBatch
	exceptions map[string]*models.ReconcileException
	matchers   []Matcher
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
}

// Matcher defines a matching strategy
type Matcher interface {
	Name() string
	Match(source, target *models.Transaction) *MatchResult
}

// MatchResult contains the result of a match attempt
type MatchResult struct {
	Matched     bool
	Confidence  float64
	MatchType   string
	Differences []Difference
}

// Difference represents a difference between matched records
type Difference struct {
	Field    string
	Source   interface{}
	Target   interface{}
	Severity string
}

// NewEngine creates a new reconciliation engine
func NewEngine(cfg *config.ReconciliationConfig) *Engine {
	e := &Engine{
		config:     cfg,
		batches:    make(map[string]*models.ReconciliationBatch),
		exceptions: make(map[string]*models.ReconcileException),
		stopCh:     make(chan struct{}),
	}
	e.initializeMatchers()
	return e
}

func (e *Engine) initializeMatchers() {
	e.matchers = []Matcher{
		NewExactMatcher(),
		NewFuzzyMatcher(e.config.MatchTolerance, e.config.DateTolerance),
		NewReferenceIDMatcher(),
	}
}

// Start starts the reconciliation engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	return nil
}

// Stop stops the reconciliation engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

// CreateBatch creates a new reconciliation batch
func (e *Engine) CreateBatch(source, target string) *models.ReconciliationBatch {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch := &models.ReconciliationBatch{
		ID:        generateBatchID(),
		Source:    source,
		Target:    target,
		Status:    models.BatchStatusPending,
		StartedAt: time.Now(),
	}

	e.batches[batch.ID] = batch
	return batch
}

// Reconcile performs reconciliation between two sets of transactions
func (e *Engine) Reconcile(ctx context.Context, batchID string, sourceTransactions, targetTransactions []*models.Transaction) error {
	e.mu.Lock()
	batch, ok := e.batches[batchID]
	if !ok {
		e.mu.Unlock()
		return ErrBatchNotFound
	}
	batch.Status = models.BatchStatusRunning
	batch.TotalRecords = len(sourceTransactions)
	e.mu.Unlock()

	// Index target transactions for faster lookup
	targetIndex := e.buildIndex(targetTransactions)

	// Process source transactions
	for _, sourceTxn := range sourceTransactions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		matched := false
		var bestMatch *models.Transaction
		var bestResult *MatchResult

		// Try to find a match using all matchers
		candidates := e.findCandidates(sourceTxn, targetIndex)

		for _, matcher := range e.matchers {
			for _, candidate := range candidates {
				result := matcher.Match(sourceTxn, candidate)
				if result.Matched {
					if bestResult == nil || result.Confidence > bestResult.Confidence {
						bestResult = result
						bestMatch = candidate
					}
				}
			}
		}

		if bestResult != nil && bestResult.Matched {
			matched = true
			sourceTxn.ReconcileStatus = models.ReconcileStatusMatched

			// Check for differences
			if len(bestResult.Differences) > 0 {
				for _, diff := range bestResult.Differences {
					if diff.Severity == "error" {
						e.createException(batchID, models.ExceptionTypeAmountDiff, sourceTxn, bestMatch, diff)
					}
				}
			}

			// Remove matched target from index
			delete(targetIndex.byID, bestMatch.ID)
			delete(targetIndex.byExternalID, bestMatch.ExternalID)
		}

		if !matched {
			sourceTxn.ReconcileStatus = models.ReconcileStatusUnmatched
			e.createException(batchID, models.ExceptionTypeMissing, sourceTxn, nil, Difference{
				Field:    "record",
				Source:   sourceTxn.ID,
				Severity: "error",
			})
		}

		e.updateBatchProgress(batchID, matched)
	}

	// Check for unmatched target transactions
	for _, targetTxn := range targetIndex.byID {
		e.createException(batchID, models.ExceptionTypeMissing, nil, targetTxn, Difference{
			Field:    "record",
			Target:   targetTxn.ID,
			Severity: "error",
		})
	}

	// Complete batch
	e.completeBatch(batchID)

	return nil
}

// TransactionIndex indexes transactions for efficient lookup
type TransactionIndex struct {
	byID         map[string]*models.Transaction
	byExternalID map[string]*models.Transaction
	byAmount     map[string][]*models.Transaction
	byDate       map[string][]*models.Transaction
}

func (e *Engine) buildIndex(transactions []*models.Transaction) *TransactionIndex {
	index := &TransactionIndex{
		byID:         make(map[string]*models.Transaction),
		byExternalID: make(map[string]*models.Transaction),
		byAmount:     make(map[string][]*models.Transaction),
		byDate:       make(map[string][]*models.Transaction),
	}

	for _, txn := range transactions {
		index.byID[txn.ID] = txn
		if txn.ExternalID != "" {
			index.byExternalID[txn.ExternalID] = txn
		}

		amountKey := txn.Amount.String()
		index.byAmount[amountKey] = append(index.byAmount[amountKey], txn)

		dateKey := txn.CreatedAt.Format("2006-01-02")
		index.byDate[dateKey] = append(index.byDate[dateKey], txn)
	}

	return index
}

func (e *Engine) findCandidates(source *models.Transaction, index *TransactionIndex) []*models.Transaction {
	var candidates []*models.Transaction
	seen := make(map[string]bool)

	// First check by external ID
	if source.ExternalID != "" {
		if txn, ok := index.byExternalID[source.ExternalID]; ok {
			candidates = append(candidates, txn)
			seen[txn.ID] = true
		}
	}

	// Then check by amount
	amountKey := source.Amount.String()
	for _, txn := range index.byAmount[amountKey] {
		if !seen[txn.ID] {
			candidates = append(candidates, txn)
			seen[txn.ID] = true
		}
	}

	// Then check by date
	dateKey := source.CreatedAt.Format("2006-01-02")
	for _, txn := range index.byDate[dateKey] {
		if !seen[txn.ID] {
			candidates = append(candidates, txn)
			seen[txn.ID] = true
		}
	}

	return candidates
}

func (e *Engine) createException(batchID string, exType models.ExceptionType, source, target *models.Transaction, diff Difference) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var amountDiff decimal.Decimal
	if source != nil && target != nil {
		amountDiff = source.Amount.Sub(target.Amount).Abs()
	} else if source != nil {
		amountDiff = source.Amount
	} else if target != nil {
		amountDiff = target.Amount
	}

	exception := &models.ReconcileException{
		ID:           generateExceptionID(),
		BatchID:      batchID,
		Type:         exType,
		SourceRecord: source,
		TargetRecord: target,
		AmountDiff:   amountDiff,
		Description:  diff.Field + " mismatch",
		Status:       models.ExceptionStatusOpen,
		CreatedAt:    time.Now(),
	}

	e.exceptions[exception.ID] = exception

	if batch, ok := e.batches[batchID]; ok {
		batch.Exceptions++
	}
}

func (e *Engine) updateBatchProgress(batchID string, matched bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch, ok := e.batches[batchID]
	if !ok {
		return
	}

	if matched {
		batch.MatchedRecords++
	} else {
		batch.UnmatchedRecords++
	}
}

func (e *Engine) completeBatch(batchID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	batch, ok := e.batches[batchID]
	if !ok {
		return
	}

	now := time.Now()
	batch.CompletedAt = &now
	batch.Status = models.BatchStatusCompleted

	// Calculate summary
	var sourceTotal, targetTotal, exceptionAmount decimal.Decimal

	for _, exc := range e.exceptions {
		if exc.BatchID == batchID {
			exceptionAmount = exceptionAmount.Add(exc.AmountDiff)
			if exc.SourceRecord != nil {
				sourceTotal = sourceTotal.Add(exc.SourceRecord.Amount)
			}
			if exc.TargetRecord != nil {
				targetTotal = targetTotal.Add(exc.TargetRecord.Amount)
			}
		}
	}

	matchRate := float64(0)
	if batch.TotalRecords > 0 {
		matchRate = float64(batch.MatchedRecords) / float64(batch.TotalRecords)
	}

	batch.Summary = &models.ReconcileSummary{
		SourceTotal:     sourceTotal,
		TargetTotal:     targetTotal,
		Difference:      sourceTotal.Sub(targetTotal).Abs(),
		MatchRate:       matchRate,
		ExceptionAmount: exceptionAmount,
	}
}

// GetBatch retrieves a batch by ID
func (e *Engine) GetBatch(id string) (*models.ReconciliationBatch, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	batch, ok := e.batches[id]
	return batch, ok
}

// GetBatches retrieves all batches
func (e *Engine) GetBatches(filter BatchFilter) []*models.ReconciliationBatch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*models.ReconciliationBatch
	for _, batch := range e.batches {
		if e.matchesBatchFilter(batch, filter) {
			results = append(results, batch)
		}
	}

	// Sort by start time descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})

	return results
}

// BatchFilter defines filters for batch queries
type BatchFilter struct {
	Status    models.BatchStatus
	Source    string
	Target    string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func (e *Engine) matchesBatchFilter(batch *models.ReconciliationBatch, filter BatchFilter) bool {
	if filter.Status != "" && batch.Status != filter.Status {
		return false
	}
	if filter.Source != "" && batch.Source != filter.Source {
		return false
	}
	if filter.Target != "" && batch.Target != filter.Target {
		return false
	}
	if filter.StartDate != nil && batch.StartedAt.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && batch.StartedAt.After(*filter.EndDate) {
		return false
	}
	return true
}

// GetExceptions retrieves exceptions for a batch
func (e *Engine) GetExceptions(batchID string) []*models.ReconcileException {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*models.ReconcileException
	for _, exc := range e.exceptions {
		if exc.BatchID == batchID {
			results = append(results, exc)
		}
	}
	return results
}

// ResolveException resolves an exception
func (e *Engine) ResolveException(id string, resolution string, writeOff bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exc, ok := e.exceptions[id]
	if !ok {
		return ErrExceptionNotFound
	}

	now := time.Now()
	exc.Resolution = resolution
	exc.ResolvedAt = &now

	if writeOff {
		exc.Status = models.ExceptionStatusWriteOff
	} else {
		exc.Status = models.ExceptionStatusResolved
	}

	return nil
}

// GetStats returns reconciliation statistics
func (e *Engine) GetStats() *ReconcileStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &ReconcileStats{
		ByStatus: make(map[string]int),
	}

	for _, batch := range e.batches {
		stats.TotalBatches++
		stats.ByStatus[string(batch.Status)]++

		stats.TotalRecords += batch.TotalRecords
		stats.TotalMatched += batch.MatchedRecords
		stats.TotalUnmatched += batch.UnmatchedRecords
		stats.TotalExceptions += batch.Exceptions
	}

	if stats.TotalRecords > 0 {
		stats.OverallMatchRate = float64(stats.TotalMatched) / float64(stats.TotalRecords)
	}

	return stats
}

// ReconcileStats contains reconciliation statistics
type ReconcileStats struct {
	TotalBatches     int            `json:"total_batches"`
	TotalRecords     int            `json:"total_records"`
	TotalMatched     int            `json:"total_matched"`
	TotalUnmatched   int            `json:"total_unmatched"`
	TotalExceptions  int            `json:"total_exceptions"`
	OverallMatchRate float64        `json:"overall_match_rate"`
	ByStatus         map[string]int `json:"by_status"`
}

func generateBatchID() string {
	return "batch-" + time.Now().Format("20060102150405")
}

func generateExceptionID() string {
	return "exc-" + time.Now().Format("20060102150405.000")
}

// Errors
var (
	ErrBatchNotFound     = &Error{Code: "BATCH_NOT_FOUND", Message: "Batch not found"}
	ErrExceptionNotFound = &Error{Code: "EXCEPTION_NOT_FOUND", Message: "Exception not found"}
)

// Error represents a reconciliation error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
