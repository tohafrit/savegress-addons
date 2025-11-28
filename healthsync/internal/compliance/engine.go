package compliance

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/pkg/models"
)

// Engine handles HIPAA compliance checks
type Engine struct {
	config     *config.ComplianceConfig
	violations map[string]*models.ComplianceViolation
	phiFields  map[models.ResourceType][]models.PHIField
	validators []Validator
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
}

// Validator defines a compliance validator
type Validator interface {
	Name() string
	Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult
}

// ValidationResult contains the result of a validation
type ValidationResult struct {
	Valid       bool
	ViolationType string
	Severity    string
	Field       string
	Description string
	Remediation string
}

// NewEngine creates a new compliance engine
func NewEngine(cfg *config.ComplianceConfig) *Engine {
	e := &Engine{
		config:     cfg,
		violations: make(map[string]*models.ComplianceViolation),
		phiFields:  make(map[models.ResourceType][]models.PHIField),
		stopCh:     make(chan struct{}),
	}
	e.initializePHIFields()
	e.initializeValidators()
	return e
}

func (e *Engine) initializePHIFields() {
	// Patient PHI fields
	e.phiFields[models.ResourceTypePatient] = []models.PHIField{
		{FieldName: "name", FieldPath: "name", PHICategory: models.PHICategoryName, Sensitivity: "high"},
		{FieldName: "birthDate", FieldPath: "birthDate", PHICategory: models.PHICategoryDates, Sensitivity: "high"},
		{FieldName: "address", FieldPath: "address", PHICategory: models.PHICategoryAddress, Sensitivity: "high"},
		{FieldName: "telecom.phone", FieldPath: "telecom[?system='phone']", PHICategory: models.PHICategoryPhone, Sensitivity: "high"},
		{FieldName: "telecom.email", FieldPath: "telecom[?system='email']", PHICategory: models.PHICategoryEmail, Sensitivity: "high"},
		{FieldName: "identifier.ssn", FieldPath: "identifier[?system='ssn']", PHICategory: models.PHICategorySSN, Sensitivity: "critical"},
		{FieldName: "identifier.mrn", FieldPath: "identifier[?system='mrn']", PHICategory: models.PHICategoryMRN, Sensitivity: "high"},
		{FieldName: "photo", FieldPath: "photo", PHICategory: models.PHICategoryPhoto, Sensitivity: "high"},
	}

	// Observation PHI fields
	e.phiFields[models.ResourceTypeObservation] = []models.PHIField{
		{FieldName: "subject", FieldPath: "subject", PHICategory: models.PHICategoryOther, Sensitivity: "medium"},
		{FieldName: "performer", FieldPath: "performer", PHICategory: models.PHICategoryOther, Sensitivity: "low"},
		{FieldName: "effectiveDateTime", FieldPath: "effectiveDateTime", PHICategory: models.PHICategoryDates, Sensitivity: "medium"},
	}

	// Encounter PHI fields
	e.phiFields[models.ResourceTypeEncounter] = []models.PHIField{
		{FieldName: "subject", FieldPath: "subject", PHICategory: models.PHICategoryOther, Sensitivity: "medium"},
		{FieldName: "period", FieldPath: "period", PHICategory: models.PHICategoryDates, Sensitivity: "medium"},
		{FieldName: "participant", FieldPath: "participant", PHICategory: models.PHICategoryOther, Sensitivity: "low"},
	}
}

func (e *Engine) initializeValidators() {
	e.validators = []Validator{
		NewMinimumNecessaryValidator(e.config.MinimumNecessary),
		NewPHIExposureValidator(e.phiFields),
		NewDataIntegrityValidator(),
		NewAccessControlValidator(),
		NewEncryptionValidator(),
	}
}

// Start starts the compliance engine
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

// Stop stops the compliance engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

// ValidateResource validates a FHIR resource for compliance
func (e *Engine) ValidateResource(resource interface{}, resourceType models.ResourceType) *ComplianceResult {
	if !e.config.HIPAAEnabled {
		return &ComplianceResult{
			Valid:      true,
			ResourceType: resourceType,
			CheckedAt:  time.Now(),
		}
	}

	result := &ComplianceResult{
		ResourceType: resourceType,
		CheckedAt:    time.Now(),
	}

	// Run all validators
	for _, validator := range e.validators {
		validationResults := validator.Validate(resource, resourceType)
		for _, vr := range validationResults {
			if !vr.Valid {
				result.Violations = append(result.Violations, vr)
			}
		}
	}

	result.Valid = len(result.Violations) == 0
	return result
}

// ComplianceResult contains the result of compliance validation
type ComplianceResult struct {
	Valid        bool                `json:"valid"`
	ResourceType models.ResourceType `json:"resource_type"`
	Violations   []ValidationResult  `json:"violations,omitempty"`
	CheckedAt    time.Time           `json:"checked_at"`
}

// RecordViolation records a compliance violation
func (e *Engine) RecordViolation(violation *models.ComplianceViolation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if violation.ID == "" {
		violation.ID = uuid.New().String()
	}
	violation.DetectedAt = time.Now()
	violation.Status = "open"

	e.violations[violation.ID] = violation
}

// GetViolation retrieves a violation by ID
func (e *Engine) GetViolation(id string) (*models.ComplianceViolation, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	v, ok := e.violations[id]
	return v, ok
}

// GetViolations retrieves violations with filters
func (e *Engine) GetViolations(filter ViolationFilter) []*models.ComplianceViolation {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*models.ComplianceViolation
	for _, v := range e.violations {
		if e.matchesViolationFilter(v, filter) {
			results = append(results, v)
		}
	}
	return results
}

// ViolationFilter defines filters for violation queries
type ViolationFilter struct {
	Type     string
	Severity string
	Status   string
	Resource string
	Limit    int
}

func (e *Engine) matchesViolationFilter(v *models.ComplianceViolation, filter ViolationFilter) bool {
	if filter.Type != "" && v.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && v.Severity != filter.Severity {
		return false
	}
	if filter.Status != "" && v.Status != filter.Status {
		return false
	}
	if filter.Resource != "" && v.Resource != filter.Resource {
		return false
	}
	return true
}

// ResolveViolation resolves a violation
func (e *Engine) ResolveViolation(id string, resolution string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	v, ok := e.violations[id]
	if !ok {
		return ErrViolationNotFound
	}

	now := time.Now()
	v.ResolvedAt = &now
	v.Status = "resolved"

	return nil
}

// GetPHIFields returns PHI fields for a resource type
func (e *Engine) GetPHIFields(resourceType models.ResourceType) []models.PHIField {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.phiFields[resourceType]
}

// CheckMinimumNecessary checks if data access follows minimum necessary principle
func (e *Engine) CheckMinimumNecessary(requestedFields []string, purpose string, resourceType models.ResourceType) *MinimumNecessaryResult {
	phiFields := e.phiFields[resourceType]

	result := &MinimumNecessaryResult{
		Purpose:    purpose,
		Compliant:  true,
		Allowed:    make([]string, 0),
		Restricted: make([]string, 0),
	}

	allowedCategories := e.getAllowedCategories(purpose)

	for _, field := range requestedFields {
		for _, phi := range phiFields {
			if phi.FieldName == field {
				if e.isCategoryAllowed(phi.PHICategory, allowedCategories) {
					result.Allowed = append(result.Allowed, field)
				} else {
					result.Restricted = append(result.Restricted, field)
					result.Compliant = false
				}
				break
			}
		}
	}

	return result
}

// MinimumNecessaryResult contains the result of minimum necessary check
type MinimumNecessaryResult struct {
	Purpose    string   `json:"purpose"`
	Compliant  bool     `json:"compliant"`
	Allowed    []string `json:"allowed"`
	Restricted []string `json:"restricted"`
}

func (e *Engine) getAllowedCategories(purpose string) []string {
	switch purpose {
	case "treatment":
		return []string{
			models.PHICategoryName,
			models.PHICategoryDates,
			models.PHICategoryPhone,
			models.PHICategoryMRN,
			models.PHICategoryAddress,
		}
	case "payment":
		return []string{
			models.PHICategoryName,
			models.PHICategoryAddress,
			models.PHICategoryDates,
			models.PHICategoryHealthPlan,
			models.PHICategoryAccount,
		}
	case "operations":
		return []string{
			models.PHICategoryMRN,
			models.PHICategoryDates,
		}
	case "research":
		// Research typically requires de-identified data
		return []string{}
	default:
		return []string{}
	}
}

func (e *Engine) isCategoryAllowed(category string, allowed []string) bool {
	for _, a := range allowed {
		if a == category {
			return true
		}
	}
	return false
}

// GetStats returns compliance statistics
func (e *Engine) GetStats() *ComplianceStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &ComplianceStats{
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
		ByStatus:   make(map[string]int),
	}

	for _, v := range e.violations {
		stats.TotalViolations++
		stats.ByType[v.Type]++
		stats.BySeverity[v.Severity]++
		stats.ByStatus[v.Status]++

		if v.Status == "open" {
			stats.OpenViolations++
		}
	}

	return stats
}

// ComplianceStats contains compliance statistics
type ComplianceStats struct {
	TotalViolations int            `json:"total_violations"`
	OpenViolations  int            `json:"open_violations"`
	ByType          map[string]int `json:"by_type"`
	BySeverity      map[string]int `json:"by_severity"`
	ByStatus        map[string]int `json:"by_status"`
}

// ScanResourceForPHI scans a resource and identifies PHI
func (e *Engine) ScanResourceForPHI(resource interface{}, resourceType models.ResourceType) *PHIScanResult {
	result := &PHIScanResult{
		ResourceType: resourceType,
		ScannedAt:    time.Now(),
		PHIFound:     make([]PHIMatch, 0),
	}

	// Convert resource to map for scanning
	data, err := json.Marshal(resource)
	if err != nil {
		return result
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return result
	}

	// Check each known PHI field
	for _, field := range e.phiFields[resourceType] {
		if value, exists := resourceMap[field.FieldName]; exists && value != nil {
			result.PHIFound = append(result.PHIFound, PHIMatch{
				Field:       field.FieldName,
				Category:    field.PHICategory,
				Sensitivity: field.Sensitivity,
				HasValue:    true,
			})
		}
	}

	result.TotalPHIFields = len(result.PHIFound)
	return result
}

// PHIScanResult contains the result of PHI scanning
type PHIScanResult struct {
	ResourceType   models.ResourceType `json:"resource_type"`
	TotalPHIFields int                 `json:"total_phi_fields"`
	PHIFound       []PHIMatch          `json:"phi_found"`
	ScannedAt      time.Time           `json:"scanned_at"`
}

// PHIMatch represents a found PHI field
type PHIMatch struct {
	Field       string `json:"field"`
	Category    string `json:"category"`
	Sensitivity string `json:"sensitivity"`
	HasValue    bool   `json:"has_value"`
}

// Errors
var (
	ErrViolationNotFound = &Error{Code: "VIOLATION_NOT_FOUND", Message: "Violation not found"}
)

// Error represents a compliance error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
