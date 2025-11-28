package compliance

import (
	"encoding/json"
	"strings"

	"github.com/savegress/healthsync/pkg/models"
)

// MinimumNecessaryValidator validates minimum necessary principle
type MinimumNecessaryValidator struct {
	enabled bool
}

// NewMinimumNecessaryValidator creates a new minimum necessary validator
func NewMinimumNecessaryValidator(enabled bool) *MinimumNecessaryValidator {
	return &MinimumNecessaryValidator{enabled: enabled}
}

func (v *MinimumNecessaryValidator) Name() string { return "minimum_necessary" }

func (v *MinimumNecessaryValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	if !v.enabled {
		return nil
	}

	// This validator is typically used during access control, not resource validation
	return nil
}

// PHIExposureValidator validates PHI exposure
type PHIExposureValidator struct {
	phiFields map[models.ResourceType][]models.PHIField
}

// NewPHIExposureValidator creates a new PHI exposure validator
func NewPHIExposureValidator(phiFields map[models.ResourceType][]models.PHIField) *PHIExposureValidator {
	return &PHIExposureValidator{phiFields: phiFields}
}

func (v *PHIExposureValidator) Name() string { return "phi_exposure" }

func (v *PHIExposureValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	var results []ValidationResult

	// Convert resource to map
	data, err := json.Marshal(resource)
	if err != nil {
		return results
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return results
	}

	fields := v.phiFields[resourceType]

	// Check for SSN exposure in text fields
	for _, field := range fields {
		if field.PHICategory == models.PHICategorySSN {
			if val, ok := resourceMap[field.FieldName]; ok && val != nil {
				// Check if SSN is exposed in plain text
				if str, isStr := val.(string); isStr {
					if v.looksLikeSSN(str) {
						results = append(results, ValidationResult{
							Valid:         false,
							ViolationType: "phi_exposure",
							Severity:      "critical",
							Field:         field.FieldName,
							Description:   "SSN appears to be stored in plain text",
							Remediation:   "Encrypt or hash SSN values before storage",
						})
					}
				}
			}
		}
	}

	// Check for unmasked phone numbers in text fields
	if narrative, ok := resourceMap["text"].(map[string]interface{}); ok {
		if div, ok := narrative["div"].(string); ok {
			if v.containsPhonePattern(div) {
				results = append(results, ValidationResult{
					Valid:         false,
					ViolationType: "phi_exposure",
					Severity:      "high",
					Field:         "text.div",
					Description:   "Phone number may be exposed in narrative text",
					Remediation:   "Mask or remove phone numbers from narrative text",
				})
			}
		}
	}

	return results
}

func (v *PHIExposureValidator) looksLikeSSN(s string) bool {
	// Remove common separators
	cleaned := strings.ReplaceAll(s, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	// Check if it's 9 digits
	if len(cleaned) != 9 {
		return false
	}

	for _, c := range cleaned {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

func (v *PHIExposureValidator) containsPhonePattern(s string) bool {
	// Simple check for phone-like patterns
	// In production, use regex
	s = strings.ToLower(s)
	if strings.Contains(s, "phone") || strings.Contains(s, "tel:") {
		return true
	}
	return false
}

// DataIntegrityValidator validates data integrity
type DataIntegrityValidator struct{}

// NewDataIntegrityValidator creates a new data integrity validator
func NewDataIntegrityValidator() *DataIntegrityValidator {
	return &DataIntegrityValidator{}
}

func (v *DataIntegrityValidator) Name() string { return "data_integrity" }

func (v *DataIntegrityValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	var results []ValidationResult

	// Convert resource to map
	data, err := json.Marshal(resource)
	if err != nil {
		return results
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return results
	}

	// Check for required fields based on resource type
	switch resourceType {
	case models.ResourceTypePatient:
		// Patient must have identifier
		if _, ok := resourceMap["identifier"]; !ok {
			results = append(results, ValidationResult{
				Valid:         false,
				ViolationType: "data_integrity",
				Severity:      "medium",
				Field:         "identifier",
				Description:   "Patient resource should have at least one identifier",
				Remediation:   "Add a patient identifier (MRN, SSN hash, etc.)",
			})
		}

	case models.ResourceTypeObservation:
		// Observation must have code and status
		if _, ok := resourceMap["code"]; !ok {
			results = append(results, ValidationResult{
				Valid:         false,
				ViolationType: "data_integrity",
				Severity:      "high",
				Field:         "code",
				Description:   "Observation must have a code",
				Remediation:   "Add observation code using appropriate coding system",
			})
		}
		if _, ok := resourceMap["status"]; !ok {
			results = append(results, ValidationResult{
				Valid:         false,
				ViolationType: "data_integrity",
				Severity:      "high",
				Field:         "status",
				Description:   "Observation must have a status",
				Remediation:   "Add observation status (registered, preliminary, final, etc.)",
			})
		}

	case models.ResourceTypeEncounter:
		// Encounter must have class and status
		if _, ok := resourceMap["class"]; !ok {
			results = append(results, ValidationResult{
				Valid:         false,
				ViolationType: "data_integrity",
				Severity:      "high",
				Field:         "class",
				Description:   "Encounter must have a class",
				Remediation:   "Add encounter class (inpatient, outpatient, emergency, etc.)",
			})
		}
	}

	// Check meta.lastUpdated for audit trail
	if meta, ok := resourceMap["meta"].(map[string]interface{}); ok {
		if _, ok := meta["lastUpdated"]; !ok {
			results = append(results, ValidationResult{
				Valid:         false,
				ViolationType: "data_integrity",
				Severity:      "low",
				Field:         "meta.lastUpdated",
				Description:   "Resource should have lastUpdated timestamp for audit trail",
				Remediation:   "Set meta.lastUpdated when creating or updating resource",
			})
		}
	}

	return results
}

// AccessControlValidator validates access control requirements
type AccessControlValidator struct{}

// NewAccessControlValidator creates a new access control validator
func NewAccessControlValidator() *AccessControlValidator {
	return &AccessControlValidator{}
}

func (v *AccessControlValidator) Name() string { return "access_control" }

func (v *AccessControlValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	var results []ValidationResult

	// Convert resource to map
	data, err := json.Marshal(resource)
	if err != nil {
		return results
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return results
	}

	// Check for security labels
	if meta, ok := resourceMap["meta"].(map[string]interface{}); ok {
		if _, ok := meta["security"]; !ok {
			// Not all resources need security labels, so this is informational
			results = append(results, ValidationResult{
				Valid:         true, // Not a violation, just a recommendation
				ViolationType: "access_control",
				Severity:      "info",
				Field:         "meta.security",
				Description:   "Consider adding security labels for fine-grained access control",
				Remediation:   "Add security labels based on data sensitivity",
			})
		}
	}

	return results
}

// EncryptionValidator validates encryption requirements
type EncryptionValidator struct{}

// NewEncryptionValidator creates a new encryption validator
func NewEncryptionValidator() *EncryptionValidator {
	return &EncryptionValidator{}
}

func (v *EncryptionValidator) Name() string { return "encryption" }

func (v *EncryptionValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	var results []ValidationResult

	// This validator checks for encryption markers or patterns
	// In production, this would integrate with encryption service

	data, err := json.Marshal(resource)
	if err != nil {
		return results
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(data, &resourceMap); err != nil {
		return results
	}

	// Check for sensitive identifiers that should be encrypted
	if identifiers, ok := resourceMap["identifier"].([]interface{}); ok {
		for i, id := range identifiers {
			if idMap, ok := id.(map[string]interface{}); ok {
				system, _ := idMap["system"].(string)

				// SSN should always be encrypted
				if strings.Contains(strings.ToLower(system), "ssn") ||
					strings.Contains(strings.ToLower(system), "social-security") {
					if value, ok := idMap["value"].(string); ok {
						// Check if value looks encrypted (simplified check)
						if !v.looksEncrypted(value) {
							results = append(results, ValidationResult{
								Valid:         false,
								ViolationType: "encryption",
								Severity:      "critical",
								Field:         "identifier[" + string(rune(i)) + "].value",
								Description:   "Social Security Number should be encrypted at rest",
								Remediation:   "Encrypt SSN using approved encryption algorithm",
							})
						}
					}
				}
			}
		}
	}

	return results
}

func (v *EncryptionValidator) looksEncrypted(s string) bool {
	// Simple heuristic: encrypted data is typically base64 encoded and longer
	// In production, check for encryption markers or use encryption service
	if len(s) < 20 {
		return false
	}

	// Check for base64-like pattern
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			continue
		}
		return false
	}

	return true
}

// RetentionValidator validates data retention requirements
type RetentionValidator struct {
	retentionYears int
}

// NewRetentionValidator creates a new retention validator
func NewRetentionValidator(retentionYears int) *RetentionValidator {
	return &RetentionValidator{retentionYears: retentionYears}
}

func (v *RetentionValidator) Name() string { return "retention" }

func (v *RetentionValidator) Validate(resource interface{}, resourceType models.ResourceType) []ValidationResult {
	// Retention validation is typically done during data lifecycle management
	// not during resource creation/update
	return nil
}
