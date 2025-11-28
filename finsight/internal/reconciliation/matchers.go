package reconciliation

import (
	"math"
	"strings"
	"time"

	"github.com/savegress/finsight/pkg/models"
)

// ExactMatcher matches transactions exactly
type ExactMatcher struct{}

// NewExactMatcher creates a new exact matcher
func NewExactMatcher() *ExactMatcher {
	return &ExactMatcher{}
}

func (m *ExactMatcher) Name() string { return "exact" }

func (m *ExactMatcher) Match(source, target *models.Transaction) *MatchResult {
	result := &MatchResult{
		MatchType: "exact",
	}

	// Check external ID match
	if source.ExternalID != "" && target.ExternalID != "" {
		if source.ExternalID == target.ExternalID {
			result.Matched = true
			result.Confidence = 1.0
		}
	}

	// Check exact amount and date match
	if source.Amount.Equal(target.Amount) {
		sameDate := source.CreatedAt.Format("2006-01-02") == target.CreatedAt.Format("2006-01-02")
		if sameDate {
			result.Matched = true
			result.Confidence = 0.95
		}
	}

	// Check for any differences
	if result.Matched {
		result.Differences = m.findDifferences(source, target)
	}

	return result
}

func (m *ExactMatcher) findDifferences(source, target *models.Transaction) []Difference {
	var diffs []Difference

	if !source.Amount.Equal(target.Amount) {
		diffs = append(diffs, Difference{
			Field:    "amount",
			Source:   source.Amount.String(),
			Target:   target.Amount.String(),
			Severity: "error",
		})
	}

	if source.Currency != target.Currency {
		diffs = append(diffs, Difference{
			Field:    "currency",
			Source:   source.Currency,
			Target:   target.Currency,
			Severity: "error",
		})
	}

	if source.Status != target.Status {
		diffs = append(diffs, Difference{
			Field:    "status",
			Source:   string(source.Status),
			Target:   string(target.Status),
			Severity: "warning",
		})
	}

	return diffs
}

// FuzzyMatcher matches transactions with tolerance
type FuzzyMatcher struct {
	amountTolerance float64
	dateTolerance   time.Duration
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher(amountTolerance float64, dateTolerance time.Duration) *FuzzyMatcher {
	return &FuzzyMatcher{
		amountTolerance: amountTolerance,
		dateTolerance:   dateTolerance,
	}
}

func (m *FuzzyMatcher) Name() string { return "fuzzy" }

func (m *FuzzyMatcher) Match(source, target *models.Transaction) *MatchResult {
	result := &MatchResult{
		MatchType: "fuzzy",
	}

	// Check amount with tolerance
	sourceAmount := source.Amount.InexactFloat64()
	targetAmount := target.Amount.InexactFloat64()

	if sourceAmount == 0 {
		return result
	}

	amountDiff := math.Abs(sourceAmount-targetAmount) / sourceAmount
	amountMatch := amountDiff <= m.amountTolerance

	// Check date with tolerance
	timeDiff := source.CreatedAt.Sub(target.CreatedAt)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	dateMatch := timeDiff <= m.dateTolerance

	if amountMatch && dateMatch {
		result.Matched = true

		// Calculate confidence based on how close the match is
		amountConfidence := 1.0 - (amountDiff / m.amountTolerance)
		dateConfidence := 1.0 - (float64(timeDiff) / float64(m.dateTolerance))
		result.Confidence = (amountConfidence + dateConfidence) / 2 * 0.9 // Max 90% for fuzzy match

		// Record differences
		if !source.Amount.Equal(target.Amount) {
			result.Differences = append(result.Differences, Difference{
				Field:    "amount",
				Source:   source.Amount.String(),
				Target:   target.Amount.String(),
				Severity: "warning",
			})
		}

		if source.CreatedAt.Format("2006-01-02") != target.CreatedAt.Format("2006-01-02") {
			result.Differences = append(result.Differences, Difference{
				Field:    "date",
				Source:   source.CreatedAt.Format("2006-01-02"),
				Target:   target.CreatedAt.Format("2006-01-02"),
				Severity: "warning",
			})
		}
	}

	return result
}

// ReferenceIDMatcher matches transactions by reference IDs
type ReferenceIDMatcher struct{}

// NewReferenceIDMatcher creates a new reference ID matcher
func NewReferenceIDMatcher() *ReferenceIDMatcher {
	return &ReferenceIDMatcher{}
}

func (m *ReferenceIDMatcher) Name() string { return "reference" }

func (m *ReferenceIDMatcher) Match(source, target *models.Transaction) *MatchResult {
	result := &MatchResult{
		MatchType: "reference",
	}

	// Check external ID
	if source.ExternalID != "" && target.ExternalID != "" {
		if source.ExternalID == target.ExternalID {
			result.Matched = true
			result.Confidence = 0.98
			result.Differences = m.findDifferences(source, target)
			return result
		}
	}

	// Check if IDs are contained in each other
	if source.ID != "" && target.ID != "" {
		if strings.Contains(target.ID, source.ID) || strings.Contains(source.ID, target.ID) {
			result.Matched = true
			result.Confidence = 0.85
			result.Differences = m.findDifferences(source, target)
			return result
		}
	}

	// Check metadata for reference IDs
	if source.Metadata != nil && target.Metadata != nil {
		sourceRef := source.Metadata["reference_id"]
		targetRef := target.Metadata["reference_id"]

		if sourceRef != "" && targetRef != "" && sourceRef == targetRef {
			result.Matched = true
			result.Confidence = 0.95
			result.Differences = m.findDifferences(source, target)
			return result
		}
	}

	return result
}

func (m *ReferenceIDMatcher) findDifferences(source, target *models.Transaction) []Difference {
	var diffs []Difference

	if !source.Amount.Equal(target.Amount) {
		diffs = append(diffs, Difference{
			Field:    "amount",
			Source:   source.Amount.String(),
			Target:   target.Amount.String(),
			Severity: "error",
		})
	}

	if source.Status != target.Status {
		diffs = append(diffs, Difference{
			Field:    "status",
			Source:   string(source.Status),
			Target:   string(target.Status),
			Severity: "warning",
		})
	}

	return diffs
}

// CompositeMatch represents a match using multiple matchers
type CompositeMatch struct {
	SourceID   string
	TargetID   string
	Matchers   []string
	Confidence float64
	Differences []Difference
}

// MultiMatcher uses multiple matchers and combines their results
type MultiMatcher struct {
	matchers []Matcher
}

// NewMultiMatcher creates a new multi-matcher
func NewMultiMatcher(matchers ...Matcher) *MultiMatcher {
	return &MultiMatcher{matchers: matchers}
}

func (m *MultiMatcher) Name() string { return "multi" }

func (m *MultiMatcher) Match(source, target *models.Transaction) *MatchResult {
	result := &MatchResult{
		MatchType: "multi",
	}

	var totalConfidence float64
	matcherNames := make([]string, 0)

	for _, matcher := range m.matchers {
		matchResult := matcher.Match(source, target)
		if matchResult.Matched {
			totalConfidence += matchResult.Confidence
			matcherNames = append(matcherNames, matcher.Name())

			// Merge differences
			for _, diff := range matchResult.Differences {
				found := false
				for _, existing := range result.Differences {
					if existing.Field == diff.Field {
						found = true
						break
					}
				}
				if !found {
					result.Differences = append(result.Differences, diff)
				}
			}
		}
	}

	if len(matcherNames) > 0 {
		result.Matched = true
		result.Confidence = totalConfidence / float64(len(matcherNames))
		result.MatchType = strings.Join(matcherNames, "+")
	}

	return result
}
