package reporting

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Generator generates financial reports
type Generator struct {
	config  *config.ReportingConfig
	reports map[string]*models.FinancialReport
	mu      sync.RWMutex
}

// NewGenerator creates a new report generator
func NewGenerator(cfg *config.ReportingConfig) *Generator {
	return &Generator{
		config:  cfg,
		reports: make(map[string]*models.FinancialReport),
	}
}

// CreateReport creates a new report
func (g *Generator) CreateReport(reportType models.ReportType, period models.ReportPeriod, startDate, endDate time.Time) *models.FinancialReport {
	g.mu.Lock()
	defer g.mu.Unlock()

	report := &models.FinancialReport{
		ID:        generateReportID(),
		Type:      reportType,
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    models.ReportStatusPending,
	}

	g.reports[report.ID] = report
	return report
}

// GenerateTransactionReport generates a transaction report
func (g *Generator) GenerateTransactionReport(ctx context.Context, reportID string, transactions []*models.Transaction) error {
	g.mu.Lock()
	report, ok := g.reports[reportID]
	if !ok {
		g.mu.Unlock()
		return ErrReportNotFound
	}
	report.Status = models.ReportStatusGenerating
	g.mu.Unlock()

	data := &models.ReportData{
		ByType:     make(map[string]models.TypeSummary),
		ByCategory: make(map[string]decimal.Decimal),
		ByStatus:   make(map[string]int),
	}

	// Filter transactions by date range
	var filtered []*models.Transaction
	for _, txn := range transactions {
		if !txn.CreatedAt.Before(report.StartDate) && !txn.CreatedAt.After(report.EndDate) {
			filtered = append(filtered, txn)
		}
	}

	// Calculate totals
	var credits, debits decimal.Decimal
	dailyData := make(map[string]*dailyAccumulator)
	merchantData := make(map[string]*merchantAccumulator)

	for _, txn := range filtered {
		data.TotalTransactions++
		data.TotalVolume = data.TotalVolume.Add(txn.Amount)

		// By type
		typeName := string(txn.Type)
		if _, ok := data.ByType[typeName]; !ok {
			data.ByType[typeName] = models.TypeSummary{}
		}
		ts := data.ByType[typeName]
		ts.Count++
		ts.Volume = ts.Volume.Add(txn.Amount)
		data.ByType[typeName] = ts

		// By category
		if txn.Category != "" {
			data.ByCategory[txn.Category] = data.ByCategory[txn.Category].Add(txn.Amount)
		}

		// By status
		data.ByStatus[string(txn.Status)]++

		// Credits vs Debits
		switch txn.Type {
		case models.TransactionTypeCredit, models.TransactionTypeRefund, models.TransactionTypeInterest:
			credits = credits.Add(txn.Amount)
		case models.TransactionTypeDebit, models.TransactionTypeFee:
			debits = debits.Add(txn.Amount)
		}

		// Daily breakdown
		day := txn.CreatedAt.Format("2006-01-02")
		if dailyData[day] == nil {
			dailyData[day] = &dailyAccumulator{
				date: txn.CreatedAt.Truncate(24 * time.Hour),
			}
		}
		dailyData[day].transactions++
		dailyData[day].volume = dailyData[day].volume.Add(txn.Amount)
		switch txn.Type {
		case models.TransactionTypeCredit:
			dailyData[day].credits = dailyData[day].credits.Add(txn.Amount)
		case models.TransactionTypeDebit:
			dailyData[day].debits = dailyData[day].debits.Add(txn.Amount)
		}

		// Top merchants
		if txn.Merchant != nil && txn.Merchant.ID != "" {
			if merchantData[txn.Merchant.ID] == nil {
				merchantData[txn.Merchant.ID] = &merchantAccumulator{
					id:   txn.Merchant.ID,
					name: txn.Merchant.Name,
				}
			}
			merchantData[txn.Merchant.ID].transactions++
			merchantData[txn.Merchant.ID].volume = merchantData[txn.Merchant.ID].volume.Add(txn.Amount)
		}
	}

	// Calculate averages
	for typeName, ts := range data.ByType {
		if ts.Count > 0 {
			ts.Average = ts.Volume.Div(decimal.NewFromInt(int64(ts.Count)))
			data.ByType[typeName] = ts
		}
	}

	// Net flow
	data.NetFlow = credits.Sub(debits)

	// Daily breakdown
	for _, acc := range dailyData {
		data.DailyBreakdown = append(data.DailyBreakdown, models.DailySummary{
			Date:         acc.date,
			Transactions: acc.transactions,
			Volume:       acc.volume,
			Credits:      acc.credits,
			Debits:       acc.debits,
		})
	}
	sort.Slice(data.DailyBreakdown, func(i, j int) bool {
		return data.DailyBreakdown[i].Date.Before(data.DailyBreakdown[j].Date)
	})

	// Top merchants
	var merchantList []*merchantAccumulator
	for _, acc := range merchantData {
		merchantList = append(merchantList, acc)
	}
	sort.Slice(merchantList, func(i, j int) bool {
		return merchantList[i].volume.GreaterThan(merchantList[j].volume)
	})
	for i, acc := range merchantList {
		if i >= 10 {
			break
		}
		data.TopMerchants = append(data.TopMerchants, models.MerchantSummary{
			MerchantID:   acc.id,
			MerchantName: acc.name,
			Transactions: acc.transactions,
			Volume:       acc.volume,
		})
	}

	// Update report
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	report.Data = data
	report.GeneratedAt = &now
	report.Status = models.ReportStatusCompleted

	return nil
}

type dailyAccumulator struct {
	date         time.Time
	transactions int
	volume       decimal.Decimal
	credits      decimal.Decimal
	debits       decimal.Decimal
}

type merchantAccumulator struct {
	id           string
	name         string
	transactions int
	volume       decimal.Decimal
}

// GenerateFraudReport generates a fraud report
func (g *Generator) GenerateFraudReport(ctx context.Context, reportID string, alerts []*models.FraudAlert, transactions []*models.Transaction) error {
	g.mu.Lock()
	report, ok := g.reports[reportID]
	if !ok {
		g.mu.Unlock()
		return ErrReportNotFound
	}
	report.Status = models.ReportStatusGenerating
	g.mu.Unlock()

	data := &models.ReportData{
		ByType:     make(map[string]models.TypeSummary),
		ByCategory: make(map[string]decimal.Decimal),
		ByStatus:   make(map[string]int),
	}

	// Filter alerts by date range
	var filteredAlerts []*models.FraudAlert
	for _, alert := range alerts {
		if !alert.CreatedAt.Before(report.StartDate) && !alert.CreatedAt.After(report.EndDate) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	// Calculate fraud metrics
	metrics := &models.FraudMetrics{}
	var blockedAmount decimal.Decimal

	for _, alert := range filteredAlerts {
		metrics.TotalAlerts++

		switch alert.Status {
		case models.AlertStatusOpen, models.AlertStatusInProgress:
			metrics.OpenAlerts++
		case models.AlertStatusResolved:
			metrics.ResolvedAlerts++
		case models.AlertStatusFalsePos:
			metrics.FalsePositives++
		}

		// Find associated transaction
		for _, txn := range transactions {
			if txn.ID == alert.TransactionID {
				if alert.Status == models.AlertStatusResolved {
					blockedAmount = blockedAmount.Add(txn.Amount)
				}
				break
			}
		}

		// By alert type
		data.ByStatus[string(alert.AlertType)]++
	}

	metrics.BlockedAmount = blockedAmount
	if metrics.TotalAlerts > 0 {
		metrics.DetectionRate = float64(metrics.ResolvedAlerts) / float64(metrics.TotalAlerts)
	}

	data.FraudMetrics = metrics
	data.TotalTransactions = len(filteredAlerts)

	// Update report
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	report.Data = data
	report.GeneratedAt = &now
	report.Status = models.ReportStatusCompleted

	return nil
}

// GenerateCashFlowReport generates a cash flow report
func (g *Generator) GenerateCashFlowReport(ctx context.Context, reportID string, transactions []*models.Transaction) error {
	g.mu.Lock()
	report, ok := g.reports[reportID]
	if !ok {
		g.mu.Unlock()
		return ErrReportNotFound
	}
	report.Status = models.ReportStatusGenerating
	g.mu.Unlock()

	data := &models.ReportData{
		ByType:     make(map[string]models.TypeSummary),
		ByCategory: make(map[string]decimal.Decimal),
		ByStatus:   make(map[string]int),
	}

	// Filter transactions
	var filtered []*models.Transaction
	for _, txn := range transactions {
		if !txn.CreatedAt.Before(report.StartDate) && !txn.CreatedAt.After(report.EndDate) {
			filtered = append(filtered, txn)
		}
	}

	var inflow, outflow decimal.Decimal
	dailyFlow := make(map[string]*cashFlowAccumulator)

	for _, txn := range filtered {
		data.TotalTransactions++
		data.TotalVolume = data.TotalVolume.Add(txn.Amount)

		day := txn.CreatedAt.Format("2006-01-02")
		if dailyFlow[day] == nil {
			dailyFlow[day] = &cashFlowAccumulator{
				date: txn.CreatedAt.Truncate(24 * time.Hour),
			}
		}

		switch txn.Type {
		case models.TransactionTypeCredit, models.TransactionTypeRefund, models.TransactionTypeInterest:
			inflow = inflow.Add(txn.Amount)
			dailyFlow[day].inflow = dailyFlow[day].inflow.Add(txn.Amount)
		case models.TransactionTypeDebit, models.TransactionTypeFee, models.TransactionTypeTransfer:
			outflow = outflow.Add(txn.Amount)
			dailyFlow[day].outflow = dailyFlow[day].outflow.Add(txn.Amount)
		}
	}

	data.NetFlow = inflow.Sub(outflow)

	// Build daily breakdown
	for _, acc := range dailyFlow {
		data.DailyBreakdown = append(data.DailyBreakdown, models.DailySummary{
			Date:    acc.date,
			Credits: acc.inflow,
			Debits:  acc.outflow,
			Volume:  acc.inflow.Add(acc.outflow),
		})
	}
	sort.Slice(data.DailyBreakdown, func(i, j int) bool {
		return data.DailyBreakdown[i].Date.Before(data.DailyBreakdown[j].Date)
	})

	// By type summaries
	data.ByType["inflow"] = models.TypeSummary{Volume: inflow}
	data.ByType["outflow"] = models.TypeSummary{Volume: outflow}
	data.ByType["net"] = models.TypeSummary{Volume: data.NetFlow}

	// Update report
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	report.Data = data
	report.GeneratedAt = &now
	report.Status = models.ReportStatusCompleted

	return nil
}

type cashFlowAccumulator struct {
	date    time.Time
	inflow  decimal.Decimal
	outflow decimal.Decimal
}

// GetReport retrieves a report by ID
func (g *Generator) GetReport(id string) (*models.FinancialReport, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	report, ok := g.reports[id]
	return report, ok
}

// GetReports retrieves reports with filters
func (g *Generator) GetReports(filter ReportFilter) []*models.FinancialReport {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var results []*models.FinancialReport
	for _, report := range g.reports {
		if g.matchesFilter(report, filter) {
			results = append(results, report)
		}
	}

	// Sort by date descending
	sort.Slice(results, func(i, j int) bool {
		if results[i].GeneratedAt == nil {
			return false
		}
		if results[j].GeneratedAt == nil {
			return true
		}
		return results[i].GeneratedAt.After(*results[j].GeneratedAt)
	})

	return results
}

// ReportFilter defines filters for report queries
type ReportFilter struct {
	Type      models.ReportType
	Status    models.ReportStatus
	Period    models.ReportPeriod
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func (g *Generator) matchesFilter(report *models.FinancialReport, filter ReportFilter) bool {
	if filter.Type != "" && report.Type != filter.Type {
		return false
	}
	if filter.Status != "" && report.Status != filter.Status {
		return false
	}
	if filter.Period != "" && report.Period != filter.Period {
		return false
	}
	return true
}

// DeleteReport deletes a report
func (g *Generator) DeleteReport(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.reports[id]; !ok {
		return ErrReportNotFound
	}

	delete(g.reports, id)
	return nil
}

func generateReportID() string {
	return "rpt-" + time.Now().Format("20060102150405")
}

// Errors
var (
	ErrReportNotFound = &Error{Code: "REPORT_NOT_FOUND", Message: "Report not found"}
)

// Error represents a reporting error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
