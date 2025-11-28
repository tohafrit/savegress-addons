package consent

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/pkg/models"
)

func TestNewManager(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:        true,
		DefaultPolicy:   "deny",
		AllowedPurposes: []string{"treatment", "payment"},
	}

	manager := NewManager(cfg)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.config != cfg {
		t.Error("config not set correctly")
	}
	if manager.consents == nil {
		t.Error("consents map not initialized")
	}
	if manager.requests == nil {
		t.Error("requests map not initialized")
	}
}

func TestManager_CreateConsent(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status: "active",
		Patient: &models.Reference{
			Reference: "Patient/patient-123",
		},
	}

	err := manager.CreateConsent(consent)
	if err != nil {
		t.Fatalf("CreateConsent failed: %v", err)
	}

	if consent.ID == "" {
		t.Error("consent ID should be generated")
	}
	if consent.DateTime == nil {
		t.Error("consent DateTime should be set")
	}

	// Check consent was stored
	stored, ok := manager.consents[consent.ID]
	if !ok {
		t.Error("consent should be stored")
	}
	if stored.Status != "active" {
		t.Errorf("expected status 'active', got %s", stored.Status)
	}
}

func TestManager_CreateConsent_WithID(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	consent := &models.Consent{
		FHIRResource: models.FHIRResource{
			ID: "existing-id",
		},
		Status: "active",
	}

	err := manager.CreateConsent(consent)
	if err != nil {
		t.Fatalf("CreateConsent failed: %v", err)
	}

	if consent.ID != "existing-id" {
		t.Errorf("expected ID to remain 'existing-id', got %s", consent.ID)
	}
}

func TestManager_GetConsent(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	consent := &models.Consent{Status: "active"}
	manager.CreateConsent(consent)

	// Get existing consent
	found, ok := manager.GetConsent(consent.ID)
	if !ok {
		t.Error("expected to find consent")
	}
	if found.ID != consent.ID {
		t.Errorf("expected consent ID %s, got %s", consent.ID, found.ID)
	}

	// Get non-existent consent
	_, ok = manager.GetConsent("non-existent")
	if ok {
		t.Error("expected not to find non-existent consent")
	}
}

func TestManager_GetPatientConsents(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	// Create consents for different patients
	consent1 := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
	}
	consent2 := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
	}
	consent3 := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-2"},
	}

	manager.CreateConsent(consent1)
	manager.CreateConsent(consent2)
	manager.CreateConsent(consent3)

	// Get consents for patient-1
	consents := manager.GetPatientConsents("patient-1")
	if len(consents) != 2 {
		t.Errorf("expected 2 consents for patient-1, got %d", len(consents))
	}

	// Get consents for patient-2
	consents = manager.GetPatientConsents("patient-2")
	if len(consents) != 1 {
		t.Errorf("expected 1 consent for patient-2, got %d", len(consents))
	}

	// Get consents for non-existent patient
	consents = manager.GetPatientConsents("patient-3")
	if len(consents) != 0 {
		t.Errorf("expected 0 consents for patient-3, got %d", len(consents))
	}
}

func TestManager_CheckAccess_ConsentNotRequired(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required: false,
	}
	manager := NewManager(cfg)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed when consent not required")
	}
	if result.Reason != "Consent not required" {
		t.Errorf("expected reason 'Consent not required', got %s", result.Reason)
	}
}

func TestManager_CheckAccess_NoConsent_DefaultDeny(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	result := manager.CheckAccess(req)

	if result.Allowed {
		t.Error("expected access to be denied when no consent and default deny")
	}
	if result.Reason != "No consent on file" {
		t.Errorf("expected reason 'No consent on file', got %s", result.Reason)
	}
}

func TestManager_CheckAccess_NoConsent_DefaultPermit(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "permit",
	}
	manager := NewManager(cfg)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed when default permit")
	}
	if result.Reason != "No specific consent, default policy permits" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
}

func TestManager_CheckAccess_AllowedPurpose(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:        true,
		DefaultPolicy:   "deny",
		AllowedPurposes: []string{"treatment", "payment"},
	}
	manager := NewManager(cfg)

	// Create an active consent without specific provision
	consent := &models.Consent{
		Status:    "active",
		Patient:   &models.Reference{Reference: "Patient/patient-1"},
		Provision: nil,
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed for treatment purpose")
	}
}

func TestManager_CheckAccess_WithProvision_Permit(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
			Purpose: []models.Coding{
				{Code: "treatment"},
			},
		},
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed with permit provision")
	}
	if result.ConsentID != consent.ID {
		t.Errorf("expected consent ID %s, got %s", consent.ID, result.ConsentID)
	}
}

func TestManager_CheckAccess_WithProvision_Deny(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "deny",
			Purpose: []models.Coding{
				{Code: "research"},
			},
		},
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "research",
	}

	result := manager.CheckAccess(req)

	if result.Allowed {
		t.Error("expected access to be denied with deny provision")
	}
}

func TestManager_CheckAccess_WithPeriod(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
			Period: &models.Period{
				Start: &start,
				End:   &end,
			},
		},
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID: "patient-1",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed within period")
	}
}

func TestManager_CheckAccess_ExpiredPeriod(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	now := time.Now()
	start := now.Add(-48 * time.Hour)
	end := now.Add(-24 * time.Hour) // Expired

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
			Period: &models.Period{
				Start: &start,
				End:   &end,
			},
		},
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID: "patient-1",
	}

	result := manager.CheckAccess(req)

	if result.Allowed {
		t.Error("expected access to be denied with expired period")
	}
}

func TestManager_CheckAccess_WithClass(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
			Class: []models.Coding{
				{Code: "Observation"},
				{Code: "Condition"},
			},
		},
	}
	manager.CreateConsent(consent)

	// Request for allowed resource type
	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed for Observation")
	}

	// Request for non-allowed resource type
	req2 := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "MedicationRequest",
	}

	result2 := manager.CheckAccess(req2)

	if result2.Allowed {
		t.Error("expected access to be denied for MedicationRequest")
	}
}

func TestManager_CheckAccess_WithAction(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
			Action: []models.CodeableConcept{
				{Coding: []models.Coding{{Code: "read"}}},
			},
		},
	}
	manager.CreateConsent(consent)

	// Request with allowed action
	req := &AccessCheckRequest{
		PatientID: "patient-1",
		Action:    "read",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed for read action")
	}

	// Request with non-allowed action
	req2 := &AccessCheckRequest{
		PatientID: "patient-1",
		Action:    "write",
	}

	result2 := manager.CheckAccess(req2)

	if result2.Allowed {
		t.Error("expected access to be denied for write action")
	}
}

func TestManager_CheckAccess_InactiveConsent(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	consent := &models.Consent{
		Status:  "inactive",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "permit",
		},
	}
	manager.CreateConsent(consent)

	req := &AccessCheckRequest{
		PatientID: "patient-1",
	}

	result := manager.CheckAccess(req)

	if result.Allowed {
		t.Error("expected access to be denied with inactive consent")
	}
}

func TestManager_RevokeConsent(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	consent := &models.Consent{Status: "active"}
	manager.CreateConsent(consent)

	err := manager.RevokeConsent(consent.ID)
	if err != nil {
		t.Fatalf("RevokeConsent failed: %v", err)
	}

	// Check status changed
	stored, _ := manager.GetConsent(consent.ID)
	if stored.Status != "inactive" {
		t.Errorf("expected status 'inactive', got %s", stored.Status)
	}
}

func TestManager_RevokeConsent_NotFound(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	err := manager.RevokeConsent("non-existent")
	if err != ErrConsentNotFound {
		t.Errorf("expected ErrConsentNotFound, got %v", err)
	}
}

func TestManager_CreateAccessRequest(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req := &models.AccessRequest{
		RequestorID:  "user-1",
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
	}

	err := manager.CreateAccessRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateAccessRequest failed: %v", err)
	}

	if req.ID == "" {
		t.Error("request ID should be generated")
	}
	if req.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", req.Status)
	}
	if req.RequestedAt.IsZero() {
		t.Error("RequestedAt should be set")
	}
}

func TestManager_GetAccessRequest(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req := &models.AccessRequest{
		RequestorID: "user-1",
		PatientID:   "patient-1",
	}
	manager.CreateAccessRequest(context.Background(), req)

	// Get existing request
	found, ok := manager.GetAccessRequest(req.ID)
	if !ok {
		t.Error("expected to find request")
	}
	if found.ID != req.ID {
		t.Errorf("expected request ID %s, got %s", req.ID, found.ID)
	}

	// Get non-existent request
	_, ok = manager.GetAccessRequest("non-existent")
	if ok {
		t.Error("expected not to find non-existent request")
	}
}

func TestManager_ApproveAccessRequest(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req := &models.AccessRequest{
		RequestorID: "user-1",
		PatientID:   "patient-1",
	}
	manager.CreateAccessRequest(context.Background(), req)

	err := manager.ApproveAccessRequest(req.ID, "approver-1", 30)
	if err != nil {
		t.Fatalf("ApproveAccessRequest failed: %v", err)
	}

	found, _ := manager.GetAccessRequest(req.ID)
	if found.Status != "approved" {
		t.Errorf("expected status 'approved', got %s", found.Status)
	}
	if found.ApprovedBy != "approver-1" {
		t.Errorf("expected approved by 'approver-1', got %s", found.ApprovedBy)
	}
	if found.ApprovedAt == nil {
		t.Error("ApprovedAt should be set")
	}
	if found.ExpiresAt == nil {
		t.Error("ExpiresAt should be set")
	}
}

func TestManager_ApproveAccessRequest_NoExpiration(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req := &models.AccessRequest{
		RequestorID: "user-1",
		PatientID:   "patient-1",
	}
	manager.CreateAccessRequest(context.Background(), req)

	err := manager.ApproveAccessRequest(req.ID, "approver-1", 0)
	if err != nil {
		t.Fatalf("ApproveAccessRequest failed: %v", err)
	}

	found, _ := manager.GetAccessRequest(req.ID)
	if found.ExpiresAt != nil {
		t.Error("ExpiresAt should not be set when expirationDays is 0")
	}
}

func TestManager_ApproveAccessRequest_NotFound(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	err := manager.ApproveAccessRequest("non-existent", "approver-1", 30)
	if err != ErrRequestNotFound {
		t.Errorf("expected ErrRequestNotFound, got %v", err)
	}
}

func TestManager_DenyAccessRequest(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req := &models.AccessRequest{
		RequestorID: "user-1",
		PatientID:   "patient-1",
	}
	manager.CreateAccessRequest(context.Background(), req)

	err := manager.DenyAccessRequest(req.ID, "Not authorized")
	if err != nil {
		t.Fatalf("DenyAccessRequest failed: %v", err)
	}

	found, _ := manager.GetAccessRequest(req.ID)
	if found.Status != "denied" {
		t.Errorf("expected status 'denied', got %s", found.Status)
	}
}

func TestManager_DenyAccessRequest_NotFound(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	err := manager.DenyAccessRequest("non-existent", "reason")
	if err != ErrRequestNotFound {
		t.Errorf("expected ErrRequestNotFound, got %v", err)
	}
}

func TestManager_GetPendingRequests(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	req1 := &models.AccessRequest{RequestorID: "user-1", PatientID: "patient-1"}
	req2 := &models.AccessRequest{RequestorID: "user-2", PatientID: "patient-2"}
	req3 := &models.AccessRequest{RequestorID: "user-3", PatientID: "patient-3"}

	manager.CreateAccessRequest(context.Background(), req1)
	manager.CreateAccessRequest(context.Background(), req2)
	manager.CreateAccessRequest(context.Background(), req3)

	// Approve one
	manager.ApproveAccessRequest(req2.ID, "approver", 30)

	pending := manager.GetPendingRequests()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending requests, got %d", len(pending))
	}
}

func TestManager_GetStats(t *testing.T) {
	cfg := &config.ConsentConfig{}
	manager := NewManager(cfg)

	// Create consents
	consent1 := &models.Consent{Status: "active"}
	consent2 := &models.Consent{Status: "active"}
	consent3 := &models.Consent{Status: "inactive"}
	manager.CreateConsent(consent1)
	manager.CreateConsent(consent2)
	manager.CreateConsent(consent3)

	// Create requests
	req1 := &models.AccessRequest{RequestorID: "user-1"}
	req2 := &models.AccessRequest{RequestorID: "user-2"}
	manager.CreateAccessRequest(context.Background(), req1)
	manager.CreateAccessRequest(context.Background(), req2)
	manager.ApproveAccessRequest(req2.ID, "approver", 30)

	stats := manager.GetStats()

	if stats.TotalConsents != 3 {
		t.Errorf("expected 3 total consents, got %d", stats.TotalConsents)
	}
	if stats.ActiveConsents != 2 {
		t.Errorf("expected 2 active consents, got %d", stats.ActiveConsents)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", stats.TotalRequests)
	}
	if stats.PendingRequests != 1 {
		t.Errorf("expected 1 pending request, got %d", stats.PendingRequests)
	}
	if stats.ByStatus["active"] != 2 {
		t.Errorf("expected 2 active in ByStatus, got %d", stats.ByStatus["active"])
	}
}

func TestAccessCheckRequest(t *testing.T) {
	req := &AccessCheckRequest{
		PatientID:    "patient-1",
		ResourceType: "Observation",
		ResourceID:   "obs-1",
		RequestorID:  "user-1",
		Purpose:      "treatment",
		Action:       "read",
	}

	if req.PatientID != "patient-1" {
		t.Errorf("expected patient ID 'patient-1', got %s", req.PatientID)
	}
	if req.Purpose != "treatment" {
		t.Errorf("expected purpose 'treatment', got %s", req.Purpose)
	}
}

func TestAccessCheckResult(t *testing.T) {
	result := &AccessCheckResult{
		Allowed:      true,
		ConsentID:    "consent-1",
		Reason:       "Consent granted",
		PatientID:    "patient-1",
		ResourceType: "Observation",
		Purpose:      "treatment",
		CheckedAt:    time.Now(),
	}

	if !result.Allowed {
		t.Error("expected allowed to be true")
	}
	if result.ConsentID != "consent-1" {
		t.Errorf("expected consent ID 'consent-1', got %s", result.ConsentID)
	}
}

func TestConsentStats(t *testing.T) {
	stats := &ConsentStats{
		TotalConsents:   10,
		ActiveConsents:  8,
		TotalRequests:   5,
		PendingRequests: 2,
		ByStatus:        map[string]int{"active": 8, "inactive": 2},
	}

	if stats.TotalConsents != 10 {
		t.Errorf("expected 10 total consents, got %d", stats.TotalConsents)
	}
}

func TestError(t *testing.T) {
	err := &Error{Code: "TEST_ERROR", Message: "test message"}
	if err.Error() != "test message" {
		t.Errorf("expected 'test message', got %s", err.Error())
	}
}

func TestErrConsentNotFound(t *testing.T) {
	if ErrConsentNotFound.Code != "CONSENT_NOT_FOUND" {
		t.Errorf("expected code 'CONSENT_NOT_FOUND', got %s", ErrConsentNotFound.Code)
	}
}

func TestErrRequestNotFound(t *testing.T) {
	if ErrRequestNotFound.Code != "REQUEST_NOT_FOUND" {
		t.Errorf("expected code 'REQUEST_NOT_FOUND', got %s", ErrRequestNotFound.Code)
	}
}

func TestManager_CheckAccess_NestedProvision(t *testing.T) {
	cfg := &config.ConsentConfig{
		Required:      true,
		DefaultPolicy: "deny",
	}
	manager := NewManager(cfg)

	// Parent provision denies "research" but has nested provision that permits "treatment"
	consent := &models.Consent{
		Status:  "active",
		Patient: &models.Reference{Reference: "Patient/patient-1"},
		Provision: &models.ConsentProvision{
			Type: "deny",
			Purpose: []models.Coding{
				{Code: "research"}, // Only denies research
			},
			Provision: []models.ConsentProvision{
				{
					Type: "permit",
					Purpose: []models.Coding{
						{Code: "treatment"},
					},
				},
			},
		},
	}
	manager.CreateConsent(consent)

	// Request for treatment - parent deny doesn't match (wrong purpose), nested permit should match
	req := &AccessCheckRequest{
		PatientID: "patient-1",
		Purpose:   "treatment",
	}

	result := manager.CheckAccess(req)

	if !result.Allowed {
		t.Error("expected access to be allowed from nested provision")
	}
}
