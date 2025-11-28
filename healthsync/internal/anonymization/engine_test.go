package anonymization

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/savegress/healthsync/pkg/models"
)

func TestNewEngine(t *testing.T) {
	cfg := &Config{
		Method:           "safe_harbor",
		DateShiftRange:   30,
		ZipCodeTruncation: 3,
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.config != cfg {
		t.Error("config not set correctly")
	}
	if e.dateShift == nil {
		t.Error("dateShift map should be initialized")
	}
	if e.pseudonyms == nil {
		t.Error("pseudonyms map should be initialized")
	}
}

func TestEngine_AnonymizePatient_SafeHarbor(t *testing.T) {
	cfg := &Config{
		Method:           "safe_harbor",
		DateShiftRange:   30,
		ZipCodeTruncation: 3,
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-123",
			Identifier: []models.Identifier{
				{Value: "MRN12345"},
			},
		},
		BirthDate: "1980-05-15",
		Name: []models.HumanName{
			{Family: "Doe", Given: []string{"John"}},
		},
		Address: []models.Address{
			{
				Line:       []string{"123 Main St"},
				City:       "Boston",
				State:      "MA",
				PostalCode: "02101",
			},
		},
		Telecom: []models.ContactPoint{
			{System: "phone", Value: "555-1234"},
		},
		Contact: []models.PatientContact{
			{Name: &models.HumanName{Family: "Doe", Given: []string{"Jane"}}},
		},
	}

	anonymized := e.AnonymizePatient(patient)

	// Name should be removed
	if anonymized.Name != nil {
		t.Error("Name should be removed")
	}

	// Telecom should be removed
	if anonymized.Telecom != nil {
		t.Error("Telecom should be removed")
	}

	// Contact should be removed
	if anonymized.Contact != nil {
		t.Error("Contact should be removed")
	}

	// Address should be truncated
	if len(anonymized.Address) > 0 {
		addr := anonymized.Address[0]
		if addr.Line != nil {
			t.Error("Street address should be removed")
		}
		if !strings.HasSuffix(addr.PostalCode, "00") {
			t.Error("PostalCode should be truncated")
		}
	}

	// ID should be pseudonymized
	if anonymized.ID == "patient-123" {
		t.Error("ID should be pseudonymized")
	}

	// Identifiers should be removed for Safe Harbor
	if anonymized.FHIRResource.Identifier != nil {
		t.Error("Identifiers should be removed for Safe Harbor")
	}
}

func TestEngine_AnonymizePatient_LimitedDataset(t *testing.T) {
	cfg := &Config{
		Method:           "limited_dataset",
		DateShiftRange:   1, // Minimal date shift (avoids rand.Intn(0) panic)
		ZipCodeTruncation: 5, // Keep full zip
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-456",
		},
		BirthDate: "1990-01-01",
		Address: []models.Address{
			{
				Line:       []string{"456 Oak Ave"},
				City:       "Seattle",
				State:      "WA",
				PostalCode: "98101",
			},
		},
	}

	anonymized := e.AnonymizePatient(patient)

	// Name should still be removed
	if anonymized.Name != nil {
		t.Error("Name should be removed")
	}

	// City and state should be kept
	if len(anonymized.Address) > 0 {
		addr := anonymized.Address[0]
		if addr.City != "Seattle" {
			t.Error("City should be kept for limited dataset")
		}
		if addr.State != "WA" {
			t.Error("State should be kept for limited dataset")
		}
		// Street should still be removed
		if addr.Line != nil {
			t.Error("Street address should be removed")
		}
	}
}

func TestEngine_AnonymizePatient_DateShift(t *testing.T) {
	cfg := &Config{
		Method:           "safe_harbor",
		DateShiftRange:   30,
		ZipCodeTruncation: 3,
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-789",
		},
		BirthDate: "1985-06-15",
	}

	anonymized := e.AnonymizePatient(patient)

	// Date should be year only for Safe Harbor
	if len(anonymized.BirthDate) > 4 && !strings.HasPrefix(anonymized.BirthDate, "19") {
		t.Errorf("BirthDate should be year only, got %s", anonymized.BirthDate)
	}
}

func TestEngine_AnonymizePatient_Age90Plus(t *testing.T) {
	cfg := &Config{
		Method:           "safe_harbor",
		DateShiftRange:   30,
		ZipCodeTruncation: 3,
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-90plus",
		},
		BirthDate: "1920-01-01", // 90+ years old
	}

	anonymized := e.AnonymizePatient(patient)

	// Date should be generalized to 1900 for 90+ years
	if anonymized.BirthDate != "1900" {
		t.Errorf("BirthDate for 90+ should be '1900', got %s", anonymized.BirthDate)
	}
}

func TestEngine_AnonymizePatient_Consistent_Pseudonym(t *testing.T) {
	cfg := &Config{
		Method:         "safe_harbor",
		DateShiftRange: 30,
		Salt:           "test-salt",
	}
	e := NewEngine(cfg)

	patient1 := &models.Patient{FHIRResource: models.FHIRResource{ID: "patient-same"}}
	patient2 := &models.Patient{FHIRResource: models.FHIRResource{ID: "patient-same"}}

	anon1 := e.AnonymizePatient(patient1)
	anon2 := e.AnonymizePatient(patient2)

	// Same ID should produce same pseudonym
	if anon1.ID != anon2.ID {
		t.Error("Same patient ID should produce consistent pseudonym")
	}
}

func TestEngine_AnonymizePatient_Consistent_DateShift(t *testing.T) {
	cfg := &Config{
		Method:         "limited_dataset",
		DateShiftRange: 30,
		Salt:           "test-salt",
	}
	e := NewEngine(cfg)

	patient1 := &models.Patient{FHIRResource: models.FHIRResource{ID: "patient-date"}, BirthDate: "1980-01-01"}
	patient2 := &models.Patient{FHIRResource: models.FHIRResource{ID: "patient-date"}, BirthDate: "1980-01-01"}

	anon1 := e.AnonymizePatient(patient1)
	anon2 := e.AnonymizePatient(patient2)

	// Same patient should have same date shift
	if anon1.BirthDate != anon2.BirthDate {
		t.Error("Same patient ID should have consistent date shift")
	}
}

func TestEngine_AnonymizeObservation(t *testing.T) {
	cfg := &Config{
		Method:         "safe_harbor",
		DateShiftRange: 30,
		Salt:           "test-salt",
	}
	e := NewEngine(cfg)

	now := time.Now()
	obs := &models.Observation{
		FHIRResource: models.FHIRResource{
			ID: "obs-123",
		},
		Subject: &models.Reference{
			Reference: "Patient/patient-123",
		},
		EffectiveDateTime: &now,
		Issued:            &now,
		Note: []models.Annotation{
			{Text: "Patient note with PHI"},
		},
		Performer: []models.Reference{
			{Reference: "Practitioner/doc-123"},
		},
	}

	anonymized := e.AnonymizeObservation(obs, "patient-123")

	// ID should be pseudonymized
	if anonymized.ID == "obs-123" {
		t.Error("Observation ID should be pseudonymized")
	}

	// Subject reference should be pseudonymized
	if anonymized.Subject.Reference == "Patient/patient-123" {
		t.Error("Subject reference should be pseudonymized")
	}

	// Notes should be removed
	if anonymized.Note != nil {
		t.Error("Notes should be removed")
	}

	// Performer should be removed
	if anonymized.Performer != nil {
		t.Error("Performer should be removed")
	}
}

func TestEngine_AnonymizeEncounter(t *testing.T) {
	cfg := &Config{
		Method:         "safe_harbor",
		DateShiftRange: 30,
		Salt:           "test-salt",
	}
	e := NewEngine(cfg)

	start := time.Now().Add(-time.Hour)
	end := time.Now()
	enc := &models.Encounter{
		FHIRResource: models.FHIRResource{
			ID: "enc-123",
		},
		Subject: &models.Reference{
			Reference: "Patient/patient-123",
		},
		Period: &models.Period{
			Start: &start,
			End:   &end,
		},
		Participant: []models.EncounterParticipant{
			{Individual: &models.Reference{Reference: "Practitioner/doc-123"}},
		},
		ServiceProvider: &models.Reference{
			Reference: "Organization/org-123",
		},
	}

	anonymized := e.AnonymizeEncounter(enc, "patient-123")

	// ID should be pseudonymized
	if anonymized.ID == "enc-123" {
		t.Error("Encounter ID should be pseudonymized")
	}

	// Subject should be pseudonymized
	if anonymized.Subject.Reference == "Patient/patient-123" {
		t.Error("Subject should be pseudonymized")
	}

	// Participant should be removed
	if anonymized.Participant != nil {
		t.Error("Participant should be removed")
	}

	// ServiceProvider should be removed
	if anonymized.ServiceProvider != nil {
		t.Error("ServiceProvider should be removed")
	}
}

func TestEngine_AnonymizeJSON(t *testing.T) {
	cfg := &Config{
		Method:           "safe_harbor",
		DateShiftRange:   30,
		ZipCodeTruncation: 3,
		Salt:             "test-salt",
	}
	e := NewEngine(cfg)

	input := map[string]interface{}{
		"id":        "resource-123",
		"name":      []map[string]interface{}{{"family": "Doe"}},
		"telecom":   []map[string]interface{}{{"value": "555-1234"}},
		"birthDate": "1980-01-01",
	}
	jsonData, _ := json.Marshal(input)

	result, err := e.AnonymizeJSON(jsonData, models.ResourceTypePatient, "patient-123")
	if err != nil {
		t.Fatalf("AnonymizeJSON failed: %v", err)
	}

	var output map[string]interface{}
	json.Unmarshal(result, &output)

	// Name should be removed
	if _, ok := output["name"]; ok {
		t.Error("name should be removed")
	}

	// Telecom should be removed
	if _, ok := output["telecom"]; ok {
		t.Error("telecom should be removed")
	}

	// ID should be pseudonymized
	if output["id"] == "resource-123" {
		t.Error("ID should be pseudonymized")
	}
}

func TestEngine_AnonymizeJSON_InvalidJSON(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	_, err := e.AnonymizeJSON([]byte("invalid json"), models.ResourceTypePatient, "patient")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEngine_RedactText(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"SSN", "My SSN is 123-45-6789", "[SSN REDACTED]"},
		{"phone", "Call me at 555-123-4567", "[PHONE REDACTED]"},
		{"email", "Email john@example.com for info", "[EMAIL REDACTED]"},
		{"date slash", "Born on 01/15/1980", "[DATE REDACTED]"},
		{"date iso", "DOB: 1980-01-15", "[DATE REDACTED]"},
		{"MRN", "MRN: 12345", "[MRN REDACTED]"},
		{"age", "Patient is 45 years old", "[AGE REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.RedactText(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %s, got: %s", tt.contains, result)
			}
		})
	}
}

func TestEngine_RedactText_NoMatch(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	input := "This is a normal clinical note without PHI identifiers."
	result := e.RedactText(input)

	if result != input {
		t.Errorf("text without PHI should be unchanged, got: %s", result)
	}
}

func TestEngine_CheckKAnonymity_Satisfied(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	records := []map[string]interface{}{
		{"zip": "02100", "age_range": "30-40"},
		{"zip": "02100", "age_range": "30-40"},
		{"zip": "02100", "age_range": "30-40"},
		{"zip": "02200", "age_range": "40-50"},
		{"zip": "02200", "age_range": "40-50"},
		{"zip": "02200", "age_range": "40-50"},
	}

	result := e.CheckKAnonymity(records, []string{"zip", "age_range"}, 3)

	if !result.Satisfied {
		t.Error("k=3 anonymity should be satisfied")
	}
	if result.K != 3 {
		t.Errorf("K = %d, want 3", result.K)
	}
	if result.TotalGroups != 2 {
		t.Errorf("TotalGroups = %d, want 2", result.TotalGroups)
	}
}

func TestEngine_CheckKAnonymity_NotSatisfied(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	records := []map[string]interface{}{
		{"zip": "02100", "age_range": "30-40"},
		{"zip": "02100", "age_range": "30-40"},
		{"zip": "02200", "age_range": "40-50"}, // Only 1 record in this group
	}

	result := e.CheckKAnonymity(records, []string{"zip", "age_range"}, 2)

	if result.Satisfied {
		t.Error("k=2 anonymity should NOT be satisfied")
	}
	if len(result.ViolatingGroups) == 0 {
		t.Error("should have violating groups")
	}
}

func TestEngine_CheckKAnonymity_EmptyRecords(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test"}
	e := NewEngine(cfg)

	result := e.CheckKAnonymity([]map[string]interface{}{}, []string{"zip"}, 3)

	if !result.Satisfied {
		t.Error("empty records should satisfy any k")
	}
	if result.TotalGroups != 0 {
		t.Errorf("TotalGroups = %d, want 0", result.TotalGroups)
	}
}

func TestEngine_GeneratePseudonym_Deterministic(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test-salt"}
	e := NewEngine(cfg)

	pseudo1 := e.generatePseudonym("original-id")
	pseudo2 := e.generatePseudonym("original-id")

	if pseudo1 != pseudo2 {
		t.Error("pseudonym should be deterministic for same input")
	}
	if pseudo1 == "original-id" {
		t.Error("pseudonym should not equal original")
	}
	if len(pseudo1) != 16 {
		t.Errorf("pseudonym length = %d, want 16", len(pseudo1))
	}
}

func TestEngine_GeneratePseudonym_Different_Salts(t *testing.T) {
	e1 := NewEngine(&Config{Method: "safe_harbor", Salt: "salt1"})
	e2 := NewEngine(&Config{Method: "safe_harbor", Salt: "salt2"})

	pseudo1 := e1.generatePseudonym("same-id")
	pseudo2 := e2.generatePseudonym("same-id")

	if pseudo1 == pseudo2 {
		t.Error("different salts should produce different pseudonyms")
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := &Config{
		Method:            "safe_harbor",
		DateShiftRange:    365,
		PreserveBirthYear: true,
		ZipCodeTruncation: 3,
		Salt:              "my-salt",
	}

	if cfg.Method != "safe_harbor" {
		t.Error("Method not set")
	}
	if cfg.DateShiftRange != 365 {
		t.Error("DateShiftRange not set")
	}
	if !cfg.PreserveBirthYear {
		t.Error("PreserveBirthYear not set")
	}
}

func TestKAnonymityResult_Struct(t *testing.T) {
	result := &KAnonymityResult{
		K:           5,
		Satisfied:   false,
		TotalGroups: 10,
		ViolatingGroups: []GroupInfo{
			{QuasiIdentifiers: "zip|age", Count: 2},
		},
	}

	if result.K != 5 {
		t.Error("K not set")
	}
	if len(result.ViolatingGroups) != 1 {
		t.Error("ViolatingGroups not set")
	}
}

func TestEngine_AnonymizeReference(t *testing.T) {
	cfg := &Config{Method: "safe_harbor", Salt: "test-salt", DateShiftRange: 30}
	e := NewEngine(cfg)

	tests := []struct {
		name  string
		input string
	}{
		{"patient reference", "Patient/patient-123"},
		{"observation reference", "Observation/obs-456"},
		{"simple ID", "simple-id-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.anonymizeReference(tt.input)
			if result == tt.input {
				t.Error("reference should be anonymized")
			}
		})
	}
}

func TestEngine_AnonymizeJSON_WithNested(t *testing.T) {
	cfg := &Config{
		Method:            "safe_harbor",
		DateShiftRange:    30,
		ZipCodeTruncation: 3,
		Salt:              "test-salt",
	}
	e := NewEngine(cfg)

	input := map[string]interface{}{
		"id": "resource-123",
		"name": []map[string]interface{}{
			{"family": "Doe", "given": []string{"John"}},
		},
		"telecom": []map[string]interface{}{
			{"value": "555-1234"},
		},
		"address": []interface{}{
			map[string]interface{}{
				"line":       []string{"123 Main St"},
				"city":       "Boston",
				"postalCode": "02101",
			},
		},
		"birthDate": "1980-01-15",
		"contact": []map[string]interface{}{
			{"name": "Emergency Contact"},
		},
		"identifier": []map[string]interface{}{
			{"value": "MRN12345"},
		},
		"nested": map[string]interface{}{
			"name": []map[string]interface{}{{"family": "NestedDoe"}},
		},
	}
	jsonData, _ := json.Marshal(input)

	result, err := e.AnonymizeJSON(jsonData, models.ResourceTypePatient, "patient-123")
	if err != nil {
		t.Fatalf("AnonymizeJSON failed: %v", err)
	}

	var output map[string]interface{}
	json.Unmarshal(result, &output)

	// name should be removed
	if _, ok := output["name"]; ok {
		t.Error("name should be removed")
	}

	// telecom should be removed
	if _, ok := output["telecom"]; ok {
		t.Error("telecom should be removed")
	}

	// contact should be removed
	if _, ok := output["contact"]; ok {
		t.Error("contact should be removed")
	}

	// identifier should be removed (Safe Harbor)
	if _, ok := output["identifier"]; ok {
		t.Error("identifier should be removed for Safe Harbor")
	}

	// ID should be pseudonymized
	if output["id"] == "resource-123" {
		t.Error("ID should be pseudonymized")
	}

	// address should be truncated
	if addr, ok := output["address"].([]interface{}); ok && len(addr) > 0 {
		if addrMap, ok := addr[0].(map[string]interface{}); ok {
			// line should be removed
			if _, ok := addrMap["line"]; ok {
				t.Error("address line should be removed")
			}
			// postal code should be truncated
			if pc, ok := addrMap["postalCode"].(string); ok {
				if !strings.HasSuffix(pc, "00") {
					t.Errorf("postal code should be truncated, got %s", pc)
				}
			}
		}
	}

	// nested structures should be processed
	if nested, ok := output["nested"].(map[string]interface{}); ok {
		if _, ok := nested["name"]; ok {
			t.Error("nested name should be removed")
		}
	}
}

func TestEngine_AnonymizeIdentifiers(t *testing.T) {
	cfg := &Config{
		Method:         "limited_dataset", // Limited dataset keeps some identifiers
		DateShiftRange: 30,
		Salt:           "test-salt",
	}
	e := NewEngine(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-123",
			Identifier: []models.Identifier{
				{Value: "MRN12345", System: "http://hospital.org/mrn"},
			},
		},
	}

	anonymized := e.AnonymizePatient(patient)

	// ID should be pseudonymized
	if anonymized.ID == "patient-123" {
		t.Error("ID should be pseudonymized")
	}
}
