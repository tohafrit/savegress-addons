package consent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/pkg/models"
)

// Manager handles patient consent management
type Manager struct {
	config   *config.ConsentConfig
	consents map[string]*models.Consent
	requests map[string]*models.AccessRequest
	mu       sync.RWMutex
}

// NewManager creates a new consent manager
func NewManager(cfg *config.ConsentConfig) *Manager {
	return &Manager{
		config:   cfg,
		consents: make(map[string]*models.Consent),
		requests: make(map[string]*models.AccessRequest),
	}
}

// CreateConsent creates a new consent record
func (m *Manager) CreateConsent(consent *models.Consent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if consent.ID == "" {
		consent.ID = uuid.New().String()
	}

	now := time.Now()
	consent.DateTime = &now

	m.consents[consent.ID] = consent
	return nil
}

// GetConsent retrieves a consent by ID
func (m *Manager) GetConsent(id string) (*models.Consent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	consent, ok := m.consents[id]
	return consent, ok
}

// GetPatientConsents retrieves all consents for a patient
func (m *Manager) GetPatientConsents(patientID string) []*models.Consent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*models.Consent
	for _, consent := range m.consents {
		if consent.Patient != nil && consent.Patient.Reference == "Patient/"+patientID {
			results = append(results, consent)
		}
	}
	return results
}

// CheckAccess checks if access is allowed based on consent
func (m *Manager) CheckAccess(req *AccessCheckRequest) *AccessCheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &AccessCheckResult{
		PatientID:    req.PatientID,
		ResourceType: req.ResourceType,
		Purpose:      req.Purpose,
		CheckedAt:    time.Now(),
	}

	// Find applicable consents for this patient
	var applicableConsents []*models.Consent
	for _, consent := range m.consents {
		if consent.Patient != nil && consent.Patient.Reference == "Patient/"+req.PatientID {
			if consent.Status == "active" {
				applicableConsents = append(applicableConsents, consent)
			}
		}
	}

	// If no consent required, allow access
	if !m.config.Required {
		result.Allowed = true
		result.Reason = "Consent not required"
		return result
	}

	// If no consent found, check default policy
	if len(applicableConsents) == 0 {
		if m.config.DefaultPolicy == "permit" {
			result.Allowed = true
			result.Reason = "No specific consent, default policy permits"
		} else {
			result.Allowed = false
			result.Reason = "No consent on file"
		}
		return result
	}

	// Check each consent
	for _, consent := range applicableConsents {
		if m.checkConsentProvision(consent.Provision, req) {
			result.Allowed = true
			result.ConsentID = consent.ID
			result.Reason = "Consent granted"
			return result
		}
	}

	// Check if purpose is in allowed purposes
	for _, allowed := range m.config.AllowedPurposes {
		if allowed == req.Purpose {
			result.Allowed = true
			result.Reason = "Purpose is in allowed list"
			return result
		}
	}

	result.Allowed = false
	result.Reason = "No matching consent provision found"
	return result
}

// AccessCheckRequest contains parameters for access check
type AccessCheckRequest struct {
	PatientID    string
	ResourceType string
	ResourceID   string
	RequestorID  string
	Purpose      string
	Action       string
}

// AccessCheckResult contains the result of access check
type AccessCheckResult struct {
	Allowed      bool      `json:"allowed"`
	ConsentID    string    `json:"consent_id,omitempty"`
	Reason       string    `json:"reason"`
	PatientID    string    `json:"patient_id"`
	ResourceType string    `json:"resource_type"`
	Purpose      string    `json:"purpose"`
	CheckedAt    time.Time `json:"checked_at"`
}

func (m *Manager) checkConsentProvision(provision *models.ConsentProvision, req *AccessCheckRequest) bool {
	if provision == nil {
		return false
	}

	// Check if provision type matches request
	if provision.Type == "deny" {
		// If deny, check if this specific access is denied
		if m.matchesProvision(provision, req) {
			return false
		}
	} else if provision.Type == "permit" {
		// If permit, check if this access is permitted
		if m.matchesProvision(provision, req) {
			return true
		}
	}

	// Check nested provisions
	for _, nested := range provision.Provision {
		if m.checkConsentProvision(&nested, req) {
			return true
		}
	}

	return false
}

func (m *Manager) matchesProvision(provision *models.ConsentProvision, req *AccessCheckRequest) bool {
	// Check period
	if provision.Period != nil {
		now := time.Now()
		if provision.Period.Start != nil && now.Before(*provision.Period.Start) {
			return false
		}
		if provision.Period.End != nil && now.After(*provision.Period.End) {
			return false
		}
	}

	// Check purpose
	if len(provision.Purpose) > 0 {
		purposeMatch := false
		for _, purpose := range provision.Purpose {
			if purpose.Code == req.Purpose {
				purposeMatch = true
				break
			}
		}
		if !purposeMatch {
			return false
		}
	}

	// Check class (resource type)
	if len(provision.Class) > 0 {
		classMatch := false
		for _, class := range provision.Class {
			if class.Code == req.ResourceType {
				classMatch = true
				break
			}
		}
		if !classMatch {
			return false
		}
	}

	// Check action
	if len(provision.Action) > 0 {
		actionMatch := false
		for _, action := range provision.Action {
			for _, coding := range action.Coding {
				if coding.Code == req.Action {
					actionMatch = true
					break
				}
			}
		}
		if !actionMatch {
			return false
		}
	}

	return true
}

// RevokeConsent revokes a consent
func (m *Manager) RevokeConsent(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	consent, ok := m.consents[id]
	if !ok {
		return ErrConsentNotFound
	}

	consent.Status = "inactive"
	return nil
}

// CreateAccessRequest creates an access request
func (m *Manager) CreateAccessRequest(ctx context.Context, req *models.AccessRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	req.RequestedAt = time.Now()
	req.Status = "pending"

	m.requests[req.ID] = req
	return nil
}

// GetAccessRequest retrieves an access request
func (m *Manager) GetAccessRequest(id string) (*models.AccessRequest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.requests[id]
	return req, ok
}

// ApproveAccessRequest approves an access request
func (m *Manager) ApproveAccessRequest(id string, approverID string, expirationDays int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return ErrRequestNotFound
	}

	now := time.Now()
	req.Status = "approved"
	req.ApprovedAt = &now
	req.ApprovedBy = approverID

	if expirationDays > 0 {
		expires := now.AddDate(0, 0, expirationDays)
		req.ExpiresAt = &expires
	}

	return nil
}

// DenyAccessRequest denies an access request
func (m *Manager) DenyAccessRequest(id string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return ErrRequestNotFound
	}

	req.Status = "denied"
	return nil
}

// GetPendingRequests retrieves pending access requests
func (m *Manager) GetPendingRequests() []*models.AccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*models.AccessRequest
	for _, req := range m.requests {
		if req.Status == "pending" {
			results = append(results, req)
		}
	}
	return results
}

// GetStats returns consent statistics
func (m *Manager) GetStats() *ConsentStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ConsentStats{
		ByStatus: make(map[string]int),
	}

	for _, consent := range m.consents {
		stats.TotalConsents++
		stats.ByStatus[consent.Status]++

		if consent.Status == "active" {
			stats.ActiveConsents++
		}
	}

	for _, req := range m.requests {
		stats.TotalRequests++
		if req.Status == "pending" {
			stats.PendingRequests++
		}
	}

	return stats
}

// ConsentStats contains consent statistics
type ConsentStats struct {
	TotalConsents   int            `json:"total_consents"`
	ActiveConsents  int            `json:"active_consents"`
	TotalRequests   int            `json:"total_requests"`
	PendingRequests int            `json:"pending_requests"`
	ByStatus        map[string]int `json:"by_status"`
}

// Errors
var (
	ErrConsentNotFound = &Error{Code: "CONSENT_NOT_FOUND", Message: "Consent not found"}
	ErrRequestNotFound = &Error{Code: "REQUEST_NOT_FOUND", Message: "Access request not found"}
)

// Error represents a consent error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
