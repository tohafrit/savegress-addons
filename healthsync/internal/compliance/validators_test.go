package compliance

import (
	"testing"

	"github.com/savegress/healthsync/pkg/models"
)

func TestMinimumNecessaryValidator_Name(t *testing.T) {
	v := NewMinimumNecessaryValidator(true)
	if v.Name() != "minimum_necessary" {
		t.Errorf("expected name 'minimum_necessary', got %s", v.Name())
	}
}

func TestMinimumNecessaryValidator_Validate_Disabled(t *testing.T) {
	v := NewMinimumNecessaryValidator(false)
	results := v.Validate(nil, models.ResourceTypePatient)

	if results != nil {
		t.Error("expected nil results when disabled")
	}
}

func TestMinimumNecessaryValidator_Validate_Enabled(t *testing.T) {
	v := NewMinimumNecessaryValidator(true)
	resource := map[string]interface{}{
		"name": "test",
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	// This validator returns nil for resource validation (used during access control)
	if results != nil {
		t.Error("expected nil results for resource validation")
	}
}

func TestPHIExposureValidator_Name(t *testing.T) {
	v := NewPHIExposureValidator(nil)
	if v.Name() != "phi_exposure" {
		t.Errorf("expected name 'phi_exposure', got %s", v.Name())
	}
}

func TestPHIExposureValidator_Validate_SSNExposure(t *testing.T) {
	phiFields := map[models.ResourceType][]models.PHIField{
		models.ResourceTypePatient: {
			{FieldName: "identifier.ssn", PHICategory: models.PHICategorySSN},
		},
	}
	v := NewPHIExposureValidator(phiFields)

	resource := map[string]interface{}{
		"identifier.ssn": "123-45-6789", // Plain text SSN
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasSSNViolation := false
	for _, r := range results {
		if r.Field == "identifier.ssn" && r.Severity == "critical" {
			hasSSNViolation = true
			break
		}
	}

	if !hasSSNViolation {
		t.Error("expected SSN exposure violation")
	}
}

func TestPHIExposureValidator_Validate_SSNWithoutDashes(t *testing.T) {
	phiFields := map[models.ResourceType][]models.PHIField{
		models.ResourceTypePatient: {
			{FieldName: "ssn", PHICategory: models.PHICategorySSN},
		},
	}
	v := NewPHIExposureValidator(phiFields)

	resource := map[string]interface{}{
		"ssn": "123456789", // Plain text SSN without dashes
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasSSNViolation := false
	for _, r := range results {
		if r.Field == "ssn" {
			hasSSNViolation = true
			break
		}
	}

	if !hasSSNViolation {
		t.Error("expected SSN exposure violation for SSN without dashes")
	}
}

func TestPHIExposureValidator_Validate_NotSSN(t *testing.T) {
	phiFields := map[models.ResourceType][]models.PHIField{
		models.ResourceTypePatient: {
			{FieldName: "identifier", PHICategory: models.PHICategorySSN},
		},
	}
	v := NewPHIExposureValidator(phiFields)

	resource := map[string]interface{}{
		"identifier": "MRN12345", // Not an SSN pattern
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.Field == "identifier" && r.ViolationType == "phi_exposure" {
			t.Error("should not flag non-SSN identifiers")
		}
	}
}

func TestPHIExposureValidator_Validate_PhoneInNarrative(t *testing.T) {
	v := NewPHIExposureValidator(nil)

	resource := map[string]interface{}{
		"text": map[string]interface{}{
			"div": "<div>Patient phone: 555-1234</div>",
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasPhoneViolation := false
	for _, r := range results {
		if r.Field == "text.div" && r.Severity == "high" {
			hasPhoneViolation = true
			break
		}
	}

	if !hasPhoneViolation {
		t.Error("expected phone exposure violation in narrative")
	}
}

func TestPHIExposureValidator_Validate_TelInNarrative(t *testing.T) {
	v := NewPHIExposureValidator(nil)

	resource := map[string]interface{}{
		"text": map[string]interface{}{
			"div": "<div>Contact: tel:+1-555-1234</div>",
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasPhoneViolation := false
	for _, r := range results {
		if r.Field == "text.div" {
			hasPhoneViolation = true
			break
		}
	}

	if !hasPhoneViolation {
		t.Error("expected tel: pattern violation in narrative")
	}
}

func TestPHIExposureValidator_Validate_CleanNarrative(t *testing.T) {
	v := NewPHIExposureValidator(nil)

	resource := map[string]interface{}{
		"text": map[string]interface{}{
			"div": "<div>Patient is in good health</div>",
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.Field == "text.div" {
			t.Error("should not flag clean narrative")
		}
	}
}

func TestPHIExposureValidator_Validate_InvalidJSON(t *testing.T) {
	v := NewPHIExposureValidator(nil)

	// Function can't be marshaled
	resource := func() {}

	results := v.Validate(resource, models.ResourceTypePatient)

	if len(results) > 0 {
		t.Error("expected empty results for invalid resource")
	}
}

func TestPHIExposureValidator_LooksLikeSSN(t *testing.T) {
	v := &PHIExposureValidator{}

	tests := []struct {
		input    string
		expected bool
	}{
		{"123-45-6789", true},
		{"123456789", true},
		{"123 45 6789", true},
		{"12345678", false},   // Too short
		{"1234567890", false}, // Too long
		{"12345678a", false},  // Contains letter
		{"", false},
	}

	for _, test := range tests {
		result := v.looksLikeSSN(test.input)
		if result != test.expected {
			t.Errorf("looksLikeSSN(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestPHIExposureValidator_ContainsPhonePattern(t *testing.T) {
	v := &PHIExposureValidator{}

	tests := []struct {
		input    string
		expected bool
	}{
		{"Patient phone: 555-1234", true},
		{"Contact: tel:+1-555-1234", true},
		{"PHONE number is listed", true},
		{"Patient is healthy", false},
		{"", false},
	}

	for _, test := range tests {
		result := v.containsPhonePattern(test.input)
		if result != test.expected {
			t.Errorf("containsPhonePattern(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestDataIntegrityValidator_Name(t *testing.T) {
	v := NewDataIntegrityValidator()
	if v.Name() != "data_integrity" {
		t.Errorf("expected name 'data_integrity', got %s", v.Name())
	}
}

func TestDataIntegrityValidator_Validate_PatientWithoutIdentifier(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"name": []interface{}{
			map[string]interface{}{"family": "Doe"},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasIdentifierViolation := false
	for _, r := range results {
		if r.Field == "identifier" {
			hasIdentifierViolation = true
			break
		}
	}

	if !hasIdentifierViolation {
		t.Error("expected identifier violation for patient")
	}
}

func TestDataIntegrityValidator_Validate_PatientWithIdentifier(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier": []interface{}{
			map[string]interface{}{"system": "mrn", "value": "12345"},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.Field == "identifier" {
			t.Error("should not flag patient with identifier")
		}
	}
}

func TestDataIntegrityValidator_Validate_ObservationMissingFields(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Observation",
		"subject": map[string]interface{}{
			"reference": "Patient/123",
		},
	}

	results := v.Validate(resource, models.ResourceTypeObservation)

	missingFields := make(map[string]bool)
	for _, r := range results {
		missingFields[r.Field] = true
	}

	if !missingFields["code"] {
		t.Error("expected code violation for observation")
	}

	if !missingFields["status"] {
		t.Error("expected status violation for observation")
	}
}

func TestDataIntegrityValidator_Validate_ObservationComplete(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Observation",
		"code": map[string]interface{}{
			"coding": []interface{}{
				map[string]interface{}{"code": "test"},
			},
		},
		"status": "final",
	}

	results := v.Validate(resource, models.ResourceTypeObservation)

	for _, r := range results {
		if r.Field == "code" || r.Field == "status" {
			t.Errorf("should not flag complete observation, got violation for %s", r.Field)
		}
	}
}

func TestDataIntegrityValidator_Validate_EncounterMissingClass(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Encounter",
		"subject": map[string]interface{}{
			"reference": "Patient/123",
		},
	}

	results := v.Validate(resource, models.ResourceTypeEncounter)

	hasClassViolation := false
	for _, r := range results {
		if r.Field == "class" {
			hasClassViolation = true
			break
		}
	}

	if !hasClassViolation {
		t.Error("expected class violation for encounter")
	}
}

func TestDataIntegrityValidator_Validate_MissingLastUpdated(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier":   []interface{}{map[string]interface{}{"value": "123"}},
		"meta":         map[string]interface{}{},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasLastUpdatedViolation := false
	for _, r := range results {
		if r.Field == "meta.lastUpdated" {
			hasLastUpdatedViolation = true
			break
		}
	}

	if !hasLastUpdatedViolation {
		t.Error("expected meta.lastUpdated violation")
	}
}

func TestDataIntegrityValidator_Validate_InvalidJSON(t *testing.T) {
	v := NewDataIntegrityValidator()

	resource := func() {}

	results := v.Validate(resource, models.ResourceTypePatient)

	if len(results) > 0 {
		t.Error("expected empty results for invalid resource")
	}
}

func TestAccessControlValidator_Name(t *testing.T) {
	v := NewAccessControlValidator()
	if v.Name() != "access_control" {
		t.Errorf("expected name 'access_control', got %s", v.Name())
	}
}

func TestAccessControlValidator_Validate_MissingSecurity(t *testing.T) {
	v := NewAccessControlValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"meta":         map[string]interface{}{},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasSecurityInfo := false
	for _, r := range results {
		if r.Field == "meta.security" && r.Severity == "info" {
			hasSecurityInfo = true
			// This should be Valid = true (just informational)
			if !r.Valid {
				t.Error("security recommendation should have Valid = true")
			}
			break
		}
	}

	if !hasSecurityInfo {
		t.Error("expected security recommendation")
	}
}

func TestAccessControlValidator_Validate_WithSecurity(t *testing.T) {
	v := NewAccessControlValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"meta": map[string]interface{}{
			"security": []interface{}{
				map[string]interface{}{"code": "R"},
			},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.Field == "meta.security" {
			t.Error("should not flag resource with security labels")
		}
	}
}

func TestAccessControlValidator_Validate_InvalidJSON(t *testing.T) {
	v := NewAccessControlValidator()

	resource := func() {}

	results := v.Validate(resource, models.ResourceTypePatient)

	if len(results) > 0 {
		t.Error("expected empty results for invalid resource")
	}
}

func TestEncryptionValidator_Name(t *testing.T) {
	v := NewEncryptionValidator()
	if v.Name() != "encryption" {
		t.Errorf("expected name 'encryption', got %s", v.Name())
	}
}

func TestEncryptionValidator_Validate_UnencryptedSSN(t *testing.T) {
	v := NewEncryptionValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier": []interface{}{
			map[string]interface{}{
				"system": "http://hl7.org/fhir/sid/us-ssn",
				"value":  "123456789",
			},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasEncryptionViolation := false
	for _, r := range results {
		if r.ViolationType == "encryption" && r.Severity == "critical" {
			hasEncryptionViolation = true
			break
		}
	}

	if !hasEncryptionViolation {
		t.Error("expected encryption violation for unencrypted SSN")
	}
}

func TestEncryptionValidator_Validate_EncryptedSSN(t *testing.T) {
	v := NewEncryptionValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier": []interface{}{
			map[string]interface{}{
				"system": "http://hl7.org/fhir/sid/us-ssn",
				"value":  "SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBsb25nIGVuY3J5cHRlZCB2YWx1ZQ==",
			},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.ViolationType == "encryption" && !r.Valid {
			t.Error("should not flag encrypted SSN")
		}
	}
}

func TestEncryptionValidator_Validate_SocialSecuritySystem(t *testing.T) {
	v := NewEncryptionValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier": []interface{}{
			map[string]interface{}{
				"system": "http://example.org/social-security",
				"value":  "123456789",
			},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	hasEncryptionViolation := false
	for _, r := range results {
		if r.ViolationType == "encryption" {
			hasEncryptionViolation = true
			break
		}
	}

	if !hasEncryptionViolation {
		t.Error("expected encryption violation for social-security system")
	}
}

func TestEncryptionValidator_Validate_NonSSNIdentifier(t *testing.T) {
	v := NewEncryptionValidator()

	resource := map[string]interface{}{
		"resourceType": "Patient",
		"identifier": []interface{}{
			map[string]interface{}{
				"system": "http://hospital.org/mrn",
				"value":  "MRN12345",
			},
		},
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	for _, r := range results {
		if r.ViolationType == "encryption" {
			t.Error("should not flag non-SSN identifiers")
		}
	}
}

func TestEncryptionValidator_Validate_InvalidJSON(t *testing.T) {
	v := NewEncryptionValidator()

	resource := func() {}

	results := v.Validate(resource, models.ResourceTypePatient)

	if len(results) > 0 {
		t.Error("expected empty results for invalid resource")
	}
}

func TestEncryptionValidator_LooksEncrypted(t *testing.T) {
	v := &EncryptionValidator{}

	tests := []struct {
		input    string
		expected bool
	}{
		{"SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBsb25nIGVuY3J5cHRlZCB2YWx1ZQ==", true}, // Base64
		{"ABCDEFGHIJKLMNOPQRST", true},                                            // Long alphanumeric
		{"123456789", false},                                                      // Too short
		{"short", false},                                                          // Too short
		{"test@example.com", false},                                               // Contains @
		{"", false},
	}

	for _, test := range tests {
		result := v.looksEncrypted(test.input)
		if result != test.expected {
			t.Errorf("looksEncrypted(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestRetentionValidator_Name(t *testing.T) {
	v := NewRetentionValidator(7)
	if v.Name() != "retention" {
		t.Errorf("expected name 'retention', got %s", v.Name())
	}
}

func TestRetentionValidator_Validate(t *testing.T) {
	v := NewRetentionValidator(7)

	resource := map[string]interface{}{
		"resourceType": "Patient",
	}

	results := v.Validate(resource, models.ResourceTypePatient)

	// Retention validation is done during lifecycle management, not creation
	if results != nil {
		t.Error("expected nil results for retention validator")
	}
}
