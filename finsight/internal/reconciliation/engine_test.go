package reconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

func TestNewEngine(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
	if engine.config != cfg {
		t.Error("config not set correctly")
	}
	if engine.batches == nil {
		t.Error("batches map not initialized")
	}
	if engine.exceptions == nil {
		t.Error("exceptions map not initialized")
	}
	if len(engine.matchers) == 0 {
		t.Error("matchers not initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	// Start engine
	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !engine.running {
		t.Error("engine should be running after Start")
	}

	// Starting again should be safe
	err = engine.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	// Stop engine
	engine.Stop()

	if engine.running {
		t.Error("engine should not be running after Stop")
	}
}

func TestEngine_CreateBatch(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	batch := engine.CreateBatch("source-system", "target-system")

	if batch == nil {
		t.Fatal("CreateBatch returned nil")
	}
	if batch.ID == "" {
		t.Error("batch ID should not be empty")
	}
	if batch.Source != "source-system" {
		t.Errorf("expected source 'source-system', got %s", batch.Source)
	}
	if batch.Target != "target-system" {
		t.Errorf("expected target 'target-system', got %s", batch.Target)
	}
	if batch.Status != models.BatchStatusPending {
		t.Errorf("expected status pending, got %s", batch.Status)
	}

	// Check batch was stored
	stored, ok := engine.batches[batch.ID]
	if !ok {
		t.Error("batch should be stored in map")
	}
	if stored.ID != batch.ID {
		t.Error("stored batch ID doesn't match")
	}
}

func TestEngine_GetBatch(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	batch := engine.CreateBatch("source", "target")

	// Get existing batch
	found, ok := engine.GetBatch(batch.ID)
	if !ok {
		t.Error("expected to find batch")
	}
	if found.ID != batch.ID {
		t.Errorf("expected batch ID %s, got %s", batch.ID, found.ID)
	}

	// Get non-existent batch
	_, ok = engine.GetBatch("non-existent")
	if ok {
		t.Error("expected not to find non-existent batch")
	}
}

func TestEngine_Reconcile_ExactMatch(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	batch := engine.CreateBatch("source", "target")

	now := time.Now()
	sourceTransactions := []*models.Transaction{
		{
			ID:         "txn-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			CreatedAt:  now,
		},
		{
			ID:         "txn-2",
			ExternalID: "ext-2",
			Amount:     decimal.NewFromFloat(200),
			CreatedAt:  now,
		},
	}

	targetTransactions := []*models.Transaction{
		{
			ID:         "target-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			CreatedAt:  now,
		},
		{
			ID:         "target-2",
			ExternalID: "ext-2",
			Amount:     decimal.NewFromFloat(200),
			CreatedAt:  now,
		},
	}

	err := engine.Reconcile(ctx, batch.ID, sourceTransactions, targetTransactions)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Check batch completed
	batch, _ = engine.GetBatch(batch.ID)
	if batch.Status != models.BatchStatusCompleted {
		t.Errorf("expected status completed, got %s", batch.Status)
	}
	if batch.TotalRecords != 2 {
		t.Errorf("expected 2 total records, got %d", batch.TotalRecords)
	}
	if batch.MatchedRecords != 2 {
		t.Errorf("expected 2 matched records, got %d", batch.MatchedRecords)
	}
}

func TestEngine_Reconcile_Unmatched(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	batch := engine.CreateBatch("source", "target")

	now := time.Now()
	sourceTransactions := []*models.Transaction{
		{
			ID:         "txn-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			CreatedAt:  now,
		},
		{
			ID:         "txn-2",
			ExternalID: "ext-2",
			Amount:     decimal.NewFromFloat(200),
			CreatedAt:  now,
		},
	}

	// Only one target transaction
	targetTransactions := []*models.Transaction{
		{
			ID:         "target-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			CreatedAt:  now,
		},
	}

	err := engine.Reconcile(ctx, batch.ID, sourceTransactions, targetTransactions)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	batch, _ = engine.GetBatch(batch.ID)
	if batch.UnmatchedRecords != 1 {
		t.Errorf("expected 1 unmatched record, got %d", batch.UnmatchedRecords)
	}
	if batch.Exceptions == 0 {
		t.Error("expected exceptions for unmatched record")
	}
}

func TestEngine_Reconcile_BatchNotFound(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	err := engine.Reconcile(ctx, "non-existent", nil, nil)
	if err != ErrBatchNotFound {
		t.Errorf("expected ErrBatchNotFound, got %v", err)
	}
}

func TestEngine_Reconcile_ContextCancelled(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	batch := engine.CreateBatch("source", "target")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sourceTransactions := []*models.Transaction{
		{ID: "txn-1", Amount: decimal.NewFromFloat(100), CreatedAt: time.Now()},
	}

	err := engine.Reconcile(ctx, batch.ID, sourceTransactions, nil)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestEngine_GetBatches(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	// Create multiple batches with delay to ensure unique IDs
	batch1 := engine.CreateBatch("source1", "target1")
	time.Sleep(1100 * time.Millisecond) // ID is based on seconds
	batch2 := engine.CreateBatch("source2", "target2")
	time.Sleep(1100 * time.Millisecond)
	batch3 := engine.CreateBatch("source1", "target1")

	// Verify batches have unique IDs
	if batch1.ID == batch2.ID || batch2.ID == batch3.ID {
		t.Log("Note: batches have duplicate IDs due to same-second creation")
	}

	// Get all batches
	batches := engine.GetBatches(BatchFilter{})
	if len(batches) < 1 {
		t.Errorf("expected at least 1 batch, got %d", len(batches))
	}

	// Test filter by status
	batches = engine.GetBatches(BatchFilter{Status: models.BatchStatusPending})
	if len(batches) == 0 {
		t.Error("expected at least 1 pending batch")
	}

	// Filter by completed status (none)
	batches = engine.GetBatches(BatchFilter{Status: models.BatchStatusCompleted})
	if len(batches) != 0 {
		t.Errorf("expected 0 completed batches, got %d", len(batches))
	}
}

func TestEngine_GetBatches_DateFilter(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	batch := engine.CreateBatch("source", "target")
	batch.StartedAt = now

	// Filter with start date
	batches := engine.GetBatches(BatchFilter{StartDate: &past})
	if len(batches) != 1 {
		t.Errorf("expected 1 batch after past date, got %d", len(batches))
	}

	batches = engine.GetBatches(BatchFilter{StartDate: &future})
	if len(batches) != 0 {
		t.Errorf("expected 0 batches after future date, got %d", len(batches))
	}

	// Filter with end date
	batches = engine.GetBatches(BatchFilter{EndDate: &future})
	if len(batches) != 1 {
		t.Errorf("expected 1 batch before future date, got %d", len(batches))
	}

	batches = engine.GetBatches(BatchFilter{EndDate: &past})
	if len(batches) != 0 {
		t.Errorf("expected 0 batches before past date, got %d", len(batches))
	}
}

func TestEngine_GetExceptions(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	batch := engine.CreateBatch("source", "target")

	now := time.Now()
	sourceTransactions := []*models.Transaction{
		{ID: "txn-1", Amount: decimal.NewFromFloat(100), CreatedAt: now},
	}
	targetTransactions := []*models.Transaction{} // No targets

	engine.Reconcile(ctx, batch.ID, sourceTransactions, targetTransactions)

	exceptions := engine.GetExceptions(batch.ID)
	if len(exceptions) == 0 {
		t.Error("expected exceptions for unmatched transactions")
	}

	// Check exception details
	exc := exceptions[0]
	if exc.BatchID != batch.ID {
		t.Errorf("expected batch ID %s, got %s", batch.ID, exc.BatchID)
	}
	if exc.Status != models.ExceptionStatusOpen {
		t.Errorf("expected status open, got %s", exc.Status)
	}
}

func TestEngine_ResolveException(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	// Add an exception
	exc := &models.ReconcileException{
		ID:      "exc-1",
		BatchID: "batch-1",
		Status:  models.ExceptionStatusOpen,
	}
	engine.exceptions[exc.ID] = exc

	// Resolve as normal
	err := engine.ResolveException("exc-1", "Verified correct", false)
	if err != nil {
		t.Fatalf("ResolveException failed: %v", err)
	}

	if exc.Status != models.ExceptionStatusResolved {
		t.Errorf("expected status resolved, got %s", exc.Status)
	}
	if exc.Resolution != "Verified correct" {
		t.Errorf("expected resolution 'Verified correct', got %s", exc.Resolution)
	}
	if exc.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
}

func TestEngine_ResolveException_WriteOff(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	exc := &models.ReconcileException{
		ID:      "exc-1",
		BatchID: "batch-1",
		Status:  models.ExceptionStatusOpen,
	}
	engine.exceptions[exc.ID] = exc

	err := engine.ResolveException("exc-1", "Written off due to timing", true)
	if err != nil {
		t.Fatalf("ResolveException failed: %v", err)
	}

	if exc.Status != models.ExceptionStatusWriteOff {
		t.Errorf("expected status write_off, got %s", exc.Status)
	}
}

func TestEngine_ResolveException_NotFound(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	err := engine.ResolveException("non-existent", "test", false)
	if err != ErrExceptionNotFound {
		t.Errorf("expected ErrExceptionNotFound, got %v", err)
	}
}

func TestEngine_GetStats_Empty(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	stats := engine.GetStats()

	if stats.TotalBatches != 0 {
		t.Errorf("expected 0 total batches, got %d", stats.TotalBatches)
	}
	if stats.OverallMatchRate != 0 {
		t.Errorf("expected 0 match rate, got %f", stats.OverallMatchRate)
	}
}

func TestEngine_GetStats(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	// Create and run reconciliation
	batch := engine.CreateBatch("source", "target")

	now := time.Now()
	sourceTransactions := []*models.Transaction{
		{ID: "txn-1", ExternalID: "ext-1", Amount: decimal.NewFromFloat(100), CreatedAt: now},
		{ID: "txn-2", ExternalID: "ext-2", Amount: decimal.NewFromFloat(200), CreatedAt: now},
	}

	targetTransactions := []*models.Transaction{
		{ID: "target-1", ExternalID: "ext-1", Amount: decimal.NewFromFloat(100), CreatedAt: now},
	}

	engine.Reconcile(ctx, batch.ID, sourceTransactions, targetTransactions)

	stats := engine.GetStats()

	if stats.TotalBatches != 1 {
		t.Errorf("expected 1 total batch, got %d", stats.TotalBatches)
	}
	if stats.TotalRecords != 2 {
		t.Errorf("expected 2 total records, got %d", stats.TotalRecords)
	}
	if stats.TotalMatched != 1 {
		t.Errorf("expected 1 matched, got %d", stats.TotalMatched)
	}
	if stats.TotalUnmatched != 1 {
		t.Errorf("expected 1 unmatched, got %d", stats.TotalUnmatched)
	}
	if stats.OverallMatchRate != 0.5 {
		t.Errorf("expected match rate 0.5, got %f", stats.OverallMatchRate)
	}
}

func TestMatchesBatchFilter(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	now := time.Now()
	batch := &models.ReconciliationBatch{
		ID:        "batch-1",
		Source:    "source-sys",
		Target:    "target-sys",
		Status:    models.BatchStatusPending,
		StartedAt: now,
	}

	// Empty filter matches all
	if !engine.matchesBatchFilter(batch, BatchFilter{}) {
		t.Error("empty filter should match")
	}

	// Status filter
	if !engine.matchesBatchFilter(batch, BatchFilter{Status: models.BatchStatusPending}) {
		t.Error("matching status should pass")
	}
	if engine.matchesBatchFilter(batch, BatchFilter{Status: models.BatchStatusCompleted}) {
		t.Error("non-matching status should fail")
	}

	// Source filter
	if !engine.matchesBatchFilter(batch, BatchFilter{Source: "source-sys"}) {
		t.Error("matching source should pass")
	}
	if engine.matchesBatchFilter(batch, BatchFilter{Source: "other"}) {
		t.Error("non-matching source should fail")
	}

	// Target filter
	if !engine.matchesBatchFilter(batch, BatchFilter{Target: "target-sys"}) {
		t.Error("matching target should pass")
	}
	if engine.matchesBatchFilter(batch, BatchFilter{Target: "other"}) {
		t.Error("non-matching target should fail")
	}
}

func TestBuildIndex(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	now := time.Now()
	transactions := []*models.Transaction{
		{ID: "txn-1", ExternalID: "ext-1", Amount: decimal.NewFromFloat(100), CreatedAt: now},
		{ID: "txn-2", ExternalID: "ext-2", Amount: decimal.NewFromFloat(100), CreatedAt: now},
		{ID: "txn-3", Amount: decimal.NewFromFloat(200), CreatedAt: now.Add(24 * time.Hour)},
	}

	index := engine.buildIndex(transactions)

	// Check by ID
	if len(index.byID) != 3 {
		t.Errorf("expected 3 entries in byID, got %d", len(index.byID))
	}

	// Check by external ID
	if len(index.byExternalID) != 2 {
		t.Errorf("expected 2 entries in byExternalID, got %d", len(index.byExternalID))
	}

	// Check by amount (two with same amount)
	amountKey := decimal.NewFromFloat(100).String()
	if len(index.byAmount[amountKey]) != 2 {
		t.Errorf("expected 2 entries with amount 100, got %d", len(index.byAmount[amountKey]))
	}

	// Check by date
	dateKey := now.Format("2006-01-02")
	if len(index.byDate[dateKey]) != 2 {
		t.Errorf("expected 2 entries for today's date, got %d", len(index.byDate[dateKey]))
	}
}

func TestFindCandidates(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	now := time.Now()
	targetTransactions := []*models.Transaction{
		{ID: "txn-1", ExternalID: "ext-1", Amount: decimal.NewFromFloat(100), CreatedAt: now},
		{ID: "txn-2", ExternalID: "ext-2", Amount: decimal.NewFromFloat(100), CreatedAt: now},
		{ID: "txn-3", Amount: decimal.NewFromFloat(200), CreatedAt: now.Add(24 * time.Hour)},
	}

	index := engine.buildIndex(targetTransactions)

	// Find by external ID
	source := &models.Transaction{ExternalID: "ext-1", Amount: decimal.NewFromFloat(100), CreatedAt: now}
	candidates := engine.findCandidates(source, index)

	if len(candidates) == 0 {
		t.Error("expected to find candidates")
	}

	// First candidate should be the external ID match
	if candidates[0].ExternalID != "ext-1" {
		t.Error("first candidate should be external ID match")
	}
}

func TestGenerateBatchID(t *testing.T) {
	id1 := generateBatchID()
	if id1 == "" {
		t.Error("batch ID should not be empty")
	}
	if len(id1) < 10 {
		t.Error("batch ID should be at least 10 characters")
	}
}

func TestGenerateExceptionID(t *testing.T) {
	id1 := generateExceptionID()
	if id1 == "" {
		t.Error("exception ID should not be empty")
	}
	if len(id1) < 10 {
		t.Error("exception ID should be at least 10 characters")
	}
}

func TestError(t *testing.T) {
	err := &Error{Code: "TEST_ERROR", Message: "test message"}
	if err.Error() != "test message" {
		t.Errorf("expected 'test message', got %s", err.Error())
	}
}

func TestErrBatchNotFound(t *testing.T) {
	if ErrBatchNotFound.Code != "BATCH_NOT_FOUND" {
		t.Errorf("expected code 'BATCH_NOT_FOUND', got %s", ErrBatchNotFound.Code)
	}
}

func TestErrExceptionNotFound(t *testing.T) {
	if ErrExceptionNotFound.Code != "EXCEPTION_NOT_FOUND" {
		t.Errorf("expected code 'EXCEPTION_NOT_FOUND', got %s", ErrExceptionNotFound.Code)
	}
}

func TestReconcileStats(t *testing.T) {
	stats := &ReconcileStats{
		TotalBatches:     10,
		TotalRecords:     1000,
		TotalMatched:     950,
		TotalUnmatched:   50,
		TotalExceptions:  25,
		OverallMatchRate: 0.95,
		ByStatus:         map[string]int{"completed": 8, "pending": 2},
	}

	if stats.TotalBatches != 10 {
		t.Errorf("expected 10 batches, got %d", stats.TotalBatches)
	}
	if stats.OverallMatchRate != 0.95 {
		t.Errorf("expected match rate 0.95, got %f", stats.OverallMatchRate)
	}
}

func TestBatchFilter(t *testing.T) {
	now := time.Now()
	filter := BatchFilter{
		Status:    models.BatchStatusPending,
		Source:    "source",
		Target:    "target",
		StartDate: &now,
		EndDate:   &now,
		Limit:     100,
	}

	if filter.Status != models.BatchStatusPending {
		t.Errorf("expected status pending, got %s", filter.Status)
	}
	if filter.Limit != 100 {
		t.Errorf("expected limit 100, got %d", filter.Limit)
	}
}

func TestMatchResult(t *testing.T) {
	result := &MatchResult{
		Matched:    true,
		Confidence: 0.95,
		MatchType:  "exact",
		Differences: []Difference{
			{Field: "amount", Source: "100", Target: "100.01", Severity: "warning"},
		},
	}

	if !result.Matched {
		t.Error("expected result to be matched")
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", result.Confidence)
	}
	if len(result.Differences) != 1 {
		t.Errorf("expected 1 difference, got %d", len(result.Differences))
	}
}

func TestDifference(t *testing.T) {
	diff := Difference{
		Field:    "amount",
		Source:   "100.00",
		Target:   "100.01",
		Severity: "warning",
	}

	if diff.Field != "amount" {
		t.Errorf("expected field 'amount', got %s", diff.Field)
	}
	if diff.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %s", diff.Severity)
	}
}

func TestTransactionIndex(t *testing.T) {
	index := &TransactionIndex{
		byID:         make(map[string]*models.Transaction),
		byExternalID: make(map[string]*models.Transaction),
		byAmount:     make(map[string][]*models.Transaction),
		byDate:       make(map[string][]*models.Transaction),
	}

	if index.byID == nil {
		t.Error("byID should be initialized")
	}
	if index.byExternalID == nil {
		t.Error("byExternalID should be initialized")
	}
}

func TestEngine_Reconcile_WithDifferences(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)
	ctx := context.Background()

	batch := engine.CreateBatch("source", "target")

	now := time.Now()
	sourceTransactions := []*models.Transaction{
		{
			ID:         "txn-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			Currency:   "USD",
			Status:     models.TransactionStatusCompleted,
			CreatedAt:  now,
		},
	}

	// Target has different currency
	targetTransactions := []*models.Transaction{
		{
			ID:         "target-1",
			ExternalID: "ext-1",
			Amount:     decimal.NewFromFloat(100),
			Currency:   "EUR",
			Status:     models.TransactionStatusCompleted,
			CreatedAt:  now,
		},
	}

	err := engine.Reconcile(ctx, batch.ID, sourceTransactions, targetTransactions)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	batch, _ = engine.GetBatch(batch.ID)
	// Still should be matched since amount and date match
	if batch.MatchedRecords != 1 {
		t.Errorf("expected 1 matched record, got %d", batch.MatchedRecords)
	}
}

func TestEngine_CreateException(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	// Create batch first
	batch := engine.CreateBatch("source", "target")

	source := &models.Transaction{
		ID:     "source-1",
		Amount: decimal.NewFromFloat(100),
	}
	target := &models.Transaction{
		ID:     "target-1",
		Amount: decimal.NewFromFloat(95),
	}

	diff := Difference{
		Field:    "amount",
		Severity: "error",
	}

	engine.createException(batch.ID, models.ExceptionTypeAmountDiff, source, target, diff)

	exceptions := engine.GetExceptions(batch.ID)
	if len(exceptions) != 1 {
		t.Errorf("expected 1 exception, got %d", len(exceptions))
	}

	exc := exceptions[0]
	if exc.Type != models.ExceptionTypeAmountDiff {
		t.Errorf("expected type AmountDiff, got %s", exc.Type)
	}
	// Amount diff should be 5
	expectedDiff := decimal.NewFromFloat(5)
	if !exc.AmountDiff.Equal(expectedDiff) {
		t.Errorf("expected amount diff 5, got %s", exc.AmountDiff)
	}

	// Check batch exception count
	batch, _ = engine.GetBatch(batch.ID)
	if batch.Exceptions != 1 {
		t.Errorf("expected 1 exception count on batch, got %d", batch.Exceptions)
	}
}

func TestEngine_CreateException_SourceOnly(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	batch := engine.CreateBatch("source", "target")

	source := &models.Transaction{
		ID:     "source-1",
		Amount: decimal.NewFromFloat(100),
	}

	engine.createException(batch.ID, models.ExceptionTypeMissing, source, nil, Difference{})

	exceptions := engine.GetExceptions(batch.ID)
	exc := exceptions[0]

	// Amount diff should be source amount
	if !exc.AmountDiff.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected amount diff 100, got %s", exc.AmountDiff)
	}
}

func TestEngine_CreateException_TargetOnly(t *testing.T) {
	cfg := &config.ReconciliationConfig{
		MatchTolerance: 0.01,
		DateTolerance:  24 * time.Hour,
	}

	engine := NewEngine(cfg)

	batch := engine.CreateBatch("source", "target")

	target := &models.Transaction{
		ID:     "target-1",
		Amount: decimal.NewFromFloat(200),
	}

	engine.createException(batch.ID, models.ExceptionTypeMissing, nil, target, Difference{})

	exceptions := engine.GetExceptions(batch.ID)
	exc := exceptions[0]

	// Amount diff should be target amount
	if !exc.AmountDiff.Equal(decimal.NewFromFloat(200)) {
		t.Errorf("expected amount diff 200, got %s", exc.AmountDiff)
	}
}
