package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/finsight/internal/fraud"
	"github.com/savegress/finsight/internal/reconciliation"
	"github.com/savegress/finsight/internal/reporting"
	"github.com/savegress/finsight/internal/transactions"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	transactions *transactions.Engine
	fraud        *fraud.Detector
	reconcile    *reconciliation.Engine
	reports      *reporting.Generator
}

// NewHandlers creates new handlers
func NewHandlers(txn *transactions.Engine, fr *fraud.Detector, recon *reconciliation.Engine, rpt *reporting.Generator) *Handlers {
	return &Handlers{
		transactions: txn,
		fraud:        fr,
		reconcile:    recon,
		reports:      rpt,
	}
}

// HealthCheck handles health check requests
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "finsight",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Transaction handlers

// ListTransactions lists all transactions
func (h *Handlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	filter := transactions.TransactionFilter{
		Limit: 100,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = models.TransactionStatus(status)
	}
	if txnType := r.URL.Query().Get("type"); txnType != "" {
		filter.Type = models.TransactionType(txnType)
	}
	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = category
	}

	txns := h.transactions.GetTransactions(filter)
	respond(w, http.StatusOK, txns)
}

// CreateTransaction creates a new transaction
func (h *Handlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var txn models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if txn.ID == "" {
		txn.ID = generateID("txn")
	}
	txn.CreatedAt = time.Now()
	txn.Status = models.TransactionStatusPending

	if err := h.transactions.ProcessTransaction(r.Context(), &txn); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, txn)
}

// GetTransaction gets a transaction by ID
func (h *Handlers) GetTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	txn, ok := h.transactions.GetTransaction(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	respond(w, http.StatusOK, txn)
}

// UpdateTransaction updates a transaction
func (h *Handlers) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, ok := h.transactions.GetTransaction(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	var update models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	update.ID = id
	if err := h.transactions.ProcessTransaction(r.Context(), &update); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, update)
}

// GetTransactionStats gets transaction statistics
func (h *Handlers) GetTransactionStats(w http.ResponseWriter, r *http.Request) {
	stats := h.transactions.GetStats()
	respond(w, http.StatusOK, stats)
}

// Account handlers

// ListAccounts lists all accounts
func (h *Handlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	// Placeholder - would need proper account listing
	respond(w, http.StatusOK, []models.Account{})
}

// CreateAccount creates a new account
func (h *Handlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var acc models.Account
	if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if acc.ID == "" {
		acc.ID = generateID("acc")
	}
	acc.Balance = decimal.Zero
	acc.AvailableBal = decimal.Zero
	acc.HoldAmount = decimal.Zero
	acc.Status = models.AccountStatusActive

	if err := h.transactions.CreateAccount(&acc); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, acc)
}

// GetAccount gets an account by ID
func (h *Handlers) GetAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	acc, ok := h.transactions.GetAccount(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	respond(w, http.StatusOK, acc)
}

// GetAccountTransactions gets transactions for an account
func (h *Handlers) GetAccountTransactions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	txns := h.transactions.GetAccountTransactions(id, 100)
	respond(w, http.StatusOK, txns)
}

// GetAccountBalance gets account balance
func (h *Handlers) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	acc, ok := h.transactions.GetAccount(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"account_id":        acc.ID,
		"balance":           acc.Balance,
		"available_balance": acc.AvailableBal,
		"hold_amount":       acc.HoldAmount,
		"currency":          acc.Currency,
	})
}

// Fraud handlers

// ListFraudAlerts lists fraud alerts
func (h *Handlers) ListFraudAlerts(w http.ResponseWriter, r *http.Request) {
	filter := fraud.AlertFilter{
		Limit: 100,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = models.AlertStatus(status)
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = models.AlertSeverity(severity)
	}

	alerts := h.fraud.GetAlerts(filter)
	respond(w, http.StatusOK, alerts)
}

// GetFraudAlert gets a fraud alert by ID
func (h *Handlers) GetFraudAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	alert, ok := h.fraud.GetAlert(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Alert not found")
		return
	}

	respond(w, http.StatusOK, alert)
}

// ResolveFraudAlert resolves a fraud alert
func (h *Handlers) ResolveFraudAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Resolution    string `json:"resolution"`
		FalsePositive bool   `json:"false_positive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.fraud.ResolveAlert(id, req.Resolution, req.FalsePositive); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// EvaluateTransaction evaluates a transaction for fraud
func (h *Handlers) EvaluateTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transaction models.Transaction      `json:"transaction"`
		Context     *fraud.EvaluationContext `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result := h.fraud.Evaluate(&req.Transaction, req.Context)
	respond(w, http.StatusOK, result)
}

// GetFraudStats gets fraud statistics
func (h *Handlers) GetFraudStats(w http.ResponseWriter, r *http.Request) {
	stats := h.fraud.GetStats()
	respond(w, http.StatusOK, stats)
}

// Reconciliation handlers

// ListReconcileBatches lists reconciliation batches
func (h *Handlers) ListReconcileBatches(w http.ResponseWriter, r *http.Request) {
	filter := reconciliation.BatchFilter{
		Limit: 100,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = models.BatchStatus(status)
	}

	batches := h.reconcile.GetBatches(filter)
	respond(w, http.StatusOK, batches)
}

// CreateReconcileBatch creates a reconciliation batch
func (h *Handlers) CreateReconcileBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	batch := h.reconcile.CreateBatch(req.Source, req.Target)
	respond(w, http.StatusCreated, batch)
}

// GetReconcileBatch gets a reconciliation batch
func (h *Handlers) GetReconcileBatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	batch, ok := h.reconcile.GetBatch(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Batch not found")
		return
	}

	respond(w, http.StatusOK, batch)
}

// RunReconciliation runs reconciliation for a batch
func (h *Handlers) RunReconciliation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		SourceTransactions []*models.Transaction `json:"source_transactions"`
		TargetTransactions []*models.Transaction `json:"target_transactions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.reconcile.Reconcile(r.Context(), id, req.SourceTransactions, req.TargetTransactions); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	batch, _ := h.reconcile.GetBatch(id)
	respond(w, http.StatusOK, batch)
}

// GetBatchExceptions gets exceptions for a batch
func (h *Handlers) GetBatchExceptions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	exceptions := h.reconcile.GetExceptions(id)
	respond(w, http.StatusOK, exceptions)
}

// ResolveException resolves a reconciliation exception
func (h *Handlers) ResolveException(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Resolution string `json:"resolution"`
		WriteOff   bool   `json:"write_off"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.reconcile.ResolveException(id, req.Resolution, req.WriteOff); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// GetReconcileStats gets reconciliation statistics
func (h *Handlers) GetReconcileStats(w http.ResponseWriter, r *http.Request) {
	stats := h.reconcile.GetStats()
	respond(w, http.StatusOK, stats)
}

// Report handlers

// ListReports lists all reports
func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	filter := reporting.ReportFilter{
		Limit: 100,
	}

	if reportType := r.URL.Query().Get("type"); reportType != "" {
		filter.Type = models.ReportType(reportType)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = models.ReportStatus(status)
	}

	reports := h.reports.GetReports(filter)
	respond(w, http.StatusOK, reports)
}

// CreateReport creates a new report
func (h *Handlers) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type      models.ReportType   `json:"type"`
		Period    models.ReportPeriod `json:"period"`
		StartDate time.Time           `json:"start_date"`
		EndDate   time.Time           `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	report := h.reports.CreateReport(req.Type, req.Period, req.StartDate, req.EndDate)
	respond(w, http.StatusCreated, report)
}

// GetReport gets a report by ID
func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	report, ok := h.reports.GetReport(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Report not found")
		return
	}

	respond(w, http.StatusOK, report)
}

// GenerateReport generates a report
func (h *Handlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	report, ok := h.reports.GetReport(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Report not found")
		return
	}

	// Get transactions for the report period
	filter := transactions.TransactionFilter{
		StartDate: &report.StartDate,
		EndDate:   &report.EndDate,
	}
	txns := h.transactions.GetTransactions(filter)

	var err error
	switch report.Type {
	case models.ReportTypeTransaction:
		err = h.reports.GenerateTransactionReport(r.Context(), id, txns)
	case models.ReportTypeCashFlow:
		err = h.reports.GenerateCashFlowReport(r.Context(), id, txns)
	case models.ReportTypeFraud:
		alerts := h.fraud.GetAlerts(fraud.AlertFilter{})
		err = h.reports.GenerateFraudReport(r.Context(), id, alerts, txns)
	default:
		err = h.reports.GenerateTransactionReport(r.Context(), id, txns)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	report, _ = h.reports.GetReport(id)
	respond(w, http.StatusOK, report)
}

// DeleteReport deletes a report
func (h *Handlers) DeleteReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.reports.DeleteReport(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetOverallStats gets overall system statistics
func (h *Handlers) GetOverallStats(w http.ResponseWriter, r *http.Request) {
	txnStats := h.transactions.GetStats()
	fraudStats := h.fraud.GetStats()
	reconStats := h.reconcile.GetStats()

	respond(w, http.StatusOK, map[string]interface{}{
		"transactions":   txnStats,
		"fraud":          fraudStats,
		"reconciliation": reconStats,
	})
}

// Helper functions

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405")
}
