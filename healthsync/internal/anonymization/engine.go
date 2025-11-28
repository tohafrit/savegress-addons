package anonymization

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

// Engine handles PHI de-identification and anonymization
type Engine struct {
	config      *Config
	dateShift   map[string]int // Patient ID -> date shift in days
	pseudonyms  map[string]string // Original -> Pseudonym
	mu          sync.RWMutex
}

// Config holds anonymization configuration
type Config struct {
	Method           string   // safe_harbor, expert_determination, limited_dataset
	DateShiftRange   int      // Range of days for date shifting
	PreserveBirthYear bool    // Preserve year for ages < 90
	ZipCodeTruncation int     // Number of digits to keep (typically 3)
	Salt             string   // Salt for hashing
}

// NewEngine creates a new anonymization engine
func NewEngine(cfg *Config) *Engine {
	return &Engine{
		config:     cfg,
		dateShift:  make(map[string]int),
		pseudonyms: make(map[string]string),
	}
}

// AnonymizePatient anonymizes a Patient resource
func (e *Engine) AnonymizePatient(patient *models.Patient) *models.Patient {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Create a copy
	anonymized := *patient

	// Get or create date shift for this patient
	patientID := patient.ID
	if _, ok := e.dateShift[patientID]; !ok {
		e.dateShift[patientID] = rand.Intn(e.config.DateShiftRange*2) - e.config.DateShiftRange
	}

	switch e.config.Method {
	case "safe_harbor":
		e.applySafeHarbor(&anonymized)
	case "limited_dataset":
		e.applyLimitedDataset(&anonymized)
	default:
		e.applySafeHarbor(&anonymized)
	}

	return &anonymized
}

// applySafeHarbor applies HIPAA Safe Harbor de-identification
func (e *Engine) applySafeHarbor(patient *models.Patient) {
	patientID := patient.ID

	// 1. Names - Remove all
	patient.Name = nil

	// 2. Geographic data - Truncate to first 3 digits of zip if population > 20,000
	for i := range patient.Address {
		e.anonymizeAddress(&patient.Address[i])
	}

	// 3. Dates - Keep only year, or shift
	patient.BirthDate = e.anonymizeDate(patient.BirthDate, patientID)

	// 4. Phone numbers - Remove
	// 5. Fax numbers - Remove
	// 6. Email addresses - Remove
	patient.Telecom = nil

	// 7-17. Other identifiers - Remove
	patient.Identifier = e.anonymizeIdentifiers(patient.Identifier)

	// 18. Biometric identifiers - Handled by removing identifiers
	// Remove photos
	// patient.Photo = nil (not in our model)

	// Remove contacts
	patient.Contact = nil

	// Anonymize ID
	patient.ID = e.generatePseudonym(patient.ID)

	// Remove references
	patient.GeneralPractitioner = nil
	patient.ManagingOrganization = nil
}

// applyLimitedDataset applies Limited Dataset rules
func (e *Engine) applyLimitedDataset(patient *models.Patient) {
	patientID := patient.ID

	// Limited dataset allows:
	// - Dates (admission, discharge, service, birth, death)
	// - City, state, zip code
	// - Age in years, months, days, or hours

	// Remove direct identifiers
	patient.Name = nil
	patient.Telecom = nil
	patient.Contact = nil

	// Keep city, state, zip but remove street address
	for i := range patient.Address {
		patient.Address[i].Line = nil
		patient.Address[i].Text = ""
	}

	// Keep birth date (can include month and day)
	// But shift if configured
	if e.config.DateShiftRange > 0 {
		patient.BirthDate = e.anonymizeDate(patient.BirthDate, patientID)
	}

	// Remove identifiers except non-direct ones
	patient.Identifier = e.anonymizeIdentifiers(patient.Identifier)

	// Generate pseudonym for ID
	patient.ID = e.generatePseudonym(patient.ID)
}

func (e *Engine) anonymizeAddress(addr *models.Address) {
	// Remove street address
	addr.Line = nil
	addr.Text = ""

	// Truncate postal code to first 3 digits
	if len(addr.PostalCode) >= e.config.ZipCodeTruncation {
		addr.PostalCode = addr.PostalCode[:e.config.ZipCodeTruncation] + "00"
	}
}

func (e *Engine) anonymizeDate(dateStr string, patientID string) string {
	if dateStr == "" {
		return ""
	}

	// Parse date
	layouts := []string{"2006-01-02", "2006-01", "2006"}
	var parsedDate time.Time
	var layout string
	var err error

	for _, l := range layouts {
		parsedDate, err = time.Parse(l, dateStr)
		if err == nil {
			layout = l
			break
		}
	}

	if err != nil {
		return ""
	}

	// Check age - if 90 or older, generalize to 90+
	age := time.Since(parsedDate).Hours() / 24 / 365
	if age >= 90 {
		return "1900" // Indicates 90+ years
	}

	// Apply date shift
	shift := e.dateShift[patientID]
	shifted := parsedDate.AddDate(0, 0, shift)

	// For Safe Harbor, return only year
	if e.config.Method == "safe_harbor" {
		return shifted.Format("2006")
	}

	return shifted.Format(layout)
}

func (e *Engine) anonymizeIdentifiers(identifiers []models.Identifier) []models.Identifier {
	// Remove all identifiers for Safe Harbor
	// For limited dataset, we could keep some non-direct identifiers
	if e.config.Method == "safe_harbor" {
		return nil
	}

	// For limited dataset, keep certain types
	var kept []models.Identifier
	for _, id := range identifiers {
		// Keep accession numbers, encounter numbers (after pseudonymization)
		if id.Type != nil {
			for _, coding := range id.Type.Coding {
				if coding.Code == "ACSN" || coding.Code == "VN" {
					id.Value = e.generatePseudonym(id.Value)
					kept = append(kept, id)
				}
			}
		}
	}
	return kept
}

func (e *Engine) generatePseudonym(original string) string {
	if existing, ok := e.pseudonyms[original]; ok {
		return existing
	}

	// Generate SHA-256 hash with salt
	h := sha256.New()
	h.Write([]byte(e.config.Salt + original))
	hash := hex.EncodeToString(h.Sum(nil))

	// Use first 16 characters as pseudonym
	pseudonym := hash[:16]
	e.pseudonyms[original] = pseudonym

	return pseudonym
}

// AnonymizeObservation anonymizes an Observation resource
func (e *Engine) AnonymizeObservation(obs *models.Observation, patientID string) *models.Observation {
	e.mu.Lock()
	defer e.mu.Unlock()

	anonymized := *obs

	// Anonymize subject reference
	if anonymized.Subject != nil {
		anonymized.Subject = &models.Reference{
			Reference: "Patient/" + e.generatePseudonym(patientID),
		}
	}

	// Anonymize encounter reference
	if anonymized.Encounter != nil {
		anonymized.Encounter.Reference = e.anonymizeReference(anonymized.Encounter.Reference)
	}

	// Anonymize performer references
	anonymized.Performer = nil

	// Anonymize dates
	if anonymized.EffectiveDateTime != nil {
		shifted := e.shiftTime(*anonymized.EffectiveDateTime, patientID)
		anonymized.EffectiveDateTime = &shifted
	}

	if anonymized.Issued != nil {
		shifted := e.shiftTime(*anonymized.Issued, patientID)
		anonymized.Issued = &shifted
	}

	// Remove notes that might contain PHI
	anonymized.Note = nil

	// Anonymize ID
	anonymized.ID = e.generatePseudonym(obs.ID)

	return &anonymized
}

// AnonymizeEncounter anonymizes an Encounter resource
func (e *Engine) AnonymizeEncounter(enc *models.Encounter, patientID string) *models.Encounter {
	e.mu.Lock()
	defer e.mu.Unlock()

	anonymized := *enc

	// Anonymize subject reference
	if anonymized.Subject != nil {
		anonymized.Subject = &models.Reference{
			Reference: "Patient/" + e.generatePseudonym(patientID),
		}
	}

	// Anonymize participant references
	anonymized.Participant = nil

	// Anonymize period
	if anonymized.Period != nil {
		if anonymized.Period.Start != nil {
			shifted := e.shiftTime(*anonymized.Period.Start, patientID)
			anonymized.Period.Start = &shifted
		}
		if anonymized.Period.End != nil {
			shifted := e.shiftTime(*anonymized.Period.End, patientID)
			anonymized.Period.End = &shifted
		}
	}

	// Anonymize service provider
	anonymized.ServiceProvider = nil

	// Anonymize ID
	anonymized.ID = e.generatePseudonym(enc.ID)

	return &anonymized
}

func (e *Engine) shiftTime(t time.Time, patientID string) time.Time {
	if _, ok := e.dateShift[patientID]; !ok {
		e.dateShift[patientID] = rand.Intn(e.config.DateShiftRange*2) - e.config.DateShiftRange
	}
	return t.AddDate(0, 0, e.dateShift[patientID])
}

func (e *Engine) anonymizeReference(ref string) string {
	// Extract resource type and ID
	parts := strings.Split(ref, "/")
	if len(parts) == 2 {
		return parts[0] + "/" + e.generatePseudonym(parts[1])
	}
	return e.generatePseudonym(ref)
}

// AnonymizeJSON anonymizes a generic JSON resource
func (e *Engine) AnonymizeJSON(data []byte, resourceType models.ResourceType, patientID string) ([]byte, error) {
	var resource map[string]interface{}
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Apply anonymization rules
	e.anonymizeMap(resource, resourceType, patientID)

	return json.Marshal(resource)
}

func (e *Engine) anonymizeMap(m map[string]interface{}, resourceType models.ResourceType, patientID string) {
	// PHI fields to remove or anonymize
	phiFields := map[string]string{
		"name":              "remove",
		"telecom":           "remove",
		"address":           "truncate",
		"birthDate":         "generalize",
		"photo":             "remove",
		"contact":           "remove",
		"identifier":        "hash",
	}

	for key, value := range m {
		action, isPHI := phiFields[key]
		if isPHI {
			switch action {
			case "remove":
				delete(m, key)
			case "truncate":
				if addr, ok := value.([]interface{}); ok {
					for _, a := range addr {
						if addrMap, ok := a.(map[string]interface{}); ok {
							delete(addrMap, "line")
							delete(addrMap, "text")
							if pc, ok := addrMap["postalCode"].(string); ok && len(pc) >= 3 {
								addrMap["postalCode"] = pc[:3] + "00"
							}
						}
					}
				}
			case "generalize":
				if dateStr, ok := value.(string); ok {
					m[key] = e.anonymizeDate(dateStr, patientID)
				}
			case "hash":
				delete(m, key) // For Safe Harbor, remove identifiers
			}
		} else if nested, ok := value.(map[string]interface{}); ok {
			e.anonymizeMap(nested, resourceType, patientID)
		} else if array, ok := value.([]interface{}); ok {
			for _, item := range array {
				if itemMap, ok := item.(map[string]interface{}); ok {
					e.anonymizeMap(itemMap, resourceType, patientID)
				}
			}
		}
	}

	// Anonymize ID
	if id, ok := m["id"].(string); ok {
		m["id"] = e.generatePseudonym(id)
	}

	// Anonymize references
	for key, value := range m {
		if strings.HasSuffix(key, "Reference") || key == "subject" || key == "patient" {
			if ref, ok := value.(map[string]interface{}); ok {
				if refStr, ok := ref["reference"].(string); ok {
					ref["reference"] = e.anonymizeReference(refStr)
				}
				delete(ref, "display")
			}
		}
	}
}

// RedactText redacts PHI from free text
func (e *Engine) RedactText(text string) string {
	// Patterns for PHI
	patterns := []struct {
		pattern     string
		replacement string
	}{
		// SSN
		{`\b\d{3}-\d{2}-\d{4}\b`, "[SSN REDACTED]"},
		// Phone numbers
		{`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`, "[PHONE REDACTED]"},
		// Email addresses
		{`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "[EMAIL REDACTED]"},
		// Dates (various formats)
		{`\b\d{1,2}/\d{1,2}/\d{2,4}\b`, "[DATE REDACTED]"},
		{`\b\d{4}-\d{2}-\d{2}\b`, "[DATE REDACTED]"},
		// MRN patterns
		{`\bMRN[:\s]*\d+\b`, "[MRN REDACTED]"},
		// Age with years
		{`\b\d{1,3}\s*(?:year|yr)s?\s*old\b`, "[AGE REDACTED]"},
	}

	result := text
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		result = re.ReplaceAllString(result, p.replacement)
	}

	return result
}

// GenerateK-Anonymity checks k-anonymity for a dataset
func (e *Engine) CheckKAnonymity(records []map[string]interface{}, quasiIdentifiers []string, k int) *KAnonymityResult {
	// Group records by quasi-identifier values
	groups := make(map[string]int)

	for _, record := range records {
		var keyParts []string
		for _, qi := range quasiIdentifiers {
			if val, ok := record[qi]; ok {
				keyParts = append(keyParts, fmt.Sprintf("%v", val))
			}
		}
		key := strings.Join(keyParts, "|")
		groups[key]++
	}

	result := &KAnonymityResult{
		K:           k,
		Satisfied:   true,
		TotalGroups: len(groups),
	}

	for key, count := range groups {
		if count < k {
			result.Satisfied = false
			result.ViolatingGroups = append(result.ViolatingGroups, GroupInfo{
				QuasiIdentifiers: key,
				Count:           count,
			})
		}
	}

	return result
}

// KAnonymityResult contains k-anonymity check results
type KAnonymityResult struct {
	K               int         `json:"k"`
	Satisfied       bool        `json:"satisfied"`
	TotalGroups     int         `json:"total_groups"`
	ViolatingGroups []GroupInfo `json:"violating_groups,omitempty"`
}

// GroupInfo contains information about an equivalence class
type GroupInfo struct {
	QuasiIdentifiers string `json:"quasi_identifiers"`
	Count           int    `json:"count"`
}
