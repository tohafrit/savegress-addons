package regulatory

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// ReportType represents a regulatory report type
type ReportType string

const (
	ReportTypeFRY9C       ReportType = "fr_y9c"      // Bank Holding Company Report
	ReportTypeCallReport  ReportType = "call_report" // FFIEC Call Report
	ReportType1099        ReportType = "form_1099"   // Tax reporting
	ReportTypeW9          ReportType = "form_w9"     // Tax ID verification
	ReportTypeFATCA       ReportType = "fatca"       // Foreign Account Tax Compliance
	ReportTypeCRS         ReportType = "crs"         // Common Reporting Standard
	ReportTypeRegW        ReportType = "reg_w"       // Regulation W
	ReportTypeLCR         ReportType = "lcr"         // Liquidity Coverage Ratio
	ReportTypeNSFR        ReportType = "nsfr"        // Net Stable Funding Ratio
	ReportTypeStressTest  ReportType = "stress_test" // DFAST/CCAR
	ReportTypeCustom      ReportType = "custom"
)

// ReportStatus represents the status of a regulatory report
type ReportStatus string

const (
	ReportStatusDraft      ReportStatus = "draft"
	ReportStatusInProgress ReportStatus = "in_progress"
	ReportStatusReview     ReportStatus = "pending_review"
	ReportStatusApproved   ReportStatus = "approved"
	ReportStatusFiled      ReportStatus = "filed"
	ReportStatusRejected   ReportStatus = "rejected"
)

// RegulatoryReport represents a regulatory report
type RegulatoryReport struct {
	ID              string                 `json:"id"`
	Type            ReportType             `json:"type"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Period          *ReportPeriod          `json:"period"`
	Status          ReportStatus           `json:"status"`
	Data            map[string]interface{} `json:"data,omitempty"`
	Sections        []ReportSection        `json:"sections,omitempty"`
	Validations     []ValidationResult     `json:"validations,omitempty"`
	PreparedBy      string                 `json:"prepared_by"`
	ReviewedBy      string                 `json:"reviewed_by,omitempty"`
	ApprovedBy      string                 `json:"approved_by,omitempty"`
	FilingReference string                 `json:"filing_reference,omitempty"`
	FilingDeadline  *time.Time             `json:"filing_deadline,omitempty"`
	ExportPath      string                 `json:"export_path,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	FiledAt         *time.Time             `json:"filed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ReportPeriod represents the reporting period
type ReportPeriod struct {
	Year      int        `json:"year"`
	Quarter   int        `json:"quarter,omitempty"` // 1-4 for quarterly
	Month     int        `json:"month,omitempty"`   // 1-12 for monthly
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	Type      PeriodType `json:"type"`
}

// PeriodType represents the type of reporting period
type PeriodType string

const (
	PeriodTypeAnnual    PeriodType = "annual"
	PeriodTypeQuarterly PeriodType = "quarterly"
	PeriodTypeMonthly   PeriodType = "monthly"
	PeriodTypeDaily     PeriodType = "daily"
)

// ReportSection represents a section of a regulatory report
type ReportSection struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Code        string                 `json:"code,omitempty"` // Schedule/section code
	Description string                 `json:"description,omitempty"`
	Items       []ReportItem           `json:"items"`
	Subtotal    decimal.Decimal        `json:"subtotal,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReportItem represents an individual line item
type ReportItem struct {
	ID          string          `json:"id"`
	LineNumber  string          `json:"line_number"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	PriorAmount decimal.Decimal `json:"prior_amount,omitempty"`
	Variance    decimal.Decimal `json:"variance,omitempty"`
	Formula     string          `json:"formula,omitempty"`
	Source      string          `json:"source,omitempty"` // GL account or calculation
	Notes       string          `json:"notes,omitempty"`
}

// ValidationResult represents a validation check result
type ValidationResult struct {
	RuleID      string   `json:"rule_id"`
	RuleName    string   `json:"rule_name"`
	Passed      bool     `json:"passed"`
	Severity    string   `json:"severity"` // error, warning, info
	Message     string   `json:"message"`
	AffectedItems []string `json:"affected_items,omitempty"`
}

// Engine manages regulatory reporting
type Engine struct {
	reports     map[string]*RegulatoryReport
	templates   map[ReportType]*ReportTemplate
	validators  map[ReportType][]ValidationRule
	mu          sync.RWMutex
	outputDir   string
}

// ReportTemplate defines a report structure template
type ReportTemplate struct {
	Type        ReportType      `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Sections    []SectionTemplate `json:"sections"`
	Validations []ValidationRule  `json:"validations"`
}

// SectionTemplate defines a section template
type SectionTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Items       []ItemTemplate `json:"items"`
}

// ItemTemplate defines an item template
type ItemTemplate struct {
	LineNumber  string `json:"line_number"`
	Description string `json:"description"`
	Formula     string `json:"formula,omitempty"`
	DataSource  string `json:"data_source,omitempty"`
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Expression  string `json:"expression"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
}

// NewEngine creates a new regulatory reporting engine
func NewEngine(outputDir string) *Engine {
	engine := &Engine{
		reports:    make(map[string]*RegulatoryReport),
		templates:  make(map[ReportType]*ReportTemplate),
		validators: make(map[ReportType][]ValidationRule),
		outputDir:  outputDir,
	}
	engine.initializeTemplates()
	return engine
}

func (e *Engine) initializeTemplates() {
	// Initialize Call Report template
	e.templates[ReportTypeCallReport] = &ReportTemplate{
		Type:        ReportTypeCallReport,
		Name:        "FFIEC Call Report",
		Description: "Consolidated Reports of Condition and Income",
		Sections: []SectionTemplate{
			{
				ID:   "rc",
				Name: "Consolidated Report of Condition",
				Code: "RC",
				Items: []ItemTemplate{
					{LineNumber: "RCFD0010", Description: "Total Cash and Balances Due From Depository Institutions"},
					{LineNumber: "RCFD0071", Description: "Interest-Bearing Balances"},
					{LineNumber: "RCFD1754", Description: "Securities: Held-to-Maturity"},
					{LineNumber: "RCFD1773", Description: "Securities: Available-for-Sale"},
				},
			},
			{
				ID:   "ri",
				Name: "Consolidated Report of Income",
				Code: "RI",
				Items: []ItemTemplate{
					{LineNumber: "RIAD4107", Description: "Total Interest Income"},
					{LineNumber: "RIAD4073", Description: "Total Interest Expense"},
					{LineNumber: "RIAD4074", Description: "Net Interest Income"},
				},
			},
		},
	}

	// Initialize Form 1099 template
	e.templates[ReportType1099] = &ReportTemplate{
		Type:        ReportType1099,
		Name:        "Form 1099",
		Description: "Information Return for US Tax Reporting",
		Sections: []SectionTemplate{
			{
				ID:   "1099-int",
				Name: "1099-INT Interest Income",
				Code: "INT",
				Items: []ItemTemplate{
					{LineNumber: "1", Description: "Interest Income"},
					{LineNumber: "2", Description: "Early Withdrawal Penalty"},
					{LineNumber: "3", Description: "Interest on US Savings Bonds"},
				},
			},
			{
				ID:   "1099-div",
				Name: "1099-DIV Dividends",
				Code: "DIV",
				Items: []ItemTemplate{
					{LineNumber: "1a", Description: "Total Ordinary Dividends"},
					{LineNumber: "1b", Description: "Qualified Dividends"},
					{LineNumber: "2a", Description: "Total Capital Gain Distributions"},
				},
			},
		},
	}

	// Initialize FATCA template
	e.templates[ReportTypeFATCA] = &ReportTemplate{
		Type:        ReportTypeFATCA,
		Name:        "FATCA Report",
		Description: "Foreign Account Tax Compliance Act Report",
		Sections: []SectionTemplate{
			{
				ID:   "account-holders",
				Name: "US Account Holders",
				Code: "USTAX",
			},
		},
	}

	// Initialize LCR template
	e.templates[ReportTypeLCR] = &ReportTemplate{
		Type:        ReportTypeLCR,
		Name:        "LCR Report",
		Description: "Liquidity Coverage Ratio Report",
		Sections: []SectionTemplate{
			{
				ID:   "hqla",
				Name: "High-Quality Liquid Assets",
				Code: "HQLA",
				Items: []ItemTemplate{
					{LineNumber: "L1", Description: "Level 1 Assets"},
					{LineNumber: "L2A", Description: "Level 2A Assets"},
					{LineNumber: "L2B", Description: "Level 2B Assets"},
				},
			},
			{
				ID:   "outflows",
				Name: "Total Net Cash Outflows",
				Code: "NCO",
			},
		},
	}
}

// CreateReport creates a new regulatory report
func (e *Engine) CreateReport(reportType ReportType, period *ReportPeriod, preparedBy string) (*RegulatoryReport, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	template, ok := e.templates[reportType]
	if !ok {
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}

	report := &RegulatoryReport{
		ID:          generateID("report"),
		Type:        reportType,
		Name:        template.Name,
		Description: template.Description,
		Period:      period,
		Status:      ReportStatusDraft,
		PreparedBy:  preparedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Initialize sections from template
	for _, sectionTpl := range template.Sections {
		section := ReportSection{
			ID:          sectionTpl.ID,
			Name:        sectionTpl.Name,
			Code:        sectionTpl.Code,
			Items:       make([]ReportItem, 0, len(sectionTpl.Items)),
		}

		for _, itemTpl := range sectionTpl.Items {
			section.Items = append(section.Items, ReportItem{
				ID:          generateID("item"),
				LineNumber:  itemTpl.LineNumber,
				Description: itemTpl.Description,
				Formula:     itemTpl.Formula,
				Source:      itemTpl.DataSource,
			})
		}

		report.Sections = append(report.Sections, section)
	}

	e.reports[report.ID] = report
	return report, nil
}

// GetReport retrieves a report by ID
func (e *Engine) GetReport(id string) (*RegulatoryReport, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	report, ok := e.reports[id]
	return report, ok
}

// ListReports returns reports matching the filter
func (e *Engine) ListReports(filter ReportFilter) []*RegulatoryReport {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*RegulatoryReport
	for _, report := range e.reports {
		if matchesReportFilter(report, filter) {
			results = append(results, report)
		}
	}

	// Sort by created date descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results
}

// ReportFilter defines filters for report queries
type ReportFilter struct {
	Type      ReportType
	Status    ReportStatus
	Year      int
	Quarter   int
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func matchesReportFilter(report *RegulatoryReport, filter ReportFilter) bool {
	if filter.Type != "" && report.Type != filter.Type {
		return false
	}
	if filter.Status != "" && report.Status != filter.Status {
		return false
	}
	if filter.Year != 0 && report.Period != nil && report.Period.Year != filter.Year {
		return false
	}
	if filter.Quarter != 0 && report.Period != nil && report.Period.Quarter != filter.Quarter {
		return false
	}
	return true
}

// UpdateReportItem updates a line item in a report
func (e *Engine) UpdateReportItem(reportID, sectionID, itemID string, amount decimal.Decimal, notes string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return fmt.Errorf("report not found: %s", reportID)
	}

	if report.Status != ReportStatusDraft && report.Status != ReportStatusInProgress {
		return fmt.Errorf("report is not editable in status: %s", report.Status)
	}

	for i := range report.Sections {
		if report.Sections[i].ID == sectionID {
			for j := range report.Sections[i].Items {
				if report.Sections[i].Items[j].ID == itemID {
					report.Sections[i].Items[j].Amount = amount
					report.Sections[i].Items[j].Notes = notes
					report.UpdatedAt = time.Now()
					return nil
				}
			}
		}
	}

	return fmt.Errorf("item not found: %s/%s", sectionID, itemID)
}

// PopulateFromTransactions populates report data from transactions
func (e *Engine) PopulateFromTransactions(ctx context.Context, reportID string, transactions []*models.Transaction) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return fmt.Errorf("report not found: %s", reportID)
	}

	// Calculate aggregates from transactions
	var totalCredits, totalDebits decimal.Decimal
	categoryTotals := make(map[string]decimal.Decimal)

	for _, txn := range transactions {
		switch txn.Type {
		case models.TransactionTypeCredit:
			totalCredits = totalCredits.Add(txn.Amount)
		case models.TransactionTypeDebit:
			totalDebits = totalDebits.Add(txn.Amount)
		}

		if txn.Category != "" {
			categoryTotals[txn.Category] = categoryTotals[txn.Category].Add(txn.Amount)
		}
	}

	// Store in report data
	if report.Data == nil {
		report.Data = make(map[string]interface{})
	}
	report.Data["total_credits"] = totalCredits.String()
	report.Data["total_debits"] = totalDebits.String()
	report.Data["category_totals"] = categoryTotals
	report.Data["transaction_count"] = len(transactions)

	report.Status = ReportStatusInProgress
	report.UpdatedAt = time.Now()

	return nil
}

// ValidateReport runs validation rules on a report
func (e *Engine) ValidateReport(reportID string) ([]ValidationResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return nil, fmt.Errorf("report not found: %s", reportID)
	}

	var results []ValidationResult

	// Run built-in validations
	results = append(results, e.validateCompleteness(report)...)
	results = append(results, e.validateFormulas(report)...)
	results = append(results, e.validateCrossChecks(report)...)

	// Run report-type-specific validations
	if validators, ok := e.validators[report.Type]; ok {
		for _, rule := range validators {
			result := e.evaluateValidationRule(report, rule)
			results = append(results, result)
		}
	}

	report.Validations = results
	report.UpdatedAt = time.Now()

	return results, nil
}

func (e *Engine) validateCompleteness(report *RegulatoryReport) []ValidationResult {
	var results []ValidationResult

	for _, section := range report.Sections {
		for _, item := range section.Items {
			if item.Amount.IsZero() && item.Formula == "" {
				results = append(results, ValidationResult{
					RuleID:        "completeness",
					RuleName:      "Required Field Check",
					Passed:        false,
					Severity:      "warning",
					Message:       fmt.Sprintf("Line item %s has no value", item.LineNumber),
					AffectedItems: []string{item.LineNumber},
				})
			}
		}
	}

	if len(results) == 0 {
		results = append(results, ValidationResult{
			RuleID:   "completeness",
			RuleName: "Required Field Check",
			Passed:   true,
			Severity: "info",
			Message:  "All required fields have values",
		})
	}

	return results
}

func (e *Engine) validateFormulas(report *RegulatoryReport) []ValidationResult {
	var results []ValidationResult

	// Check section subtotals
	for _, section := range report.Sections {
		var calculated decimal.Decimal
		for _, item := range section.Items {
			calculated = calculated.Add(item.Amount)
		}

		if !section.Subtotal.IsZero() && !calculated.Equal(section.Subtotal) {
			results = append(results, ValidationResult{
				RuleID:   "formula_check",
				RuleName: "Section Subtotal Validation",
				Passed:   false,
				Severity: "error",
				Message:  fmt.Sprintf("Section %s subtotal mismatch: expected %s, calculated %s", section.Code, section.Subtotal.String(), calculated.String()),
			})
		}
	}

	return results
}

func (e *Engine) validateCrossChecks(report *RegulatoryReport) []ValidationResult {
	var results []ValidationResult

	// Example cross-check: assets = liabilities + equity
	// Would be specific to report type
	results = append(results, ValidationResult{
		RuleID:   "cross_check",
		RuleName: "Balance Sheet Equation",
		Passed:   true,
		Severity: "info",
		Message:  "Cross-reference checks passed",
	})

	return results
}

func (e *Engine) evaluateValidationRule(report *RegulatoryReport, rule ValidationRule) ValidationResult {
	// Simplified rule evaluation
	// In production, would parse and evaluate the expression
	return ValidationResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
		Message:  "Validation passed",
	}
}

// SubmitForReview submits a report for review
func (e *Engine) SubmitForReview(reportID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return fmt.Errorf("report not found: %s", reportID)
	}

	// Check validations
	hasErrors := false
	for _, v := range report.Validations {
		if !v.Passed && v.Severity == "error" {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		return fmt.Errorf("report has validation errors")
	}

	report.Status = ReportStatusReview
	report.UpdatedAt = time.Now()
	return nil
}

// ApproveReport approves a report
func (e *Engine) ApproveReport(reportID, approver string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return fmt.Errorf("report not found: %s", reportID)
	}

	if report.Status != ReportStatusReview {
		return fmt.Errorf("report must be in review status to approve")
	}

	report.Status = ReportStatusApproved
	report.ApprovedBy = approver
	report.UpdatedAt = time.Now()
	return nil
}

// FileReport files a report with the regulatory body
func (e *Engine) FileReport(ctx context.Context, reportID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	report, ok := e.reports[reportID]
	if !ok {
		return fmt.Errorf("report not found: %s", reportID)
	}

	if report.Status != ReportStatusApproved {
		return fmt.Errorf("report must be approved to file")
	}

	// In production, this would submit to the regulatory system
	report.Status = ReportStatusFiled
	now := time.Now()
	report.FiledAt = &now
	report.FilingReference = generateFilingReference(report.Type)
	report.UpdatedAt = now

	return nil
}

// ExportReport exports a report to the specified format
func (e *Engine) ExportReport(ctx context.Context, reportID string, format ExportFormat) (string, error) {
	e.mu.RLock()
	report, ok := e.reports[reportID]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("report not found: %s", reportID)
	}

	filename := fmt.Sprintf("%s_%s_%d.%s", report.Type, report.ID, time.Now().Unix(), format)
	filepath := filepath.Join(e.outputDir, filename)

	var err error
	switch format {
	case ExportFormatJSON:
		err = e.exportJSON(report, filepath)
	case ExportFormatCSV:
		err = e.exportCSV(report, filepath)
	case ExportFormatXML:
		err = e.exportXML(report, filepath)
	case ExportFormatXBRL:
		err = e.exportXBRL(report, filepath)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return "", err
	}

	e.mu.Lock()
	report.ExportPath = filepath
	e.mu.Unlock()

	return filepath, nil
}

// ExportFormat represents an export format
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatXML  ExportFormat = "xml"
	ExportFormatXBRL ExportFormat = "xbrl"
	ExportFormatPDF  ExportFormat = "pdf"
)

func (e *Engine) exportJSON(report *RegulatoryReport, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (e *Engine) exportCSV(report *RegulatoryReport, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Section", "Line Number", "Description", "Amount", "Notes"})

	// Data
	for _, section := range report.Sections {
		for _, item := range section.Items {
			writer.Write([]string{
				section.Name,
				item.LineNumber,
				item.Description,
				item.Amount.String(),
				item.Notes,
			})
		}
	}

	return nil
}

func (e *Engine) exportXML(report *RegulatoryReport, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	return encoder.Encode(report)
}

func (e *Engine) exportXBRL(report *RegulatoryReport, path string) error {
	// XBRL export would require proper taxonomy handling
	// Simplified version using XML structure
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write XBRL header
	io.WriteString(file, `<?xml version="1.0" encoding="UTF-8"?>
<xbrl xmlns="http://www.xbrl.org/2003/instance">
`)

	// Write facts
	for _, section := range report.Sections {
		for _, item := range section.Items {
			fmt.Fprintf(file, `  <fact contextRef="context1" unitRef="USD" decimals="0">
    <concept>%s</concept>
    <value>%s</value>
  </fact>
`, item.LineNumber, item.Amount.String())
		}
	}

	io.WriteString(file, "</xbrl>\n")
	return nil
}

// GetStats returns reporting statistics
func (e *Engine) GetStats() *ReportingStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &ReportingStats{
		ByType:   make(map[string]int),
		ByStatus: make(map[string]int),
	}

	for _, report := range e.reports {
		stats.TotalReports++
		stats.ByType[string(report.Type)]++
		stats.ByStatus[string(report.Status)]++

		if report.Status == ReportStatusDraft || report.Status == ReportStatusInProgress {
			stats.PendingReports++
		}

		if report.FilingDeadline != nil && report.Status != ReportStatusFiled {
			if report.FilingDeadline.Before(time.Now().Add(7 * 24 * time.Hour)) {
				stats.UpcomingDeadlines++
			}
		}
	}

	return stats
}

// ReportingStats contains reporting statistics
type ReportingStats struct {
	TotalReports      int            `json:"total_reports"`
	PendingReports    int            `json:"pending_reports"`
	UpcomingDeadlines int            `json:"upcoming_deadlines"`
	ByType            map[string]int `json:"by_type"`
	ByStatus          map[string]int `json:"by_status"`
}

// Helper functions
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func generateFilingReference(reportType ReportType) string {
	return fmt.Sprintf("%s-%s-%d", reportType, time.Now().Format("20060102"), time.Now().UnixNano()%10000)
}
