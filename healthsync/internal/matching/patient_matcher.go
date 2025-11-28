package matching

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/savegress/healthsync/pkg/models"
)

// PatientMatcher handles patient matching and deduplication
type PatientMatcher struct {
	config       *MatchConfig
	indexByName  map[string][]string // normalized name -> patient IDs
	indexByDOB   map[string][]string // DOB -> patient IDs
	indexBySSN   map[string]string   // SSN -> patient ID
	indexByMRN   map[string]string   // MRN -> patient ID
	patients     map[string]*models.Patient
	mu           sync.RWMutex
}

// MatchConfig holds patient matching configuration
type MatchConfig struct {
	// Weight for each attribute (0-1)
	NameWeight        float64 `json:"name_weight"`
	DOBWeight         float64 `json:"dob_weight"`
	SSNWeight         float64 `json:"ssn_weight"`
	AddressWeight     float64 `json:"address_weight"`
	PhoneWeight       float64 `json:"phone_weight"`
	EmailWeight       float64 `json:"email_weight"`
	GenderWeight      float64 `json:"gender_weight"`

	// Thresholds
	AutoLinkThreshold   float64 `json:"auto_link_threshold"`   // Score for automatic linking
	ReviewThreshold     float64 `json:"review_threshold"`       // Score requiring manual review
	MinimumThreshold    float64 `json:"minimum_threshold"`      // Minimum score to consider

	// Matching options
	UseSoundex         bool `json:"use_soundex"`
	UseMetaphone       bool `json:"use_metaphone"`
	UseLevenshtein     bool `json:"use_levenshtein"`
	FuzzyNameMatching  bool `json:"fuzzy_name_matching"`
}

// MatchResult represents the result of a patient match operation
type MatchResult struct {
	SearchPatient  *models.Patient   `json:"search_patient"`
	Candidates     []MatchCandidate  `json:"candidates"`
	BestMatch      *MatchCandidate   `json:"best_match,omitempty"`
	Recommendation MatchRecommendation `json:"recommendation"`
	Timestamp      time.Time         `json:"timestamp"`
}

// MatchCandidate represents a potential patient match
type MatchCandidate struct {
	Patient       *models.Patient   `json:"patient"`
	Score         float64           `json:"score"`
	ScoreBreakdown map[string]float64 `json:"score_breakdown"`
	MatchedFields []string          `json:"matched_fields"`
}

// MatchRecommendation represents the recommended action
type MatchRecommendation string

const (
	RecommendationAutoLink    MatchRecommendation = "auto_link"
	RecommendationManualReview MatchRecommendation = "manual_review"
	RecommendationNewRecord   MatchRecommendation = "new_record"
	RecommendationNoMatch     MatchRecommendation = "no_match"
)

// NewPatientMatcher creates a new patient matcher
func NewPatientMatcher(config *MatchConfig) *PatientMatcher {
	if config == nil {
		config = DefaultMatchConfig()
	}

	return &PatientMatcher{
		config:      config,
		indexByName: make(map[string][]string),
		indexByDOB:  make(map[string][]string),
		indexBySSN:  make(map[string]string),
		indexByMRN:  make(map[string]string),
		patients:    make(map[string]*models.Patient),
	}
}

// DefaultMatchConfig returns a default configuration
func DefaultMatchConfig() *MatchConfig {
	return &MatchConfig{
		NameWeight:        0.30,
		DOBWeight:         0.25,
		SSNWeight:         0.20,
		AddressWeight:     0.10,
		PhoneWeight:       0.08,
		EmailWeight:       0.05,
		GenderWeight:      0.02,
		AutoLinkThreshold: 0.95,
		ReviewThreshold:   0.70,
		MinimumThreshold:  0.50,
		UseSoundex:        true,
		UseLevenshtein:    true,
		FuzzyNameMatching: true,
	}
}

// IndexPatient adds a patient to the matching index
func (m *PatientMatcher) IndexPatient(patient *models.Patient) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.patients[patient.ID] = patient

	// Index by name
	for _, name := range patient.Name {
		normalized := m.normalizeName(name)
		m.indexByName[normalized] = append(m.indexByName[normalized], patient.ID)

		// Also index soundex versions
		if m.config.UseSoundex {
			for _, part := range strings.Fields(normalized) {
				soundex := m.soundex(part)
				m.indexByName[soundex] = append(m.indexByName[soundex], patient.ID)
			}
		}
	}

	// Index by DOB
	if patient.BirthDate != "" {
		m.indexByDOB[patient.BirthDate] = append(m.indexByDOB[patient.BirthDate], patient.ID)
	}

	// Index by SSN
	for _, identifier := range patient.Identifier {
		if identifier.System == "http://hl7.org/fhir/sid/us-ssn" {
			m.indexBySSN[identifier.Value] = patient.ID
		}
		if identifier.System == "http://hospital.example.org/mrn" || strings.Contains(identifier.System, "mrn") {
			m.indexByMRN[identifier.Value] = patient.ID
		}
	}
}

// RemovePatient removes a patient from the index
func (m *PatientMatcher) RemovePatient(patientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	patient, ok := m.patients[patientID]
	if !ok {
		return
	}

	delete(m.patients, patientID)

	// Remove from name index
	for _, name := range patient.Name {
		normalized := m.normalizeName(name)
		m.removeFromIndex(m.indexByName, normalized, patientID)
	}

	// Remove from DOB index
	if patient.BirthDate != "" {
		m.removeFromIndex(m.indexByDOB, patient.BirthDate, patientID)
	}

	// Remove from SSN/MRN index
	for _, identifier := range patient.Identifier {
		if identifier.System == "http://hl7.org/fhir/sid/us-ssn" {
			delete(m.indexBySSN, identifier.Value)
		}
		if strings.Contains(identifier.System, "mrn") {
			delete(m.indexByMRN, identifier.Value)
		}
	}
}

func (m *PatientMatcher) removeFromIndex(index map[string][]string, key, patientID string) {
	ids := index[key]
	for i, id := range ids {
		if id == patientID {
			index[key] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	if len(index[key]) == 0 {
		delete(index, key)
	}
}

// FindMatches finds potential matches for a patient
func (m *PatientMatcher) FindMatches(ctx context.Context, patient *models.Patient) (*MatchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &MatchResult{
		SearchPatient: patient,
		Candidates:    []MatchCandidate{},
		Timestamp:     time.Now(),
	}

	// Collect candidate IDs
	candidateIDs := make(map[string]bool)

	// Check deterministic identifiers first (SSN, MRN)
	for _, identifier := range patient.Identifier {
		if identifier.System == "http://hl7.org/fhir/sid/us-ssn" {
			if id, ok := m.indexBySSN[identifier.Value]; ok {
				candidateIDs[id] = true
			}
		}
		if strings.Contains(identifier.System, "mrn") {
			if id, ok := m.indexByMRN[identifier.Value]; ok {
				candidateIDs[id] = true
			}
		}
	}

	// Find candidates by name
	for _, name := range patient.Name {
		normalized := m.normalizeName(name)
		if ids, ok := m.indexByName[normalized]; ok {
			for _, id := range ids {
				candidateIDs[id] = true
			}
		}

		// Soundex matching
		if m.config.UseSoundex {
			for _, part := range strings.Fields(normalized) {
				soundex := m.soundex(part)
				if ids, ok := m.indexByName[soundex]; ok {
					for _, id := range ids {
						candidateIDs[id] = true
					}
				}
			}
		}
	}

	// Find candidates by DOB
	if patient.BirthDate != "" {
		if ids, ok := m.indexByDOB[patient.BirthDate]; ok {
			for _, id := range ids {
				candidateIDs[id] = true
			}
		}
	}

	// Score each candidate
	for id := range candidateIDs {
		if id == patient.ID {
			continue // Skip self
		}

		candidate := m.patients[id]
		if candidate == nil {
			continue
		}

		score, breakdown := m.calculateScore(patient, candidate)

		if score >= m.config.MinimumThreshold {
			matchedFields := m.getMatchedFields(breakdown)
			result.Candidates = append(result.Candidates, MatchCandidate{
				Patient:        candidate,
				Score:          score,
				ScoreBreakdown: breakdown,
				MatchedFields:  matchedFields,
			})
		}
	}

	// Sort by score descending
	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].Score > result.Candidates[j].Score
	})

	// Determine recommendation
	if len(result.Candidates) == 0 {
		result.Recommendation = RecommendationNewRecord
	} else {
		bestScore := result.Candidates[0].Score
		result.BestMatch = &result.Candidates[0]

		if bestScore >= m.config.AutoLinkThreshold {
			result.Recommendation = RecommendationAutoLink
		} else if bestScore >= m.config.ReviewThreshold {
			result.Recommendation = RecommendationManualReview
		} else {
			result.Recommendation = RecommendationNoMatch
		}
	}

	return result, nil
}

func (m *PatientMatcher) calculateScore(patient1, patient2 *models.Patient) (float64, map[string]float64) {
	breakdown := make(map[string]float64)
	var totalScore float64

	// Name matching
	nameScore := m.compareNames(patient1.Name, patient2.Name)
	breakdown["name"] = nameScore
	totalScore += nameScore * m.config.NameWeight

	// DOB matching
	if patient1.BirthDate != "" && patient2.BirthDate != "" {
		if patient1.BirthDate == patient2.BirthDate {
			breakdown["dob"] = 1.0
			totalScore += m.config.DOBWeight
		} else {
			// Partial match for close dates
			dobScore := m.compareDOB(patient1.BirthDate, patient2.BirthDate)
			breakdown["dob"] = dobScore
			totalScore += dobScore * m.config.DOBWeight
		}
	}

	// SSN matching
	ssnScore := m.compareIdentifiers(patient1.Identifier, patient2.Identifier, "http://hl7.org/fhir/sid/us-ssn")
	breakdown["ssn"] = ssnScore
	totalScore += ssnScore * m.config.SSNWeight

	// Address matching
	addressScore := m.compareAddresses(patient1.Address, patient2.Address)
	breakdown["address"] = addressScore
	totalScore += addressScore * m.config.AddressWeight

	// Phone matching
	phoneScore := m.compareContacts(patient1.Telecom, patient2.Telecom, "phone")
	breakdown["phone"] = phoneScore
	totalScore += phoneScore * m.config.PhoneWeight

	// Email matching
	emailScore := m.compareContacts(patient1.Telecom, patient2.Telecom, "email")
	breakdown["email"] = emailScore
	totalScore += emailScore * m.config.EmailWeight

	// Gender matching
	if patient1.Gender != "" && patient2.Gender != "" {
		if patient1.Gender == patient2.Gender {
			breakdown["gender"] = 1.0
			totalScore += m.config.GenderWeight
		}
	}

	return totalScore, breakdown
}

func (m *PatientMatcher) compareNames(names1, names2 []models.HumanName) float64 {
	if len(names1) == 0 || len(names2) == 0 {
		return 0
	}

	var maxScore float64

	for _, name1 := range names1 {
		for _, name2 := range names2 {
			score := m.compareHumanNames(name1, name2)
			if score > maxScore {
				maxScore = score
			}
		}
	}

	return maxScore
}

func (m *PatientMatcher) compareHumanNames(name1, name2 models.HumanName) float64 {
	var scores []float64

	// Compare family name
	if name1.Family != "" && name2.Family != "" {
		familyScore := m.compareStrings(name1.Family, name2.Family)
		scores = append(scores, familyScore*0.5)
	}

	// Compare given names
	if len(name1.Given) > 0 && len(name2.Given) > 0 {
		givenScore := m.compareStrings(name1.Given[0], name2.Given[0])
		scores = append(scores, givenScore*0.5)
	}

	if len(scores) == 0 {
		return 0
	}

	var total float64
	for _, s := range scores {
		total += s
	}
	return total
}

func (m *PatientMatcher) compareStrings(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	if m.config.UseLevenshtein {
		distance := levenshteinDistance(s1, s2)
		maxLen := max(len(s1), len(s2))
		if maxLen == 0 {
			return 0
		}
		return 1.0 - float64(distance)/float64(maxLen)
	}

	if m.config.UseSoundex && m.soundex(s1) == m.soundex(s2) {
		return 0.8
	}

	return 0
}

func (m *PatientMatcher) compareDOB(dob1, dob2 string) float64 {
	// Parse dates (assuming YYYY-MM-DD format)
	t1, err1 := time.Parse("2006-01-02", dob1)
	t2, err2 := time.Parse("2006-01-02", dob2)

	if err1 != nil || err2 != nil {
		return 0
	}

	// Calculate day difference
	diff := t1.Sub(t2)
	days := int(diff.Hours() / 24)
	if days < 0 {
		days = -days
	}

	// Allow for potential data entry errors
	if days == 0 {
		return 1.0
	} else if days <= 1 {
		return 0.9 // One day off (typo)
	} else if days <= 30 {
		return 0.5 // Within a month
	} else if days <= 365 {
		return 0.2 // Within a year
	}

	return 0
}

func (m *PatientMatcher) compareIdentifiers(ids1, ids2 []models.Identifier, system string) float64 {
	for _, id1 := range ids1 {
		if id1.System == system && id1.Value != "" {
			for _, id2 := range ids2 {
				if id2.System == system && id1.Value == id2.Value {
					return 1.0
				}
			}
		}
	}
	return 0
}

func (m *PatientMatcher) compareAddresses(addrs1, addrs2 []models.Address) float64 {
	if len(addrs1) == 0 || len(addrs2) == 0 {
		return 0
	}

	var maxScore float64

	for _, addr1 := range addrs1 {
		for _, addr2 := range addrs2 {
			score := m.compareAddress(addr1, addr2)
			if score > maxScore {
				maxScore = score
			}
		}
	}

	return maxScore
}

func (m *PatientMatcher) compareAddress(addr1, addr2 models.Address) float64 {
	var scores []float64
	var weights []float64

	// Postal code (most reliable)
	if addr1.PostalCode != "" && addr2.PostalCode != "" {
		if m.normalizePostalCode(addr1.PostalCode) == m.normalizePostalCode(addr2.PostalCode) {
			scores = append(scores, 1.0)
		} else {
			scores = append(scores, 0)
		}
		weights = append(weights, 0.4)
	}

	// City
	if addr1.City != "" && addr2.City != "" {
		scores = append(scores, m.compareStrings(addr1.City, addr2.City))
		weights = append(weights, 0.2)
	}

	// Street address
	if len(addr1.Line) > 0 && len(addr2.Line) > 0 {
		lineScore := m.compareStrings(m.normalizeStreetAddress(addr1.Line[0]), m.normalizeStreetAddress(addr2.Line[0]))
		scores = append(scores, lineScore)
		weights = append(weights, 0.4)
	}

	if len(scores) == 0 {
		return 0
	}

	var totalScore, totalWeight float64
	for i := range scores {
		totalScore += scores[i] * weights[i]
		totalWeight += weights[i]
	}

	return totalScore / totalWeight
}

func (m *PatientMatcher) compareContacts(contacts1, contacts2 []models.ContactPoint, system string) float64 {
	for _, c1 := range contacts1 {
		if c1.System == system && c1.Value != "" {
			for _, c2 := range contacts2 {
				if c2.System == system {
					normalized1 := m.normalizeContactValue(c1.Value, system)
					normalized2 := m.normalizeContactValue(c2.Value, system)
					if normalized1 == normalized2 {
						return 1.0
					}
				}
			}
		}
	}
	return 0
}

func (m *PatientMatcher) normalizeName(name models.HumanName) string {
	parts := []string{}
	if name.Family != "" {
		parts = append(parts, strings.ToLower(name.Family))
	}
	for _, given := range name.Given {
		parts = append(parts, strings.ToLower(given))
	}
	return strings.Join(parts, " ")
}

func (m *PatientMatcher) normalizePostalCode(pc string) string {
	// Remove non-alphanumeric characters and take first 5 digits
	var result strings.Builder
	for _, r := range pc {
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			result.WriteRune(r)
		}
	}
	s := result.String()
	if len(s) > 5 {
		return s[:5]
	}
	return s
}

func (m *PatientMatcher) normalizeStreetAddress(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))

	// Standardize common abbreviations
	replacements := map[string]string{
		"street":    "st",
		"avenue":    "ave",
		"boulevard": "blvd",
		"drive":     "dr",
		"road":      "rd",
		"lane":      "ln",
		"court":     "ct",
		"place":     "pl",
		"apartment": "apt",
		"suite":     "ste",
	}

	for full, abbr := range replacements {
		addr = strings.ReplaceAll(addr, full, abbr)
	}

	return addr
}

func (m *PatientMatcher) normalizeContactValue(value, system string) string {
	if system == "phone" {
		// Remove all non-digits
		re := regexp.MustCompile(`\D`)
		return re.ReplaceAllString(value, "")
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func (m *PatientMatcher) getMatchedFields(breakdown map[string]float64) []string {
	var matched []string
	for field, score := range breakdown {
		if score > 0 {
			matched = append(matched, field)
		}
	}
	return matched
}

// soundex implements the Soundex algorithm for phonetic matching
func (m *PatientMatcher) soundex(s string) string {
	if s == "" {
		return ""
	}

	s = strings.ToUpper(strings.TrimSpace(s))

	// Keep first letter
	result := string(s[0])

	// Map letters to soundex codes
	codes := map[byte]byte{
		'B': '1', 'F': '1', 'P': '1', 'V': '1',
		'C': '2', 'G': '2', 'J': '2', 'K': '2', 'Q': '2', 'S': '2', 'X': '2', 'Z': '2',
		'D': '3', 'T': '3',
		'L': '4',
		'M': '5', 'N': '5',
		'R': '6',
	}

	prevCode := byte('0')
	for i := 1; i < len(s) && len(result) < 4; i++ {
		if code, ok := codes[s[i]]; ok {
			if code != prevCode {
				result += string(code)
				prevCode = code
			}
		} else {
			prevCode = '0'
		}
	}

	// Pad with zeros
	for len(result) < 4 {
		result += "0"
	}

	return result
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	d := make([][]int, len(s1)+1)
	for i := range d {
		d[i] = make([]int, len(s2)+1)
	}

	// Initialize
	for i := 0; i <= len(s1); i++ {
		d[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		d[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return d[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// LinkPatients creates a link between two patient records
func (m *PatientMatcher) LinkPatients(masterID, duplicateID string) (*PatientLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	master, ok := m.patients[masterID]
	if !ok {
		return nil, fmt.Errorf("master patient not found: %s", masterID)
	}

	duplicate, ok := m.patients[duplicateID]
	if !ok {
		return nil, fmt.Errorf("duplicate patient not found: %s", duplicateID)
	}

	link := &PatientLink{
		ID:          fmt.Sprintf("link_%d", time.Now().UnixNano()),
		MasterID:    masterID,
		DuplicateID: duplicateID,
		LinkType:    "replaced-by",
		CreatedAt:   time.Now(),
	}

	// Mark duplicate as inactive
	duplicate.Active = false

	return link, nil
}

// PatientLink represents a link between patient records
type PatientLink struct {
	ID          string    `json:"id"`
	MasterID    string    `json:"master_id"`
	DuplicateID string    `json:"duplicate_id"`
	LinkType    string    `json:"link_type"` // replaced-by, refer, seealso
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by,omitempty"`
}
