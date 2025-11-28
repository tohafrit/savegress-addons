package metrics

import (
	"regexp"
	"strings"
)

// AutoDiscovery handles automatic field type inference and metric generation
type AutoDiscovery struct {
	// Patterns for field type inference
	numericPatterns  []*regexp.Regexp
	statusPatterns   []*regexp.Regexp
	timestampPatterns []*regexp.Regexp
	idPatterns       []*regexp.Regexp

	// Known status values
	statusValues map[string]bool
}

// NewAutoDiscovery creates a new auto-discovery instance
func NewAutoDiscovery() *AutoDiscovery {
	ad := &AutoDiscovery{
		statusValues: make(map[string]bool),
	}
	ad.initPatterns()
	ad.initStatusValues()
	return ad
}

func (ad *AutoDiscovery) initPatterns() {
	// Numeric field patterns
	ad.numericPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(amount|price|total|sum|count|quantity|qty|cost|fee|tax|discount|balance|rate|score|value|size|weight|height|width|length|duration|age|rating)$`),
		regexp.MustCompile(`(?i)_(amount|price|total|sum|count|quantity|qty|cost|fee|tax|discount|balance|rate|score|value|size|num|number)$`),
		regexp.MustCompile(`(?i)^(num_|total_|sum_|count_|avg_|min_|max_)`),
	}

	// Status/enum field patterns
	ad.statusPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(status|state|type|kind|category|level|tier|role|phase|stage|mode|priority)$`),
		regexp.MustCompile(`(?i)_(status|state|type|kind|category|level|tier|role)$`),
		regexp.MustCompile(`(?i)^(is_|has_|can_|should_|was_|will_)`),
	}

	// Timestamp field patterns
	ad.timestampPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(created_at|updated_at|deleted_at|timestamp|date|time|datetime|modified_at|expires_at|started_at|ended_at|completed_at|published_at|scheduled_at)$`),
		regexp.MustCompile(`(?i)_(at|date|time|timestamp)$`),
	}

	// ID field patterns
	ad.idPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(id|uuid|guid)$`),
		regexp.MustCompile(`(?i)_id$`),
		regexp.MustCompile(`(?i)^(user_id|customer_id|order_id|product_id|account_id|session_id|transaction_id|request_id)$`),
	}
}

func (ad *AutoDiscovery) initStatusValues() {
	// Common status values
	statusValues := []string{
		// Order/Transaction statuses
		"pending", "processing", "completed", "cancelled", "failed", "refunded",
		"paid", "unpaid", "shipped", "delivered", "returned",
		// User statuses
		"active", "inactive", "suspended", "banned", "deleted", "archived",
		// Task statuses
		"todo", "in_progress", "done", "blocked", "review",
		// Generic
		"enabled", "disabled", "open", "closed", "draft", "published",
		"approved", "rejected", "pending_approval",
		"success", "error", "warning", "info",
		"high", "medium", "low", "critical",
		"new", "old", "expired", "valid", "invalid",
	}
	for _, v := range statusValues {
		ad.statusValues[strings.ToLower(v)] = true
	}
}

// InferFieldType infers the type of a field based on its name and value
func (ad *AutoDiscovery) InferFieldType(fieldName string, value interface{}) FieldType {
	// First, check by value type
	switch v := value.(type) {
	case bool:
		return FieldTypeBoolean
	case float64, float32, int, int32, int64, uint, uint32, uint64:
		// Could be numeric or ID - check field name
		if ad.matchesPatterns(fieldName, ad.idPatterns) {
			return FieldTypeID
		}
		if ad.matchesPatterns(fieldName, ad.numericPatterns) {
			return FieldTypeNumeric
		}
		// Default to numeric for numbers
		return FieldTypeNumeric
	case string:
		// Check if it's a known status value
		if ad.statusValues[strings.ToLower(v)] {
			return FieldTypeStatus
		}
		// Check by field name patterns
		if ad.matchesPatterns(fieldName, ad.statusPatterns) {
			return FieldTypeStatus
		}
		if ad.matchesPatterns(fieldName, ad.timestampPatterns) {
			return FieldTypeTimestamp
		}
		if ad.matchesPatterns(fieldName, ad.idPatterns) {
			return FieldTypeID
		}
		return FieldTypeText
	case nil:
		return FieldTypeUnknown
	}

	// Check by field name only
	if ad.matchesPatterns(fieldName, ad.numericPatterns) {
		return FieldTypeNumeric
	}
	if ad.matchesPatterns(fieldName, ad.statusPatterns) {
		return FieldTypeStatus
	}
	if ad.matchesPatterns(fieldName, ad.timestampPatterns) {
		return FieldTypeTimestamp
	}
	if ad.matchesPatterns(fieldName, ad.idPatterns) {
		return FieldTypeID
	}

	return FieldTypeUnknown
}

func (ad *AutoDiscovery) matchesPatterns(fieldName string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(fieldName) {
			return true
		}
	}
	return false
}

// GenerateMetricsForTable generates auto-metrics for a table schema
func (ad *AutoDiscovery) GenerateMetricsForTable(schema *TableSchema) []*Metric {
	var metrics []*Metric

	tableName := schema.Name

	// Event count metrics (always generated)
	metrics = append(metrics,
		&Metric{
			Name:        tableName + "_events_total",
			Type:        MetricTypeCounter,
			Description: "Total CDC events for " + tableName,
			Table:       tableName,
			AutoGen:     true,
		},
		&Metric{
			Name:        tableName + "_inserts_total",
			Type:        MetricTypeCounter,
			Description: "Total INSERT events for " + tableName,
			Table:       tableName,
			AutoGen:     true,
		},
		&Metric{
			Name:        tableName + "_updates_total",
			Type:        MetricTypeCounter,
			Description: "Total UPDATE events for " + tableName,
			Table:       tableName,
			AutoGen:     true,
		},
		&Metric{
			Name:        tableName + "_deletes_total",
			Type:        MetricTypeCounter,
			Description: "Total DELETE events for " + tableName,
			Table:       tableName,
			AutoGen:     true,
		},
	)

	// Field-based metrics
	for _, col := range schema.Columns {
		fieldType := ad.InferFieldType(col.Name, nil)
		if col.InferredType != "" {
			fieldType = col.InferredType
		}

		switch fieldType {
		case FieldTypeNumeric:
			// Generate sum, avg, min, max, p99 metrics
			metrics = append(metrics,
				&Metric{
					Name:        tableName + "_" + col.Name + "_sum",
					Type:        MetricTypeGauge,
					Description: "Sum of " + col.Name + " for " + tableName,
					Table:       tableName,
					Field:       col.Name,
					AutoGen:     true,
				},
				&Metric{
					Name:        tableName + "_" + col.Name + "_avg",
					Type:        MetricTypeGauge,
					Description: "Average of " + col.Name + " for " + tableName,
					Table:       tableName,
					Field:       col.Name,
					AutoGen:     true,
				},
				&Metric{
					Name:        tableName + "_" + col.Name + "_p99",
					Type:        MetricTypeHistogram,
					Description: "P99 of " + col.Name + " for " + tableName,
					Table:       tableName,
					Field:       col.Name,
					AutoGen:     true,
				},
			)

		case FieldTypeStatus:
			// Generate distribution metric
			metrics = append(metrics,
				&Metric{
					Name:        tableName + "_by_" + col.Name,
					Type:        MetricTypeCounter,
					Description: "Count by " + col.Name + " for " + tableName,
					Table:       tableName,
					Field:       col.Name,
					Labels:      []string{col.Name},
					AutoGen:     true,
				},
			)

		case FieldTypeID:
			// Generate cardinality metric
			metrics = append(metrics,
				&Metric{
					Name:        tableName + "_" + col.Name + "_cardinality",
					Type:        MetricTypeGauge,
					Description: "Unique values of " + col.Name + " for " + tableName,
					Table:       tableName,
					Field:       col.Name,
					AutoGen:     true,
				},
			)
		}
	}

	return metrics
}
