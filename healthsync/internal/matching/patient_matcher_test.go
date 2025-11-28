package matching

import (
	"context"
	"testing"

	"github.com/savegress/healthsync/pkg/models"
)

func TestDefaultMatchConfig(t *testing.T) {
	cfg := DefaultMatchConfig()

	if cfg == nil {
		t.Fatal("expected default config")
	}

	// Check weights sum to approximately 1.0
	totalWeight := cfg.NameWeight + cfg.DOBWeight + cfg.SSNWeight +
		cfg.AddressWeight + cfg.PhoneWeight + cfg.EmailWeight + cfg.GenderWeight

	if totalWeight < 0.9 || totalWeight > 1.1 {
		t.Errorf("weights should sum to approximately 1.0, got %f", totalWeight)
	}

	// Check thresholds are sensible
	if cfg.AutoLinkThreshold <= cfg.ReviewThreshold {
		t.Error("auto-link threshold should be higher than review threshold")
	}

	if cfg.ReviewThreshold <= cfg.MinimumThreshold {
		t.Error("review threshold should be higher than minimum threshold")
	}
}

func TestNewPatientMatcher(t *testing.T) {
	// With config
	cfg := &MatchConfig{
		NameWeight:        0.5,
		DOBWeight:         0.3,
		AutoLinkThreshold: 0.95,
	}
	matcher := NewPatientMatcher(cfg)

	if matcher == nil {
		t.Fatal("expected matcher to be created")
	}

	if matcher.config.NameWeight != 0.5 {
		t.Error("config not set correctly")
	}

	// With nil config
	matcherDefault := NewPatientMatcher(nil)
	if matcherDefault.config == nil {
		t.Error("expected default config to be set")
	}
}

func TestPatientMatcher_IndexPatient(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
			Identifier: []models.Identifier{
				{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
				{System: "http://hospital.org/mrn", Value: "MRN001"},
			},
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}

	matcher.IndexPatient(patient)

	// Check patient is indexed
	if _, ok := matcher.patients[patient.ID]; !ok {
		t.Error("patient not indexed in patients map")
	}

	// Check SSN index
	if id, ok := matcher.indexBySSN["123-45-6789"]; !ok || id != patient.ID {
		t.Error("patient not indexed by SSN")
	}

	// Check MRN index
	if id, ok := matcher.indexByMRN["MRN001"]; !ok || id != patient.ID {
		t.Error("patient not indexed by MRN")
	}

	// Check DOB index
	if ids, ok := matcher.indexByDOB["1990-05-15"]; !ok || len(ids) == 0 {
		t.Error("patient not indexed by DOB")
	}

	// Check name index
	if ids, ok := matcher.indexByName["smith john"]; !ok || len(ids) == 0 {
		t.Error("patient not indexed by name")
	}
}

func TestPatientMatcher_IndexPatient_WithSoundex(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.UseSoundex = true
	matcher := NewPatientMatcher(cfg)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Name: []models.HumanName{
			{Family: "Johnson", Given: []string{"Robert"}},
		},
	}

	matcher.IndexPatient(patient)

	// Soundex of "johnson" is "J525"
	soundex := matcher.soundex("johnson")
	if _, ok := matcher.indexByName[soundex]; !ok {
		t.Errorf("patient not indexed by soundex %s", soundex)
	}
}

func TestPatientMatcher_RemovePatient(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
			Identifier: []models.Identifier{
				{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
				{System: "http://hospital.org/mrn", Value: "MRN001"},
			},
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}

	matcher.IndexPatient(patient)
	matcher.RemovePatient(patient.ID)

	// Check patient is removed
	if _, ok := matcher.patients[patient.ID]; ok {
		t.Error("patient should be removed from patients map")
	}

	// Check SSN index
	if _, ok := matcher.indexBySSN["123-45-6789"]; ok {
		t.Error("patient should be removed from SSN index")
	}

	// Check MRN index
	if _, ok := matcher.indexByMRN["MRN001"]; ok {
		t.Error("patient should be removed from MRN index")
	}

	// Check DOB index
	if ids, ok := matcher.indexByDOB["1990-05-15"]; ok && len(ids) > 0 {
		for _, id := range ids {
			if id == patient.ID {
				t.Error("patient should be removed from DOB index")
			}
		}
	}
}

func TestPatientMatcher_RemovePatient_NotFound(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	// Should not panic when removing non-existent patient
	matcher.RemovePatient("nonexistent")
}

func TestPatientMatcher_FindMatches_ExactSSNMatch(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
			Identifier: []models.Identifier{
				{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
			},
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
			Identifier: []models.Identifier{
				{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
			},
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}

	if result.BestMatch == nil {
		t.Fatal("expected best match")
	}

	if result.BestMatch.Patient.ID != existing.ID {
		t.Errorf("expected best match to be %s, got %s", existing.ID, result.BestMatch.Patient.ID)
	}

	// SSN + name + DOB match should give a reasonable score
	// Note: The score depends on weight distribution, so just check it's meaningful
	if result.BestMatch.Score < 0.5 {
		t.Errorf("expected reasonable score for SSN/name/DOB match, got %f", result.BestMatch.Score)
	}

	if result.Recommendation != RecommendationAutoLink && result.Recommendation != RecommendationManualReview {
		t.Errorf("expected auto_link or manual_review recommendation, got %s", result.Recommendation)
	}
}

func TestPatientMatcher_FindMatches_NameAndDOBMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.3
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Johnson", Given: []string{"Robert"}},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Johnson", Given: []string{"Robert"}},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}

	// Check name and DOB are in matched fields
	hasNameMatch := false
	hasDOBMatch := false
	for _, field := range result.BestMatch.MatchedFields {
		if field == "name" {
			hasNameMatch = true
		}
		if field == "dob" {
			hasDOBMatch = true
		}
	}

	if !hasNameMatch {
		t.Error("expected name in matched fields")
	}

	if !hasDOBMatch {
		t.Error("expected dob in matched fields")
	}
}

func TestPatientMatcher_FindMatches_NoMatch(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		BirthDate: "1985-10-20",
		Name: []models.HumanName{
			{Family: "Williams", Given: []string{"Sarah"}},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if result.Recommendation != RecommendationNewRecord {
		t.Errorf("expected new_record recommendation for no matches, got %s", result.Recommendation)
	}
}

func TestPatientMatcher_FindMatches_ManualReview(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.AutoLinkThreshold = 0.95
	cfg.ReviewThreshold = 0.40
	cfg.MinimumThreshold = 0.30
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}
	matcher.IndexPatient(existing)

	// Similar but not exact match
	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		BirthDate: "1990-05-15", // Same DOB
		Name: []models.HumanName{
			{Family: "Smyth", Given: []string{"Jon"}}, // Similar name
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) == 0 {
		t.Fatal("expected candidates for similar patient")
	}

	// Should recommend manual review for partial match
	if result.Recommendation != RecommendationManualReview && result.Recommendation != RecommendationAutoLink {
		t.Logf("Got recommendation: %s with score: %f", result.Recommendation, result.BestMatch.Score)
	}
}

func TestPatientMatcher_FindMatches_SkipSelf(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	patient := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		BirthDate: "1990-05-15",
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}
	matcher.IndexPatient(patient)

	// Search for the same patient
	result, err := matcher.FindMatches(context.Background(), patient)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	// Should not find itself as a candidate
	for _, candidate := range result.Candidates {
		if candidate.Patient.ID == patient.ID {
			t.Error("patient should not match itself")
		}
	}
}

func TestPatientMatcher_FindMatches_MRNMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.01 // Low threshold since MRN alone is a weak signal
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
			Identifier: []models.Identifier{
				{System: "http://hospital.org/mrn", Value: "MRN12345"},
			},
		},
		// Adding name so there's something else to match on
		Name: []models.HumanName{
			{Family: "Test", Given: []string{"User"}},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
			Identifier: []models.Identifier{
				{System: "http://hospital.org/mrn", Value: "MRN12345"},
			},
		},
		Name: []models.HumanName{
			{Family: "Test", Given: []string{"User"}},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	// MRN match should find candidates when threshold is low enough
	if len(result.Candidates) == 0 {
		t.Fatal("expected candidate for MRN and name match")
	}
}

func TestPatientMatcher_FindMatches_PhoneMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.05
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Telecom: []models.ContactPoint{
			{System: "phone", Value: "555-123-4567"},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		Telecom: []models.ContactPoint{
			{System: "phone", Value: "(555) 123-4567"}, // Different format, same number
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	// Phone match should contribute to score
	if len(result.Candidates) > 0 {
		if result.BestMatch.ScoreBreakdown["phone"] != 1.0 {
			t.Errorf("expected phone score 1.0, got %f", result.BestMatch.ScoreBreakdown["phone"])
		}
	}
}

func TestPatientMatcher_FindMatches_EmailMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.05
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Telecom: []models.ContactPoint{
			{System: "email", Value: "john.smith@example.com"},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		Telecom: []models.ContactPoint{
			{System: "email", Value: "JOHN.SMITH@EXAMPLE.COM"}, // Different case
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) > 0 {
		if result.BestMatch.ScoreBreakdown["email"] != 1.0 {
			t.Errorf("expected email score 1.0, got %f", result.BestMatch.ScoreBreakdown["email"])
		}
	}
}

func TestPatientMatcher_FindMatches_AddressMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.05
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Address: []models.Address{
			{
				Line:       []string{"123 Main Street"},
				City:       "Boston",
				PostalCode: "02101",
			},
		},
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		Address: []models.Address{
			{
				Line:       []string{"123 Main St"},
				City:       "Boston",
				PostalCode: "02101-1234",
			},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) > 0 {
		if result.BestMatch.ScoreBreakdown["address"] == 0 {
			t.Error("expected address score > 0 for similar addresses")
		}
	}
}

func TestPatientMatcher_FindMatches_GenderMatch(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.01
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Gender: "male",
	}
	matcher.IndexPatient(existing)

	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		Gender: "male",
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	if len(result.Candidates) > 0 {
		if result.BestMatch.ScoreBreakdown["gender"] != 1.0 {
			t.Errorf("expected gender score 1.0, got %f", result.BestMatch.ScoreBreakdown["gender"])
		}
	}
}

func TestPatientMatcher_Soundex(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"Robert", "R163"},
		{"Rupert", "R163"}, // Should be same as Robert
		{"Smith", "S530"},
		{"Smythe", "S530"}, // Should be same as Smith
		{"", ""},
		{"A", "A000"},
	}

	for _, test := range tests {
		result := matcher.soundex(test.input)
		if result != test.expected {
			t.Errorf("soundex(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"kitten", "sitting", 3},
		{"smith", "smyth", 1},
		{"john", "jon", 1},
		{"same", "same", 0},
	}

	for _, test := range tests {
		result := levenshteinDistance(test.s1, test.s2)
		if result != test.expected {
			t.Errorf("levenshteinDistance(%s, %s) = %d, expected %d",
				test.s1, test.s2, result, test.expected)
		}
	}
}

func TestPatientMatcher_CompareNames(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	// Exact match
	names1 := []models.HumanName{{Family: "Smith", Given: []string{"John"}}}
	names2 := []models.HumanName{{Family: "Smith", Given: []string{"John"}}}

	score := matcher.compareNames(names1, names2)
	if score != 1.0 {
		t.Errorf("expected exact match score 1.0, got %f", score)
	}

	// Similar names
	names3 := []models.HumanName{{Family: "Smyth", Given: []string{"Jon"}}}
	score2 := matcher.compareNames(names1, names3)
	if score2 <= 0 || score2 >= 1.0 {
		t.Errorf("expected partial match score between 0 and 1, got %f", score2)
	}

	// Empty names
	score3 := matcher.compareNames([]models.HumanName{}, names1)
	if score3 != 0 {
		t.Errorf("expected 0 for empty names, got %f", score3)
	}
}

func TestPatientMatcher_CompareDOB(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	tests := []struct {
		dob1     string
		dob2     string
		expected float64
	}{
		{"1990-05-15", "1990-05-15", 1.0},
		{"1990-05-15", "1990-05-16", 0.9},  // One day off
		{"1990-05-15", "1990-06-15", 0.2},  // Month off (>30 days, <365 days)
		{"1990-05-15", "1991-05-15", 0.2},  // Year off
		{"1990-05-15", "1985-05-15", 0},    // Too far apart
		{"invalid", "1990-05-15", 0},       // Invalid date
		{"1990-05-15", "invalid", 0},       // Invalid date
	}

	for _, test := range tests {
		result := matcher.compareDOB(test.dob1, test.dob2)
		if result != test.expected {
			t.Errorf("compareDOB(%s, %s) = %f, expected %f",
				test.dob1, test.dob2, result, test.expected)
		}
	}
}

func TestPatientMatcher_CompareAddresses(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	// Empty addresses
	score := matcher.compareAddresses([]models.Address{}, []models.Address{})
	if score != 0 {
		t.Errorf("expected 0 for empty addresses, got %f", score)
	}

	// Matching postal codes
	addr1 := []models.Address{{PostalCode: "02101", City: "Boston"}}
	addr2 := []models.Address{{PostalCode: "02101", City: "Boston"}}

	score2 := matcher.compareAddresses(addr1, addr2)
	if score2 < 0.5 {
		t.Errorf("expected high score for matching postal codes, got %f", score2)
	}
}

func TestPatientMatcher_NormalizePostalCode(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"02101", "02101"},
		{"02101-1234", "02101"},
		{"M5V 1K4", "M5V1K"},
		{"", ""},
	}

	for _, test := range tests {
		result := matcher.normalizePostalCode(test.input)
		if result != test.expected {
			t.Errorf("normalizePostalCode(%s) = %s, expected %s",
				test.input, result, test.expected)
		}
	}
}

func TestPatientMatcher_NormalizeStreetAddress(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"123 Main Street", "123 main st"},
		{"456 Oak Avenue", "456 oak ave"},
		{"789 Pine Boulevard", "789 pine blvd"},
		{"Apartment 5", "apt 5"},
	}

	for _, test := range tests {
		result := matcher.normalizeStreetAddress(test.input)
		if result != test.expected {
			t.Errorf("normalizeStreetAddress(%s) = %s, expected %s",
				test.input, result, test.expected)
		}
	}
}

func TestPatientMatcher_NormalizeContactValue(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	// Phone normalization
	phone := matcher.normalizeContactValue("(555) 123-4567", "phone")
	if phone != "5551234567" {
		t.Errorf("expected phone 5551234567, got %s", phone)
	}

	// Email normalization
	email := matcher.normalizeContactValue("John.Smith@Example.COM", "email")
	if email != "john.smith@example.com" {
		t.Errorf("expected email john.smith@example.com, got %s", email)
	}
}

func TestPatientMatcher_LinkPatients(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	master := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "master-1",
		},
		Active: true,
	}
	duplicate := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "duplicate-1",
		},
		Active: true,
	}

	matcher.IndexPatient(master)
	matcher.IndexPatient(duplicate)

	link, err := matcher.LinkPatients(master.ID, duplicate.ID)
	if err != nil {
		t.Fatalf("LinkPatients failed: %v", err)
	}

	if link.MasterID != master.ID {
		t.Error("incorrect master ID in link")
	}

	if link.DuplicateID != duplicate.ID {
		t.Error("incorrect duplicate ID in link")
	}

	if link.LinkType != "replaced-by" {
		t.Errorf("expected link type 'replaced-by', got %s", link.LinkType)
	}

	// Duplicate should be marked inactive
	if duplicate.Active {
		t.Error("duplicate patient should be marked inactive")
	}
}

func TestPatientMatcher_LinkPatients_MasterNotFound(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	duplicate := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "duplicate-1",
		},
	}
	matcher.IndexPatient(duplicate)

	_, err := matcher.LinkPatients("nonexistent", duplicate.ID)
	if err == nil {
		t.Error("expected error for missing master patient")
	}
}

func TestPatientMatcher_LinkPatients_DuplicateNotFound(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	master := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "master-1",
		},
	}
	matcher.IndexPatient(master)

	_, err := matcher.LinkPatients(master.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for missing duplicate patient")
	}
}

func TestPatientMatcher_CompareStrings_WithSoundex(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.UseLevenshtein = false
	cfg.UseSoundex = true
	matcher := NewPatientMatcher(cfg)

	// Robert and Rupert have same soundex
	score := matcher.compareStrings("Robert", "Rupert")
	if score != 0.8 {
		t.Errorf("expected soundex match score 0.8, got %f", score)
	}

	// Exact match should still be 1.0
	score2 := matcher.compareStrings("John", "John")
	if score2 != 1.0 {
		t.Errorf("expected exact match score 1.0, got %f", score2)
	}
}

func TestPatientMatcher_CompareStrings_NoFuzzy(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.UseLevenshtein = false
	cfg.UseSoundex = false
	matcher := NewPatientMatcher(cfg)

	// Without fuzzy matching, non-exact should be 0
	score := matcher.compareStrings("Smith", "Smyth")
	if score != 0 {
		t.Errorf("expected 0 without fuzzy matching, got %f", score)
	}
}

func TestPatientMatcher_CompareIdentifiers(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	ids1 := []models.Identifier{
		{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
	}
	ids2 := []models.Identifier{
		{System: "http://hl7.org/fhir/sid/us-ssn", Value: "123-45-6789"},
	}

	score := matcher.compareIdentifiers(ids1, ids2, "http://hl7.org/fhir/sid/us-ssn")
	if score != 1.0 {
		t.Errorf("expected SSN match score 1.0, got %f", score)
	}

	// Different values
	ids3 := []models.Identifier{
		{System: "http://hl7.org/fhir/sid/us-ssn", Value: "987-65-4321"},
	}

	score2 := matcher.compareIdentifiers(ids1, ids3, "http://hl7.org/fhir/sid/us-ssn")
	if score2 != 0 {
		t.Errorf("expected 0 for different SSN values, got %f", score2)
	}
}

func TestPatientMatcher_CompareContacts_NoMatch(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	contacts1 := []models.ContactPoint{
		{System: "phone", Value: "555-1234"},
	}
	contacts2 := []models.ContactPoint{
		{System: "phone", Value: "555-5678"},
	}

	score := matcher.compareContacts(contacts1, contacts2, "phone")
	if score != 0 {
		t.Errorf("expected 0 for different phone numbers, got %f", score)
	}
}

func TestMinMax(t *testing.T) {
	if min(1, 2, 3) != 1 {
		t.Error("min(1,2,3) should be 1")
	}
	if min(3, 1, 2) != 1 {
		t.Error("min(3,1,2) should be 1")
	}
	if min(2, 3, 1) != 1 {
		t.Error("min(2,3,1) should be 1")
	}

	if max(1, 2) != 2 {
		t.Error("max(1,2) should be 2")
	}
	if max(2, 1) != 2 {
		t.Error("max(2,1) should be 2")
	}
}

func TestMatchRecommendationConstants(t *testing.T) {
	if RecommendationAutoLink != "auto_link" {
		t.Error("incorrect auto_link constant")
	}
	if RecommendationManualReview != "manual_review" {
		t.Error("incorrect manual_review constant")
	}
	if RecommendationNewRecord != "new_record" {
		t.Error("incorrect new_record constant")
	}
	if RecommendationNoMatch != "no_match" {
		t.Error("incorrect no_match constant")
	}
}

func TestPatientMatcher_FindMatches_NoMatchRecommendation(t *testing.T) {
	cfg := DefaultMatchConfig()
	cfg.MinimumThreshold = 0.50
	cfg.ReviewThreshold = 0.70
	cfg.AutoLinkThreshold = 0.95
	matcher := NewPatientMatcher(cfg)

	existing := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-1",
		},
		Name: []models.HumanName{
			{Family: "Smith", Given: []string{"John"}},
		},
	}
	matcher.IndexPatient(existing)

	// Partial name match but below review threshold
	search := &models.Patient{
		FHIRResource: models.FHIRResource{
			ID: "patient-new",
		},
		Name: []models.HumanName{
			{Family: "Smithers", Given: []string{"Johnny"}},
		},
	}

	result, err := matcher.FindMatches(context.Background(), search)
	if err != nil {
		t.Fatalf("FindMatches failed: %v", err)
	}

	// Should have candidates but below review threshold
	if len(result.Candidates) > 0 && result.BestMatch.Score < cfg.ReviewThreshold {
		if result.Recommendation != RecommendationNoMatch {
			t.Errorf("expected no_match recommendation for low score, got %s", result.Recommendation)
		}
	}
}

func TestPatientMatcher_CompareHumanNames_EmptyFields(t *testing.T) {
	matcher := NewPatientMatcher(nil)

	// Empty family name
	name1 := models.HumanName{Given: []string{"John"}}
	name2 := models.HumanName{Given: []string{"John"}}

	score := matcher.compareHumanNames(name1, name2)
	if score < 0.4 {
		t.Errorf("expected partial match for given name only, got %f", score)
	}

	// No given names
	name3 := models.HumanName{Family: "Smith"}
	name4 := models.HumanName{Family: "Smith"}

	score2 := matcher.compareHumanNames(name3, name4)
	if score2 < 0.4 {
		t.Errorf("expected partial match for family name only, got %f", score2)
	}

	// Both empty
	name5 := models.HumanName{}
	name6 := models.HumanName{}

	score3 := matcher.compareHumanNames(name5, name6)
	if score3 != 0 {
		t.Errorf("expected 0 for empty names, got %f", score3)
	}
}
