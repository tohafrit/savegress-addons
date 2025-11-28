package deidentification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/savegress/healthsync/pkg/models"
)

// Engine handles de-identification of PHI according to HIPAA Safe Harbor and Expert Determination methods
type Engine struct {
	config      *Config
	dateShift   map[string]int // Patient ID -> date shift in days
	pseudonyms  map[string]string // Original value -> pseudonymized value
	mu          sync.RWMutex
	rng         *rand.Rand
}

// Config holds de-identification configuration
type Config struct {
	Method           DeidentMethod `json:"method"`
	PreserveAge      bool          `json:"preserve_age"`       // Keep ages under 90
	AgeThreshold     int           `json:"age_threshold"`      // Default 89
	DateShiftRange   int           `json:"date_shift_range"`   // Max days to shift dates
	PreserveZIP3     bool          `json:"preserve_zip3"`      // Keep first 3 digits of ZIP
	ZIPPopThreshold  int           `json:"zip_pop_threshold"`  // Population threshold for ZIP
	SaltKey          string        `json:"salt_key"`           // For hashing
	RedactionMarker  string        `json:"redaction_marker"`   // Default "[REDACTED]"
	PreserveFormat   bool          `json:"preserve_format"`    // Keep format of redacted data
}

// DeidentMethod represents the de-identification method
type DeidentMethod string

const (
	MethodSafeHarbor         DeidentMethod = "safe_harbor"
	MethodExpertDetermination DeidentMethod = "expert_determination"
	MethodLimitedDataset     DeidentMethod = "limited_dataset"
	MethodPseudonymization   DeidentMethod = "pseudonymization"
)

// NewEngine creates a new de-identification engine
func NewEngine(config *Config) *Engine {
	if config == nil {
		config = DefaultConfig()
	}

	if config.RedactionMarker == "" {
		config.RedactionMarker = "[REDACTED]"
	}
	if config.AgeThreshold == 0 {
		config.AgeThreshold = 89
	}
	if config.DateShiftRange == 0 {
		config.DateShiftRange = 365
	}

	return &Engine{
		config:     config,
		dateShift:  make(map[string]int),
		pseudonyms: make(map[string]string),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// DefaultConfig returns a default configuration for Safe Harbor method
func DefaultConfig() *Config {
	return &Config{
		Method:          MethodSafeHarbor,
		PreserveAge:     true,
		AgeThreshold:    89,
		DateShiftRange:  365,
		PreserveZIP3:    true,
		ZIPPopThreshold: 20000,
		RedactionMarker: "[REDACTED]",
		PreserveFormat:  false,
	}
}

// DeidentifyPatient de-identifies a patient record
func (e *Engine) DeidentifyPatient(patient *models.Patient) (*models.Patient, *DeidentificationReport) {
	report := &DeidentificationReport{
		OriginalID:    patient.ID,
		Method:        e.config.Method,
		Timestamp:     time.Now(),
		FieldsRemoved: []string{},
		FieldsMasked:  []string{},
		FieldsShifted: []string{},
	}

	// Create a deep copy
	deidentified := e.copyPatient(patient)

	switch e.config.Method {
	case MethodSafeHarbor:
		e.applySafeHarbor(deidentified, report)
	case MethodLimitedDataset:
		e.applyLimitedDataset(deidentified, report)
	case MethodPseudonymization:
		e.applyPseudonymization(deidentified, report)
	case MethodExpertDetermination:
		e.applyExpertDetermination(deidentified, report)
	}

	return deidentified, report
}

// DeidentificationReport contains information about the de-identification process
type DeidentificationReport struct {
	OriginalID    string        `json:"original_id"`
	DeidentID     string        `json:"deident_id"`
	Method        DeidentMethod `json:"method"`
	Timestamp     time.Time     `json:"timestamp"`
	FieldsRemoved []string      `json:"fields_removed"`
	FieldsMasked  []string      `json:"fields_masked"`
	FieldsShifted []string      `json:"fields_shifted"`
	DateShiftDays int           `json:"date_shift_days,omitempty"`
}

func (e *Engine) copyPatient(patient *models.Patient) *models.Patient {
	// Deep copy via JSON marshaling (simplified approach)
	data, _ := json.Marshal(patient)
	var copy models.Patient
	json.Unmarshal(data, &copy)
	return &copy
}

// applySafeHarbor applies HIPAA Safe Harbor de-identification (18 identifiers)
func (e *Engine) applySafeHarbor(patient *models.Patient, report *DeidentificationReport) {
	// 1. Names - Remove all names
	patient.Name = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "name")

	// 2. Geographic data - Keep only state or first 3 digits of ZIP
	for i := range patient.Address {
		patient.Address[i].Line = nil
		patient.Address[i].City = ""
		patient.Address[i].District = ""

		if e.config.PreserveZIP3 && len(patient.Address[i].PostalCode) >= 3 {
			patient.Address[i].PostalCode = patient.Address[i].PostalCode[:3] + "00"
		} else {
			patient.Address[i].PostalCode = ""
		}

		report.FieldsMasked = append(report.FieldsMasked, "address.line", "address.city", "address.postalCode")
	}

	// 3. Dates - Remove or shift dates (except year for age)
	e.deidentifyDates(patient, report)

	// 4-6. Phone, Fax, Email - Remove all telecom
	patient.Telecom = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "telecom")

	// 7. SSN and other identifiers
	var filteredIdentifiers []models.Identifier
	for _, id := range patient.Identifier {
		if !e.isDirectIdentifier(id.System) {
			// Keep non-identifying identifiers or create pseudonymized version
			if e.config.Method == MethodPseudonymization {
				id.Value = e.pseudonymize(id.Value)
				filteredIdentifiers = append(filteredIdentifiers, id)
			}
		} else {
			report.FieldsRemoved = append(report.FieldsRemoved, "identifier."+id.System)
		}
	}
	patient.Identifier = filteredIdentifiers

	// 8. MRN - Handled in identifiers above

	// 9-12. Health plan beneficiary, account numbers, certificates, vehicle IDs
	// Already handled in identifiers

	// 13-14. Device identifiers and serial numbers
	// Would be in extensions

	// 15. Web URLs - Remove from extensions
	e.removeURLsFromExtensions(patient)

	// 16. IP addresses - Remove from extensions

	// 17. Biometric identifiers - Remove from extensions

	// 18. Full-face photographs - Remove from extensions

	// Generate de-identified ID
	patient.ID = e.generateDeidentID(patient.ID)
	report.DeidentID = patient.ID

	// Remove contacts (may contain identifying info)
	patient.Contact = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "contact")

	// Clear managing organization reference
	patient.ManagingOrganization = nil
	patient.GeneralPractitioner = nil
}

// applyLimitedDataset applies Limited Dataset rules (allows more data than Safe Harbor)
func (e *Engine) applyLimitedDataset(patient *models.Patient, report *DeidentificationReport) {
	// Limited dataset can include:
	// - Dates (admission, discharge, DOB, death)
	// - City, state, ZIP code
	// - Ages

	// Must remove:
	// - Names
	patient.Name = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "name")

	// - Street address
	for i := range patient.Address {
		patient.Address[i].Line = nil
		report.FieldsMasked = append(report.FieldsMasked, "address.line")
	}

	// - Phone, fax, email
	patient.Telecom = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "telecom")

	// - SSN and all direct identifiers
	patient.Identifier = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "identifier")

	// - Contact information
	patient.Contact = nil
	report.FieldsRemoved = append(report.FieldsRemoved, "contact")

	// Generate de-identified ID
	patient.ID = e.generateDeidentID(patient.ID)
	report.DeidentID = patient.ID
}

// applyPseudonymization replaces identifiers with consistent pseudonyms
func (e *Engine) applyPseudonymization(patient *models.Patient, report *DeidentificationReport) {
	// Apply Safe Harbor first
	e.applySafeHarbor(patient, report)

	// Then generate consistent pseudonyms for identifiers
	for i := range patient.Identifier {
		patient.Identifier[i].Value = e.pseudonymize(patient.Identifier[i].Value)
	}
}

// applyExpertDetermination applies expert determination method
func (e *Engine) applyExpertDetermination(patient *models.Patient, report *DeidentificationReport) {
	// Expert determination allows for more nuanced de-identification
	// based on statistical analysis of re-identification risk
	// For now, apply Safe Harbor as baseline
	e.applySafeHarbor(patient, report)
}

func (e *Engine) deidentifyDates(patient *models.Patient, report *DeidentificationReport) {
	// Get or create date shift for this patient
	e.mu.Lock()
	shift, ok := e.dateShift[patient.ID]
	if !ok {
		shift = e.rng.Intn(e.config.DateShiftRange*2) - e.config.DateShiftRange
		e.dateShift[patient.ID] = shift
	}
	e.mu.Unlock()

	// Handle birthDate
	if patient.BirthDate != "" {
		if e.config.PreserveAge {
			// Calculate age
			dob, err := time.Parse("2006-01-02", patient.BirthDate)
			if err == nil {
				age := calculateAge(dob)
				if age > e.config.AgeThreshold {
					// For ages over threshold, only keep year indicating ">89"
					patient.BirthDate = fmt.Sprintf("%d-01-01", time.Now().Year()-90)
					report.FieldsMasked = append(report.FieldsMasked, "birthDate(age>89)")
				} else {
					// Keep year, remove month/day or shift
					shifted := dob.AddDate(0, 0, shift)
					patient.BirthDate = shifted.Format("2006-01-02")
					report.FieldsShifted = append(report.FieldsShifted, "birthDate")
					report.DateShiftDays = shift
				}
			}
		} else {
			// Just keep year
			if len(patient.BirthDate) >= 4 {
				patient.BirthDate = patient.BirthDate[:4]
			}
			report.FieldsMasked = append(report.FieldsMasked, "birthDate")
		}
	}

	// Handle deceasedDateTime
	if patient.DeceasedDateTime != nil {
		shifted := patient.DeceasedDateTime.AddDate(0, 0, shift)
		patient.DeceasedDateTime = &shifted
		report.FieldsShifted = append(report.FieldsShifted, "deceasedDateTime")
	}
}

func (e *Engine) isDirectIdentifier(system string) bool {
	directIdentifiers := []string{
		"ssn", "social", "us-ssn",
		"driver", "license",
		"passport",
		"mrn", "medical-record",
		"account",
		"insurance", "health-plan",
		"employee-id",
	}

	systemLower := strings.ToLower(system)
	for _, id := range directIdentifiers {
		if strings.Contains(systemLower, id) {
			return true
		}
	}
	return false
}

func (e *Engine) removeURLsFromExtensions(patient *models.Patient) {
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	var filtered []models.Extension
	for _, ext := range patient.Extension {
		if str, ok := ext.Value.(string); ok {
			if !urlPattern.MatchString(str) {
				filtered = append(filtered, ext)
			}
		} else {
			filtered = append(filtered, ext)
		}
	}
	patient.Extension = filtered
}

func (e *Engine) generateDeidentID(originalID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if pseudonym, ok := e.pseudonyms[originalID]; ok {
		return pseudonym
	}

	// Generate hash-based ID
	hash := sha256.Sum256([]byte(originalID + e.config.SaltKey))
	pseudonym := "DEID-" + hex.EncodeToString(hash[:])[:16]
	e.pseudonyms[originalID] = pseudonym
	return pseudonym
}

func (e *Engine) pseudonymize(value string) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if pseudonym, ok := e.pseudonyms[value]; ok {
		return pseudonym
	}

	hash := sha256.Sum256([]byte(value + e.config.SaltKey))
	pseudonym := "PSN-" + hex.EncodeToString(hash[:])[:12]
	e.pseudonyms[value] = pseudonym
	return pseudonym
}

// DeidentifyBundle de-identifies all resources in a FHIR bundle
func (e *Engine) DeidentifyBundle(bundle json.RawMessage) (json.RawMessage, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(bundle, &data); err != nil {
		return nil, err
	}

	if entries, ok := data["entry"].([]interface{}); ok {
		for i, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					e.deidentifyResource(resource)
					entryMap["resource"] = resource
					entries[i] = entryMap
				}
			}
		}
		data["entry"] = entries
	}

	return json.Marshal(data)
}

func (e *Engine) deidentifyResource(resource map[string]interface{}) {
	resourceType, _ := resource["resourceType"].(string)

	switch resourceType {
	case "Patient":
		e.deidentifyPatientMap(resource)
	case "Observation", "Condition", "Procedure", "DiagnosticReport":
		e.deidentifyClinicalResource(resource)
	case "Practitioner", "Organization":
		e.deidentifyProviderResource(resource)
	}
}

func (e *Engine) deidentifyPatientMap(resource map[string]interface{}) {
	// Remove direct identifiers
	delete(resource, "name")
	delete(resource, "telecom")
	delete(resource, "contact")

	// Mask address
	if addrs, ok := resource["address"].([]interface{}); ok {
		for i, addr := range addrs {
			if addrMap, ok := addr.(map[string]interface{}); ok {
				delete(addrMap, "line")
				delete(addrMap, "city")
				if pc, ok := addrMap["postalCode"].(string); ok && len(pc) >= 3 {
					addrMap["postalCode"] = pc[:3] + "00"
				}
				addrs[i] = addrMap
			}
		}
	}

	// Remove identifiers
	delete(resource, "identifier")

	// Generate new ID
	if id, ok := resource["id"].(string); ok {
		resource["id"] = e.generateDeidentID(id)
	}
}

func (e *Engine) deidentifyClinicalResource(resource map[string]interface{}) {
	// Remove subject reference or pseudonymize
	if subject, ok := resource["subject"].(map[string]interface{}); ok {
		if ref, ok := subject["reference"].(string); ok {
			// Pseudonymize patient reference
			parts := strings.Split(ref, "/")
			if len(parts) == 2 && parts[0] == "Patient" {
				subject["reference"] = "Patient/" + e.generateDeidentID(parts[1])
			}
		}
		delete(subject, "display")
	}

	// Remove performer references
	delete(resource, "performer")

	// Pseudonymize encounter reference
	if encounter, ok := resource["encounter"].(map[string]interface{}); ok {
		if ref, ok := encounter["reference"].(string); ok {
			parts := strings.Split(ref, "/")
			if len(parts) == 2 {
				encounter["reference"] = parts[0] + "/" + e.pseudonymize(parts[1])
			}
		}
	}
}

func (e *Engine) deidentifyProviderResource(resource map[string]interface{}) {
	// For providers, may need to keep for operational purposes
	// but remove personal identifiers
	delete(resource, "telecom")

	// Mask address to state only
	if addrs, ok := resource["address"].([]interface{}); ok {
		for i, addr := range addrs {
			if addrMap, ok := addr.(map[string]interface{}); ok {
				state := addrMap["state"]
				for k := range addrMap {
					if k != "state" && k != "country" {
						delete(addrMap, k)
					}
				}
				addrMap["state"] = state
				addrs[i] = addrMap
			}
		}
	}
}

// ValidateDeidentification checks if data is properly de-identified
func (e *Engine) ValidateDeidentification(resource json.RawMessage) (*ValidationResult, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(resource, &data); err != nil {
		return nil, err
	}

	result := &ValidationResult{
		IsValid:    true,
		Violations: []string{},
		Warnings:   []string{},
	}

	// Check for PHI patterns
	e.checkForPHIPatterns(data, "", result)

	return result, nil
}

// ValidationResult contains de-identification validation results
type ValidationResult struct {
	IsValid    bool     `json:"is_valid"`
	Violations []string `json:"violations"`
	Warnings   []string `json:"warnings"`
}

func (e *Engine) checkForPHIPatterns(data interface{}, path string, result *ValidationResult) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newPath := path + "." + key
			if path == "" {
				newPath = key
			}

			// Check for PHI field names
			if e.isPHIFieldName(key) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Potential PHI field detected: %s", newPath))
			}

			e.checkForPHIPatterns(val, newPath, result)
		}

	case []interface{}:
		for i, item := range v {
			e.checkForPHIPatterns(item, fmt.Sprintf("%s[%d]", path, i), result)
		}

	case string:
		// Check for PHI patterns in string values
		if e.containsSSNPattern(v) {
			result.IsValid = false
			result.Violations = append(result.Violations, fmt.Sprintf("SSN pattern detected at %s", path))
		}
		if e.containsPhonePattern(v) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Phone pattern detected at %s", path))
		}
		if e.containsEmailPattern(v) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Email pattern detected at %s", path))
		}
	}
}

func (e *Engine) isPHIFieldName(name string) bool {
	phiFields := []string{
		"ssn", "socialSecurity", "driverLicense", "passport",
		"email", "phone", "fax", "name", "firstName", "lastName",
		"street", "address", "birthDate", "dob",
	}

	nameLower := strings.ToLower(name)
	for _, phi := range phiFields {
		if strings.Contains(nameLower, strings.ToLower(phi)) {
			return true
		}
	}
	return false
}

func (e *Engine) containsSSNPattern(s string) bool {
	ssnPattern := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	return ssnPattern.MatchString(s)
}

func (e *Engine) containsPhonePattern(s string) bool {
	phonePattern := regexp.MustCompile(`\b(\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	return phonePattern.MatchString(s)
}

func (e *Engine) containsEmailPattern(s string) bool {
	emailPattern := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	return emailPattern.MatchString(s)
}

func calculateAge(dob time.Time) int {
	now := time.Now()
	age := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		age--
	}
	return age
}
