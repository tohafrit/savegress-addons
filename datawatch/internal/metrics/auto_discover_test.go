package metrics

import (
	"testing"
)

func TestNewAutoDiscovery(t *testing.T) {
	ad := NewAutoDiscovery()
	if ad == nil {
		t.Fatal("expected non-nil AutoDiscovery")
	}
	if len(ad.numericPatterns) == 0 {
		t.Error("expected numeric patterns to be initialized")
	}
	if len(ad.statusPatterns) == 0 {
		t.Error("expected status patterns to be initialized")
	}
	if len(ad.timestampPatterns) == 0 {
		t.Error("expected timestamp patterns to be initialized")
	}
	if len(ad.idPatterns) == 0 {
		t.Error("expected id patterns to be initialized")
	}
	if len(ad.statusValues) == 0 {
		t.Error("expected status values to be initialized")
	}
}

func TestAutoDiscovery_InferFieldType_ByValue(t *testing.T) {
	ad := NewAutoDiscovery()

	tests := []struct {
		name      string
		fieldName string
		value     interface{}
		want      FieldType
	}{
		// Boolean
		{"bool true", "is_active", true, FieldTypeBoolean},
		{"bool false", "enabled", false, FieldTypeBoolean},

		// Numeric
		{"float64", "amount", float64(100.5), FieldTypeNumeric},
		{"float32", "price", float32(50.0), FieldTypeNumeric},
		{"int", "count", int(10), FieldTypeNumeric},
		{"int64", "total", int64(1000), FieldTypeNumeric},
		{"uint", "quantity", uint(5), FieldTypeNumeric},

		// Nil
		{"nil value", "unknown_field", nil, FieldTypeUnknown},

		// Status values
		{"status pending", "status", "pending", FieldTypeStatus},
		{"status active", "state", "active", FieldTypeStatus},
		{"status completed", "order_status", "completed", FieldTypeStatus},

		// ID fields with numeric value
		{"user_id numeric", "user_id", int(12345), FieldTypeID},
		{"id field", "id", int64(1), FieldTypeID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ad.InferFieldType(tt.fieldName, tt.value)
			if got != tt.want {
				t.Errorf("InferFieldType(%s, %v) = %v, want %v", tt.fieldName, tt.value, got, tt.want)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_NumericPatterns(t *testing.T) {
	ad := NewAutoDiscovery()

	numericFields := []string{
		"amount", "price", "total", "sum", "count", "quantity",
		"total_amount", "order_price", "item_count",
		"num_items", "total_cost", "avg_score",
	}

	for _, field := range numericFields {
		t.Run(field, func(t *testing.T) {
			got := ad.InferFieldType(field, float64(100))
			if got != FieldTypeNumeric {
				t.Errorf("InferFieldType(%s) = %v, want %v", field, got, FieldTypeNumeric)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_StatusPatterns(t *testing.T) {
	ad := NewAutoDiscovery()

	statusFields := []string{
		"status", "state", "type", "kind", "category",
		"order_status", "user_state", "item_type",
		"is_active", "has_access", "can_edit",
	}

	for _, field := range statusFields {
		t.Run(field, func(t *testing.T) {
			got := ad.InferFieldType(field, "some_value")
			if got != FieldTypeStatus {
				t.Errorf("InferFieldType(%s) = %v, want %v", field, got, FieldTypeStatus)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_TimestampPatterns(t *testing.T) {
	ad := NewAutoDiscovery()

	timestampFields := []string{
		"created_at", "updated_at", "deleted_at", "timestamp",
		"order_date", "modified_at", "expires_at",
	}

	for _, field := range timestampFields {
		t.Run(field, func(t *testing.T) {
			got := ad.InferFieldType(field, "2024-01-01T00:00:00Z")
			if got != FieldTypeTimestamp {
				t.Errorf("InferFieldType(%s) = %v, want %v", field, got, FieldTypeTimestamp)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_IDPatterns(t *testing.T) {
	ad := NewAutoDiscovery()

	idFields := []string{
		"id", "uuid", "guid",
		"user_id", "customer_id", "order_id",
	}

	for _, field := range idFields {
		t.Run(field, func(t *testing.T) {
			got := ad.InferFieldType(field, "abc-123")
			if got != FieldTypeID {
				t.Errorf("InferFieldType(%s) = %v, want %v", field, got, FieldTypeID)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_KnownStatusValues(t *testing.T) {
	ad := NewAutoDiscovery()

	statusValues := []string{
		"pending", "processing", "completed", "cancelled", "failed",
		"active", "inactive", "suspended", "banned",
		"todo", "in_progress", "done",
		"enabled", "disabled", "open", "closed",
		"success", "error", "warning",
		"high", "medium", "low", "critical",
	}

	for _, value := range statusValues {
		t.Run(value, func(t *testing.T) {
			got := ad.InferFieldType("some_field", value)
			if got != FieldTypeStatus {
				t.Errorf("InferFieldType(some_field, %s) = %v, want %v", value, got, FieldTypeStatus)
			}
		})
	}

	// Test case insensitivity
	t.Run("uppercase PENDING", func(t *testing.T) {
		got := ad.InferFieldType("field", "PENDING")
		if got != FieldTypeStatus {
			t.Errorf("InferFieldType with uppercase PENDING = %v, want %v", got, FieldTypeStatus)
		}
	})
}

func TestAutoDiscovery_InferFieldType_Text(t *testing.T) {
	ad := NewAutoDiscovery()

	// Field names and values that should be inferred as text
	tests := []struct {
		fieldName string
		value     string
	}{
		{"description", "Some description text"},
		{"name", "John Doe"},
		{"email", "test@example.com"},
		{"notes", "Random notes here"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			got := ad.InferFieldType(tt.fieldName, tt.value)
			if got != FieldTypeText {
				t.Errorf("InferFieldType(%s, %s) = %v, want %v", tt.fieldName, tt.value, got, FieldTypeText)
			}
		})
	}
}

func TestAutoDiscovery_InferFieldType_ByFieldNameOnly(t *testing.T) {
	ad := NewAutoDiscovery()

	tests := []struct {
		fieldName string
		want      FieldType
	}{
		{"amount", FieldTypeNumeric},
		{"status", FieldTypeStatus},
		{"created_at", FieldTypeTimestamp},
		{"user_id", FieldTypeID},
		{"random_field", FieldTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			// Use a struct value that doesn't match any value-based type
			got := ad.InferFieldType(tt.fieldName, struct{}{})
			if got != tt.want {
				t.Errorf("InferFieldType(%s, struct) = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}

func TestAutoDiscovery_GenerateMetricsForTable(t *testing.T) {
	ad := NewAutoDiscovery()

	schema := &TableSchema{
		Name:   "orders",
		Schema: "public",
		Columns: []ColumnInfo{
			{Name: "id", DataType: "integer", IsPrimaryKey: true, InferredType: FieldTypeID},
			{Name: "total_amount", DataType: "decimal", InferredType: FieldTypeNumeric},
			{Name: "status", DataType: "varchar", InferredType: FieldTypeStatus},
			{Name: "user_id", DataType: "integer", InferredType: FieldTypeID},
			{Name: "created_at", DataType: "timestamp", InferredType: FieldTypeTimestamp},
		},
	}

	metrics := ad.GenerateMetricsForTable(schema)

	if len(metrics) == 0 {
		t.Fatal("expected metrics to be generated")
	}

	// Check for base event metrics
	expectedMetrics := map[string]bool{
		"orders_events_total":  false,
		"orders_inserts_total": false,
		"orders_updates_total": false,
		"orders_deletes_total": false,
	}

	for _, m := range metrics {
		if _, ok := expectedMetrics[m.Name]; ok {
			expectedMetrics[m.Name] = true
		}
	}

	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("expected metric %s not found", name)
		}
	}

	// Check that numeric field generates sum/avg/p99 metrics
	hasNumericMetrics := false
	for _, m := range metrics {
		if m.Name == "orders_total_amount_sum" || m.Name == "orders_total_amount_avg" || m.Name == "orders_total_amount_p99" {
			hasNumericMetrics = true
			break
		}
	}
	if !hasNumericMetrics {
		t.Error("expected numeric field metrics to be generated")
	}

	// Check that status field generates distribution metric
	hasStatusMetric := false
	for _, m := range metrics {
		if m.Name == "orders_by_status" {
			hasStatusMetric = true
			if len(m.Labels) == 0 || m.Labels[0] != "status" {
				t.Error("expected status metric to have status label")
			}
			break
		}
	}
	if !hasStatusMetric {
		t.Error("expected status distribution metric to be generated")
	}

	// Check that ID field generates cardinality metric
	hasCardinalityMetric := false
	for _, m := range metrics {
		if m.Name == "orders_user_id_cardinality" || m.Name == "orders_id_cardinality" {
			hasCardinalityMetric = true
			break
		}
	}
	if !hasCardinalityMetric {
		t.Error("expected ID cardinality metric to be generated")
	}

	// Verify all metrics are marked as auto-generated
	for _, m := range metrics {
		if !m.AutoGen {
			t.Errorf("metric %s should be marked as auto-generated", m.Name)
		}
	}
}

func TestAutoDiscovery_GenerateMetricsForTable_EmptySchema(t *testing.T) {
	ad := NewAutoDiscovery()

	schema := &TableSchema{
		Name:    "empty_table",
		Schema:  "public",
		Columns: []ColumnInfo{},
	}

	metrics := ad.GenerateMetricsForTable(schema)

	// Should still generate base event metrics
	if len(metrics) != 4 {
		t.Errorf("expected 4 base metrics, got %d", len(metrics))
	}
}

func TestAutoDiscovery_GenerateMetricsForTable_InferredVsExplicit(t *testing.T) {
	ad := NewAutoDiscovery()

	// Column with explicit InferredType should use that
	schema := &TableSchema{
		Name:   "test",
		Schema: "public",
		Columns: []ColumnInfo{
			{Name: "custom_field", DataType: "integer", InferredType: FieldTypeNumeric},
		},
	}

	metrics := ad.GenerateMetricsForTable(schema)

	// Should generate numeric metrics for custom_field
	hasNumericMetric := false
	for _, m := range metrics {
		if m.Field == "custom_field" {
			hasNumericMetric = true
			break
		}
	}
	if !hasNumericMetric {
		t.Error("expected numeric metric for custom_field with explicit InferredType")
	}
}

func TestAutoDiscovery_MatchesPatterns(t *testing.T) {
	ad := NewAutoDiscovery()

	// Test numeric patterns
	if !ad.matchesPatterns("amount", ad.numericPatterns) {
		t.Error("amount should match numeric patterns")
	}
	if !ad.matchesPatterns("total_price", ad.numericPatterns) {
		t.Error("total_price should match numeric patterns")
	}

	// Test status patterns
	if !ad.matchesPatterns("status", ad.statusPatterns) {
		t.Error("status should match status patterns")
	}
	if !ad.matchesPatterns("is_active", ad.statusPatterns) {
		t.Error("is_active should match status patterns")
	}

	// Non-matching
	if ad.matchesPatterns("random_field", ad.numericPatterns) {
		t.Error("random_field should not match numeric patterns")
	}
}
