package compliance

import (
	"context"
	"testing"

	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/pkg/models"
)

func TestNewEngine(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled:     true,
		MinimumNecessary: true,
	}

	engine := NewEngine(cfg)

	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	if engine.config != cfg {
		t.Error("config not set correctly")
	}

	// Check PHI fields were initialized
	patientFields := engine.GetPHIFields(models.ResourceTypePatient)
	if len(patientFields) == 0 {
		t.Error("expected patient PHI fields to be initialized")
	}

	// Check validators were initialized
	if len(engine.validators) != 5 {
		t.Errorf("expected 5 validators, got %d", len(engine.validators))
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: true,
	}
	engine := NewEngine(cfg)

	// Start engine
	err := engine.Start(context.Background())
	if err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}

	if !engine.running {
		t.Error("engine should be running")
	}

	// Starting again should be no-op
	err = engine.Start(context.Background())
	if err != nil {
		t.Fatalf("second start should not fail: %v", err)
	}

	// Stop engine
	engine.Stop()

	if engine.running {
		t.Error("engine should not be running after stop")
	}

	// Stopping again should be safe (no panic)
	engine.Stop()
}

func TestEngine_ValidateResource_HIPAADisabled(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: false,
	}
	engine := NewEngine(cfg)

	resource := map[string]interface{}{
		"resourceType": "Patient",
	}

	result := engine.ValidateResource(resource, models.ResourceTypePatient)

	if !result.Valid {
		t.Error("expected validation to pass when HIPAA is disabled")
	}

	if len(result.Violations) > 0 {
		t.Error("expected no violations when HIPAA is disabled")
	}
}

func TestEngine_ValidateResource_WithViolations(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled:     true,
		MinimumNecessary: true,
	}
	engine := NewEngine(cfg)

	// Patient without identifier should trigger data integrity violation
	resource := map[string]interface{}{
		"resourceType": "Patient",
		"name": []interface{}{
			map[string]interface{}{
				"family": "Doe",
				"given":  []interface{}{"John"},
			},
		},
	}

	result := engine.ValidateResource(resource, models.ResourceTypePatient)

	if result.Valid {
		t.Error("expected validation to fail for patient without identifier")
	}

	// Check for data integrity violation
	hasIdentifierViolation := false
	for _, v := range result.Violations {
		if v.Field == "identifier" {
			hasIdentifierViolation = true
			break
		}
	}

	if !hasIdentifierViolation {
		t.Error("expected identifier violation")
	}
}

func TestEngine_ValidateResource_Observation(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: true,
	}
	engine := NewEngine(cfg)

	// Observation without code and status
	resource := map[string]interface{}{
		"resourceType": "Observation",
		"subject": map[string]interface{}{
			"reference": "Patient/123",
		},
	}

	result := engine.ValidateResource(resource, models.ResourceTypeObservation)

	if result.Valid {
		t.Error("expected validation to fail for observation without code/status")
	}

	// Should have violations for missing code and status
	violationCount := 0
	for _, v := range result.Violations {
		if v.Field == "code" || v.Field == "status" {
			violationCount++
		}
	}

	if violationCount < 2 {
		t.Errorf("expected at least 2 violations for code and status, got %d", violationCount)
	}
}

func TestEngine_ValidateResource_Encounter(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: true,
	}
	engine := NewEngine(cfg)

	// Encounter without class
	resource := map[string]interface{}{
		"resourceType": "Encounter",
		"subject": map[string]interface{}{
			"reference": "Patient/123",
		},
	}

	result := engine.ValidateResource(resource, models.ResourceTypeEncounter)

	if result.Valid {
		t.Error("expected validation to fail for encounter without class")
	}
}

func TestEngine_RecordViolation(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: true,
	}
	engine := NewEngine(cfg)

	violation := &models.ComplianceViolation{
		Type:     "phi_exposure",
		Severity: "high",
		Resource: "Patient/123",
	}

	engine.RecordViolation(violation)

	if violation.ID == "" {
		t.Error("expected violation ID to be set")
	}

	if violation.Status != "open" {
		t.Error("expected violation status to be 'open'")
	}

	if violation.DetectedAt.IsZero() {
		t.Error("expected DetectedAt to be set")
	}

	// Retrieve the violation
	retrieved, ok := engine.GetViolation(violation.ID)
	if !ok {
		t.Error("expected to find recorded violation")
	}

	if retrieved.Type != "phi_exposure" {
		t.Errorf("expected type 'phi_exposure', got %s", retrieved.Type)
	}
}

func TestEngine_RecordViolation_WithExistingID(t *testing.T) {
	cfg := &config.ComplianceConfig{
		HIPAAEnabled: true,
	}
	engine := NewEngine(cfg)

	violation := &models.ComplianceViolation{
		ID:       "existing-id",
		Type:     "data_integrity",
		Severity: "medium",
	}

	engine.RecordViolation(violation)

	// ID should remain unchanged
	if violation.ID != "existing-id" {
		t.Errorf("expected ID to remain 'existing-id', got %s", violation.ID)
	}

	retrieved, ok := engine.GetViolation("existing-id")
	if !ok {
		t.Error("expected to find violation by existing ID")
	}

	if retrieved.Type != "data_integrity" {
		t.Error("violation not retrieved correctly")
	}
}

func TestEngine_GetViolation_NotFound(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	_, ok := engine.GetViolation("nonexistent")

	if ok {
		t.Error("expected violation not to be found")
	}
}

func TestEngine_GetViolations_NoFilter(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	// Add multiple violations
	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "high"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "data_integrity", Severity: "medium"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "access_control", Severity: "low"})

	violations := engine.GetViolations(ViolationFilter{})

	if len(violations) != 3 {
		t.Errorf("expected 3 violations, got %d", len(violations))
	}
}

func TestEngine_GetViolations_FilterByType(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "high"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "medium"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "data_integrity", Severity: "high"})

	violations := engine.GetViolations(ViolationFilter{Type: "phi_exposure"})

	if len(violations) != 2 {
		t.Errorf("expected 2 phi_exposure violations, got %d", len(violations))
	}

	for _, v := range violations {
		if v.Type != "phi_exposure" {
			t.Errorf("expected type 'phi_exposure', got %s", v.Type)
		}
	}
}

func TestEngine_GetViolations_FilterBySeverity(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "high"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "data_integrity", Severity: "high"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "access_control", Severity: "low"})

	violations := engine.GetViolations(ViolationFilter{Severity: "high"})

	if len(violations) != 2 {
		t.Errorf("expected 2 high severity violations, got %d", len(violations))
	}
}

func TestEngine_GetViolations_FilterByStatus(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	v1 := &models.ComplianceViolation{Type: "phi_exposure"}
	v2 := &models.ComplianceViolation{Type: "data_integrity"}
	engine.RecordViolation(v1)
	engine.RecordViolation(v2)

	// Resolve one violation
	engine.ResolveViolation(v1.ID, "Fixed SSN exposure")

	openViolations := engine.GetViolations(ViolationFilter{Status: "open"})
	if len(openViolations) != 1 {
		t.Errorf("expected 1 open violation, got %d", len(openViolations))
	}

	resolvedViolations := engine.GetViolations(ViolationFilter{Status: "resolved"})
	if len(resolvedViolations) != 1 {
		t.Errorf("expected 1 resolved violation, got %d", len(resolvedViolations))
	}
}

func TestEngine_GetViolations_FilterByResource(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Resource: "Patient/123"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "data_integrity", Resource: "Patient/123"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "access_control", Resource: "Observation/456"})

	violations := engine.GetViolations(ViolationFilter{Resource: "Patient/123"})

	if len(violations) != 2 {
		t.Errorf("expected 2 Patient/123 violations, got %d", len(violations))
	}
}

func TestEngine_GetViolations_MultipleFilters(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "high", Resource: "Patient/123"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "phi_exposure", Severity: "low", Resource: "Patient/123"})
	engine.RecordViolation(&models.ComplianceViolation{Type: "data_integrity", Severity: "high", Resource: "Patient/456"})

	violations := engine.GetViolations(ViolationFilter{
		Type:     "phi_exposure",
		Severity: "high",
	})

	if len(violations) != 1 {
		t.Errorf("expected 1 violation matching both filters, got %d", len(violations))
	}
}

func TestEngine_ResolveViolation(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	violation := &models.ComplianceViolation{
		Type:     "phi_exposure",
		Severity: "high",
	}
	engine.RecordViolation(violation)

	err := engine.ResolveViolation(violation.ID, "Fixed by masking SSN")

	if err != nil {
		t.Fatalf("failed to resolve violation: %v", err)
	}

	resolved, _ := engine.GetViolation(violation.ID)

	if resolved.Status != "resolved" {
		t.Errorf("expected status 'resolved', got %s", resolved.Status)
	}

	if resolved.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
}

func TestEngine_ResolveViolation_NotFound(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	err := engine.ResolveViolation("nonexistent", "reason")

	if err != ErrViolationNotFound {
		t.Errorf("expected ErrViolationNotFound, got %v", err)
	}
}

func TestEngine_GetPHIFields(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	patientFields := engine.GetPHIFields(models.ResourceTypePatient)

	if len(patientFields) == 0 {
		t.Error("expected patient PHI fields")
	}

	// Check specific fields exist
	fieldNames := make(map[string]bool)
	for _, field := range patientFields {
		fieldNames[field.FieldName] = true
	}

	expectedFields := []string{"name", "birthDate", "address", "identifier.ssn"}
	for _, expected := range expectedFields {
		if !fieldNames[expected] {
			t.Errorf("expected field %s not found", expected)
		}
	}

	// Check observation fields
	obsFields := engine.GetPHIFields(models.ResourceTypeObservation)
	if len(obsFields) == 0 {
		t.Error("expected observation PHI fields")
	}

	// Check encounter fields
	encFields := engine.GetPHIFields(models.ResourceTypeEncounter)
	if len(encFields) == 0 {
		t.Error("expected encounter PHI fields")
	}
}

func TestEngine_CheckMinimumNecessary_Treatment(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	requestedFields := []string{"name", "birthDate", "telecom.phone"}
	result := engine.CheckMinimumNecessary(requestedFields, "treatment", models.ResourceTypePatient)

	if !result.Compliant {
		t.Error("expected treatment request to be compliant")
	}

	if result.Purpose != "treatment" {
		t.Errorf("expected purpose 'treatment', got %s", result.Purpose)
	}

	if len(result.Restricted) > 0 {
		t.Errorf("expected no restricted fields for treatment, got %v", result.Restricted)
	}
}

func TestEngine_CheckMinimumNecessary_Research(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	// Research should restrict PHI fields
	requestedFields := []string{"name", "birthDate", "identifier.ssn"}
	result := engine.CheckMinimumNecessary(requestedFields, "research", models.ResourceTypePatient)

	if result.Compliant {
		t.Error("expected research request with PHI to be non-compliant")
	}

	// All fields should be restricted for research
	if len(result.Restricted) < len(requestedFields) {
		t.Errorf("expected all fields to be restricted for research, restricted: %v", result.Restricted)
	}
}

func TestEngine_CheckMinimumNecessary_Payment(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	requestedFields := []string{"name", "address", "birthDate"}
	result := engine.CheckMinimumNecessary(requestedFields, "payment", models.ResourceTypePatient)

	if !result.Compliant {
		t.Error("expected payment request to be compliant for these fields")
	}
}

func TestEngine_CheckMinimumNecessary_Operations(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	// Operations should only allow MRN and dates
	requestedFields := []string{"identifier.mrn", "birthDate"}
	result := engine.CheckMinimumNecessary(requestedFields, "operations", models.ResourceTypePatient)

	// MRN is allowed
	hasMRNAllowed := false
	for _, f := range result.Allowed {
		if f == "identifier.mrn" {
			hasMRNAllowed = true
			break
		}
	}

	if !hasMRNAllowed {
		t.Error("expected MRN to be allowed for operations")
	}
}

func TestEngine_CheckMinimumNecessary_UnknownPurpose(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	requestedFields := []string{"name", "birthDate"}
	result := engine.CheckMinimumNecessary(requestedFields, "unknown", models.ResourceTypePatient)

	// Unknown purpose should restrict all PHI
	if result.Compliant {
		t.Error("expected unknown purpose to be non-compliant")
	}
}

func TestEngine_GetStats(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	// Add violations
	v1 := &models.ComplianceViolation{Type: "phi_exposure", Severity: "high"}
	v2 := &models.ComplianceViolation{Type: "phi_exposure", Severity: "low"}
	v3 := &models.ComplianceViolation{Type: "data_integrity", Severity: "medium"}

	engine.RecordViolation(v1)
	engine.RecordViolation(v2)
	engine.RecordViolation(v3)

	// Resolve one
	engine.ResolveViolation(v1.ID, "fixed")

	stats := engine.GetStats()

	if stats.TotalViolations != 3 {
		t.Errorf("expected 3 total violations, got %d", stats.TotalViolations)
	}

	if stats.OpenViolations != 2 {
		t.Errorf("expected 2 open violations, got %d", stats.OpenViolations)
	}

	if stats.ByType["phi_exposure"] != 2 {
		t.Errorf("expected 2 phi_exposure violations, got %d", stats.ByType["phi_exposure"])
	}

	if stats.BySeverity["high"] != 1 {
		t.Errorf("expected 1 high severity violation, got %d", stats.BySeverity["high"])
	}

	if stats.ByStatus["resolved"] != 1 {
		t.Errorf("expected 1 resolved violation, got %d", stats.ByStatus["resolved"])
	}
}

func TestEngine_ScanResourceForPHI(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"name": []interface{}{
			map[string]interface{}{
				"family": "Doe",
			},
		},
		"birthDate": "1990-01-01",
		"address": []interface{}{
			map[string]interface{}{
				"city": "Boston",
			},
		},
	}

	result := engine.ScanResourceForPHI(resource, models.ResourceTypePatient)

	if result.ResourceType != models.ResourceTypePatient {
		t.Errorf("expected resource type Patient, got %s", result.ResourceType)
	}

	if result.TotalPHIFields == 0 {
		t.Error("expected PHI fields to be found")
	}

	if result.ScannedAt.IsZero() {
		t.Error("expected ScannedAt to be set")
	}

	// Check that name was found
	hasName := false
	for _, phi := range result.PHIFound {
		if phi.Field == "name" {
			hasName = true
			if phi.Category != models.PHICategoryName {
				t.Errorf("expected name category to be %s", models.PHICategoryName)
			}
		}
	}

	if !hasName {
		t.Error("expected name field to be found in PHI scan")
	}
}

func TestEngine_ScanResourceForPHI_InvalidJSON(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	// Function type can't be marshaled to JSON
	resource := func() {}

	result := engine.ScanResourceForPHI(resource, models.ResourceTypePatient)

	// Should return empty result without panic
	if result.TotalPHIFields != 0 {
		t.Error("expected no PHI fields for invalid resource")
	}
}

func TestEngine_ScanResourceForPHI_NilValues(t *testing.T) {
	cfg := &config.ComplianceConfig{}
	engine := NewEngine(cfg)

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"name":         nil,
		"birthDate":    nil,
	}

	result := engine.ScanResourceForPHI(resource, models.ResourceTypePatient)

	// nil values should not be counted as PHI
	for _, phi := range result.PHIFound {
		if phi.Field == "name" || phi.Field == "birthDate" {
			t.Errorf("nil field %s should not be counted as PHI", phi.Field)
		}
	}
}

func TestErrViolationNotFound(t *testing.T) {
	if ErrViolationNotFound.Code != "VIOLATION_NOT_FOUND" {
		t.Errorf("expected code 'VIOLATION_NOT_FOUND', got %s", ErrViolationNotFound.Code)
	}

	if ErrViolationNotFound.Error() != "Violation not found" {
		t.Errorf("expected message 'Violation not found', got %s", ErrViolationNotFound.Error())
	}
}
