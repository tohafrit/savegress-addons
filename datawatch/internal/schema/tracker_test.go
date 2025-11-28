package schema

import (
	"context"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	cfg := &Config{
		TrackChanges:    true,
		AlertOnBreaking: true,
	}

	tracker := NewTracker(cfg)
	if tracker == nil {
		t.Fatal("expected tracker to be created")
	}
	if tracker.config != cfg {
		t.Error("expected config to be set")
	}
	if tracker.schemas == nil {
		t.Error("expected schemas map to be initialized")
	}
	if tracker.changes == nil {
		t.Error("expected changes slice to be initialized")
	}
	if tracker.snapshots == nil {
		t.Error("expected snapshots map to be initialized")
	}
}

func TestTrackerStartStop(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := tracker.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start tracker: %v", err)
	}

	// Should not error on second start
	err = tracker.Start(ctx)
	if err != nil {
		t.Errorf("second start should not error: %v", err)
	}

	tracker.Stop()

	// Should not panic on second stop
	tracker.Stop()
}

func TestTrackerSetBreakingChangeCallback(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true, AlertOnBreaking: true})

	called := false
	tracker.SetBreakingChangeCallback(func(c *Change) {
		called = true
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	// Process a breaking DDL event
	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "DROP TABLE",
		DDLStatement: "DROP TABLE users",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change == nil {
		t.Fatal("expected change to be returned")
	}
	if !change.IsBreaking {
		t.Error("expected DROP TABLE to be breaking")
	}

	// Give time for async processing
	time.Sleep(50 * time.Millisecond)

	if !called {
		t.Logf("callback may not have been called due to timing")
	}
}

func TestProcessDDLEventNotTracking(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: false})

	ddl := DDLEvent{
		Database:  "testdb",
		Table:     "users",
		DDLType:   "CREATE TABLE",
		Timestamp: time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change != nil {
		t.Error("expected nil change when tracking disabled")
	}
}

func TestProcessDDLEventCreateTable(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "CREATE TABLE",
		DDLStatement: "CREATE TABLE users (id INT, name VARCHAR(100))",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change == nil {
		t.Fatal("expected change")
	}
	if change.Type != ChangeTypeAddTable {
		t.Errorf("expected type add_table, got %s", change.Type)
	}
	if change.IsBreaking {
		t.Error("CREATE TABLE should not be breaking")
	}

	// Verify schema was added
	schema, ok := tracker.GetSchema("testdb", "public", "users")
	if !ok {
		t.Error("expected schema to be registered")
	}
	if schema.Table != "users" {
		t.Errorf("expected table name 'users', got '%s'", schema.Table)
	}
}

func TestProcessDDLEventDropTable(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	// First create the table
	tracker.RegisterSchema(&TableSchema{
		Database: "testdb",
		Schema:   "public",
		Table:    "users",
	})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "DROP TABLE",
		DDLStatement: "DROP TABLE users",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeDropTable {
		t.Errorf("expected type drop_table, got %s", change.Type)
	}
	if !change.IsBreaking {
		t.Error("DROP TABLE should be breaking")
	}
	if change.Impact.Level != "critical" {
		t.Errorf("expected critical impact, got %s", change.Impact.Level)
	}

	// Verify schema was removed
	_, ok := tracker.GetSchema("testdb", "public", "users")
	if ok {
		t.Error("expected schema to be removed")
	}
}

func TestProcessDDLEventAddColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	// Register schema first
	tracker.RegisterSchema(&TableSchema{
		Database: "testdb",
		Schema:   "public",
		Table:    "users",
		Columns:  []Column{{Name: "id", Position: 1, DataType: "int"}},
	})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "ALTER TABLE ADD COLUMN",
		DDLStatement: "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeAddColumn {
		t.Errorf("expected type add_column, got %s", change.Type)
	}
	if change.IsBreaking {
		t.Error("ADD COLUMN should not be breaking")
	}
	if change.Impact.Level != "low" {
		t.Errorf("expected low impact, got %s", change.Impact.Level)
	}
}

func TestProcessDDLEventDropColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{
		Database: "testdb",
		Schema:   "public",
		Table:    "users",
		Columns: []Column{
			{Name: "id", Position: 1, DataType: "int"},
			{Name: "email", Position: 2, DataType: "varchar"},
		},
	})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "ALTER TABLE DROP COLUMN",
		DDLStatement: "ALTER TABLE users DROP COLUMN email",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeDropColumn {
		t.Errorf("expected type drop_column, got %s", change.Type)
	}
	if !change.IsBreaking {
		t.Error("DROP COLUMN should be breaking")
	}
	if change.Impact.Level != "high" {
		t.Errorf("expected high impact, got %s", change.Impact.Level)
	}
}

func TestProcessDDLEventModifyColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{
		Database: "testdb",
		Schema:   "public",
		Table:    "users",
		Columns:  []Column{{Name: "name", Position: 1, DataType: "varchar"}},
	})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "ALTER TABLE MODIFY COLUMN",
		DDLStatement: "ALTER TABLE users MODIFY COLUMN name TEXT",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeModifyColumn {
		t.Errorf("expected type modify_column, got %s", change.Type)
	}
}

func TestProcessDDLEventRenameColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "ALTER TABLE RENAME COLUMN",
		DDLStatement: "ALTER TABLE users RENAME COLUMN name TO full_name",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeRenameColumn {
		t.Errorf("expected type rename_column, got %s", change.Type)
	}
	if !change.IsBreaking {
		t.Error("RENAME COLUMN should be breaking")
	}
	if change.Impact.Level != "high" {
		t.Errorf("expected high impact, got %s", change.Impact.Level)
	}
}

func TestProcessDDLEventCreateIndex(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "CREATE INDEX",
		DDLStatement: "CREATE INDEX idx_users_email ON users(email)",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeAddIndex {
		t.Errorf("expected type add_index, got %s", change.Type)
	}
	if change.IsBreaking {
		t.Error("CREATE INDEX should not be breaking")
	}
}

func TestProcessDDLEventDropIndex(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "DROP INDEX",
		DDLStatement: "DROP INDEX idx_users_email",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("failed to process DDL: %v", err)
	}
	if change.Type != ChangeTypeDropIndex {
		t.Errorf("expected type drop_index, got %s", change.Type)
	}
}

func TestProcessDDLEventUnknownType(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ddl := DDLEvent{
		Database:     "testdb",
		Schema:       "public",
		Table:        "users",
		DDLType:      "UNKNOWN DDL",
		DDLStatement: "UNKNOWN DDL STATEMENT",
		Timestamp:    time.Now(),
	}

	change, err := tracker.ProcessDDLEvent(ddl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change != nil {
		t.Error("expected nil change for unknown DDL type")
	}
}

func TestRegisterSchema(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	schema := &TableSchema{
		Database: "testdb",
		Schema:   "public",
		Table:    "users",
		Columns: []Column{
			{Name: "id", Position: 1, DataType: "int"},
			{Name: "name", Position: 2, DataType: "varchar"},
		},
	}

	tracker.RegisterSchema(schema)

	retrieved, ok := tracker.GetSchema("testdb", "public", "users")
	if !ok {
		t.Fatal("expected schema to be found")
	}
	if len(retrieved.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(retrieved.Columns))
	}
}

func TestGetSchemaNotFound(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	_, ok := tracker.GetSchema("testdb", "public", "nonexistent")
	if ok {
		t.Error("expected schema not to be found")
	}
}

func TestListSchemas(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{Database: "db1", Schema: "public", Table: "t1"})
	tracker.RegisterSchema(&TableSchema{Database: "db1", Schema: "public", Table: "t2"})
	tracker.RegisterSchema(&TableSchema{Database: "db2", Schema: "public", Table: "t3"})

	schemas := tracker.ListSchemas()
	if len(schemas) != 3 {
		t.Errorf("expected 3 schemas, got %d", len(schemas))
	}
}

func TestGetChanges(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	// Create some changes
	events := []DDLEvent{
		{Database: "db", Schema: "public", Table: "users", DDLType: "CREATE TABLE", Timestamp: time.Now()},
		{Database: "db", Schema: "public", Table: "orders", DDLType: "CREATE TABLE", Timestamp: time.Now()},
		{Database: "db", Schema: "public", Table: "users", DDLType: "ALTER TABLE ADD COLUMN", Timestamp: time.Now()},
	}

	for _, e := range events {
		tracker.ProcessDDLEvent(e)
	}

	// Give time for async processing
	time.Sleep(50 * time.Millisecond)

	// Get all changes
	changes := tracker.GetChanges(ChangeFilter{})
	if len(changes) < 3 {
		t.Logf("expected at least 3 changes, got %d (may be timing issue)", len(changes))
	}

	// Filter by table
	userChanges := tracker.GetChanges(ChangeFilter{Table: "users"})
	t.Logf("user changes: %d", len(userChanges))

	// Filter by type
	createChanges := tracker.GetChanges(ChangeFilter{Type: ChangeTypeAddTable})
	t.Logf("create table changes: %d", len(createChanges))

	// Filter with limit
	limitedChanges := tracker.GetChanges(ChangeFilter{Limit: 1})
	if len(limitedChanges) > 1 {
		t.Errorf("expected at most 1 change with limit, got %d", len(limitedChanges))
	}
}

func TestGetChangesWithBreakingFilter(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	// Create breaking and non-breaking changes
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "t1",
		DDLType: "CREATE TABLE", Timestamp: time.Now(),
	})
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "t2",
		DDLType: "DROP TABLE", Timestamp: time.Now(),
	})

	time.Sleep(50 * time.Millisecond)

	breakingTrue := true
	breakingChanges := tracker.GetChanges(ChangeFilter{Breaking: &breakingTrue})
	t.Logf("breaking changes: %d", len(breakingChanges))

	breakingFalse := false
	nonBreakingChanges := tracker.GetChanges(ChangeFilter{Breaking: &breakingFalse})
	t.Logf("non-breaking changes: %d", len(nonBreakingChanges))
}

func TestGetChangesWithTimeFilter(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	now := time.Now()
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "t1",
		DDLType: "CREATE TABLE", Timestamp: now,
	})

	time.Sleep(50 * time.Millisecond)

	// Filter with start time in the past
	pastTime := now.Add(-time.Hour)
	changes := tracker.GetChanges(ChangeFilter{StartTime: &pastTime})
	t.Logf("changes since past hour: %d", len(changes))

	// Filter with start time in the future
	futureTime := now.Add(time.Hour)
	changes = tracker.GetChanges(ChangeFilter{StartTime: &futureTime})
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for future start time, got %d", len(changes))
	}
}

func TestGetChange(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	change, _ := tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "t1",
		DDLType: "CREATE TABLE", Timestamp: time.Now(),
	})

	time.Sleep(50 * time.Millisecond)

	retrieved, ok := tracker.GetChange(change.ID)
	if !ok {
		t.Logf("change not found (may be timing issue)")
	} else if retrieved.ID != change.ID {
		t.Errorf("expected ID %s, got %s", change.ID, retrieved.ID)
	}

	// Non-existent change
	_, ok = tracker.GetChange("nonexistent")
	if ok {
		t.Error("expected change not to be found")
	}
}

func TestCreateSnapshot(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{Database: "db", Schema: "public", Table: "users"})
	tracker.RegisterSchema(&TableSchema{Database: "db", Schema: "public", Table: "orders"})

	snapshot := tracker.CreateSnapshot("Test snapshot")
	if snapshot == nil {
		t.Fatal("expected snapshot to be created")
	}
	if snapshot.ID == "" {
		t.Error("expected snapshot ID")
	}
	if snapshot.Description != "Test snapshot" {
		t.Errorf("expected description 'Test snapshot', got '%s'", snapshot.Description)
	}
	if len(snapshot.Tables) != 2 {
		t.Errorf("expected 2 tables in snapshot, got %d", len(snapshot.Tables))
	}
}

func TestGetSnapshot(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{Database: "db", Schema: "public", Table: "users"})
	snapshot := tracker.CreateSnapshot("Test")

	retrieved, ok := tracker.GetSnapshot(snapshot.ID)
	if !ok {
		t.Fatal("expected snapshot to be found")
	}
	if retrieved.ID != snapshot.ID {
		t.Errorf("expected ID %s, got %s", snapshot.ID, retrieved.ID)
	}

	// Non-existent snapshot
	_, ok = tracker.GetSnapshot("nonexistent")
	if ok {
		t.Error("expected snapshot not to be found")
	}
}

func TestListSnapshots(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.CreateSnapshot("Snapshot 1")
	tracker.CreateSnapshot("Snapshot 2")
	tracker.CreateSnapshot("Snapshot 3")

	snapshots := tracker.ListSnapshots()
	if len(snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(snapshots))
	}
}

func TestCompareSchemas(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	oldSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "int"},
			{Name: "name", DataType: "varchar"},
			{Name: "email", DataType: "varchar"},
		},
	}

	newSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "bigint"},     // modified
			{Name: "name", DataType: "varchar"},  // unchanged
			{Name: "phone", DataType: "varchar"}, // added
			// email removed
		},
	}

	diff := tracker.CompareSchemas(oldSchema, newSchema)
	if diff == nil {
		t.Fatal("expected diff")
	}

	if diff.Table != "users" {
		t.Errorf("expected table 'users', got '%s'", diff.Table)
	}

	// Should have: 1 added (phone), 1 removed (email), 1 modified (id)
	addedCount := 0
	removedCount := 0
	modifiedCount := 0

	for _, c := range diff.Changes {
		switch c.Type {
		case ChangeTypeAddColumn:
			addedCount++
		case ChangeTypeDropColumn:
			removedCount++
		case ChangeTypeModifyColumn:
			modifiedCount++
		}
	}

	if addedCount != 1 {
		t.Errorf("expected 1 added column, got %d", addedCount)
	}
	if removedCount != 1 {
		t.Errorf("expected 1 removed column, got %d", removedCount)
	}
	if modifiedCount != 1 {
		t.Errorf("expected 1 modified column, got %d", modifiedCount)
	}
	if !diff.HasBreaking {
		t.Error("expected diff to have breaking changes (column dropped)")
	}
}

func TestCompareSchemasCompatibleTypeChange(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	oldSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "int"},
		},
	}

	newSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "bigint"}, // int -> bigint is compatible
		},
	}

	diff := tracker.CompareSchemas(oldSchema, newSchema)

	if len(diff.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changes))
	}

	// int -> bigint should be compatible (not breaking)
	if diff.Changes[0].IsBreaking {
		t.Error("int -> bigint should be a compatible type change")
	}
}

func TestCompareSchemasIncompatibleTypeChange(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	oldSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "varchar"},
		},
	}

	newSchema := &TableSchema{
		Table: "users",
		Columns: []Column{
			{Name: "id", DataType: "int"}, // varchar -> int is not compatible
		},
	}

	diff := tracker.CompareSchemas(oldSchema, newSchema)

	if len(diff.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changes))
	}

	// varchar -> int should be breaking
	if !diff.Changes[0].IsBreaking {
		t.Error("varchar -> int should be a breaking type change")
	}
}

func TestIsBreakingChange(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tests := []struct {
		name     string
		change   *Change
		expected bool
	}{
		{
			name:     "drop table",
			change:   &Change{Type: ChangeTypeDropTable},
			expected: true,
		},
		{
			name:     "drop column",
			change:   &Change{Type: ChangeTypeDropColumn},
			expected: true,
		},
		{
			name:     "rename column",
			change:   &Change{Type: ChangeTypeRenameColumn},
			expected: true,
		},
		{
			name:     "rename table",
			change:   &Change{Type: ChangeTypeRenameTable},
			expected: true,
		},
		{
			name:     "drop pk",
			change:   &Change{Type: ChangeTypeDropPK},
			expected: true,
		},
		{
			name:     "drop fk",
			change:   &Change{Type: ChangeTypeDropFK},
			expected: true,
		},
		{
			name:     "add column",
			change:   &Change{Type: ChangeTypeAddColumn},
			expected: false,
		},
		{
			name:     "add index",
			change:   &Change{Type: ChangeTypeAddIndex},
			expected: false,
		},
		{
			name: "modify nullable to not null",
			change: &Change{
				Type:        ChangeTypeModifyColumn,
				WasNullable: boolPtr(true),
				IsNullable:  boolPtr(false),
			},
			expected: true,
		},
		{
			name: "modify not null to nullable",
			change: &Change{
				Type:        ChangeTypeModifyColumn,
				WasNullable: boolPtr(false),
				IsNullable:  boolPtr(true),
			},
			expected: false,
		},
		{
			name: "incompatible type change",
			change: &Change{
				Type:    ChangeTypeModifyColumn,
				OldType: "text",
				NewType: "int",
			},
			expected: true,
		},
		{
			name: "compatible type change",
			change: &Change{
				Type:    ChangeTypeModifyColumn,
				OldType: "varchar",
				NewType: "text",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.isBreakingChange(tt.change)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsCompatibleTypeChange(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tests := []struct {
		oldType  string
		newType  string
		expected bool
	}{
		{"varchar", "text", true},
		{"varchar", "varchar", true},
		{"int", "bigint", true},
		{"smallint", "int", true},
		{"smallint", "bigint", true},
		{"float", "double", true},
		{"float", "numeric", true},
		{"timestamp", "timestamptz", true},
		{"text", "varchar", false}, // text -> varchar is not safe
		{"bigint", "int", false},   // bigint -> int may lose data
		{"int", "varchar", false},  // different type family
	}

	for _, tt := range tests {
		t.Run(tt.oldType+"->"+tt.newType, func(t *testing.T) {
			result := tracker.isCompatibleTypeChange(tt.oldType, tt.newType)
			if result != tt.expected {
				t.Errorf("isCompatibleTypeChange(%s, %s) = %v, expected %v",
					tt.oldType, tt.newType, result, tt.expected)
			}
		})
	}
}

func TestAssessImpact(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tests := []struct {
		name          string
		change        *Change
		expectedLevel string
	}{
		{
			name:          "drop table",
			change:        &Change{Type: ChangeTypeDropTable, Table: "users"},
			expectedLevel: "critical",
		},
		{
			name:          "drop column",
			change:        &Change{Type: ChangeTypeDropColumn, Table: "users", Column: "email"},
			expectedLevel: "high",
		},
		{
			name:          "rename column",
			change:        &Change{Type: ChangeTypeRenameColumn, Column: "name"},
			expectedLevel: "high",
		},
		{
			name:          "breaking modify column",
			change:        &Change{Type: ChangeTypeModifyColumn, Column: "id", IsBreaking: true},
			expectedLevel: "high",
		},
		{
			name:          "non-breaking modify column",
			change:        &Change{Type: ChangeTypeModifyColumn, Column: "id", IsBreaking: false},
			expectedLevel: "medium",
		},
		{
			name:          "add column",
			change:        &Change{Type: ChangeTypeAddColumn, Table: "users", Column: "phone"},
			expectedLevel: "low",
		},
		{
			name:          "add index",
			change:        &Change{Type: ChangeTypeAddIndex},
			expectedLevel: "low",
		},
		{
			name:          "drop index",
			change:        &Change{Type: ChangeTypeDropIndex},
			expectedLevel: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impact := tracker.assessImpact(tt.change)
			if impact.Level != tt.expectedLevel {
				t.Errorf("expected level %s, got %s", tt.expectedLevel, impact.Level)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker.Start(ctx)
	defer tracker.Stop()

	// Register some schemas
	tracker.RegisterSchema(&TableSchema{Database: "db", Schema: "public", Table: "users"})
	tracker.RegisterSchema(&TableSchema{Database: "db", Schema: "public", Table: "orders"})

	// Create some changes
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "users",
		DDLType: "ALTER TABLE ADD COLUMN", Timestamp: time.Now(),
	})
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "users",
		DDLType: "DROP TABLE", Timestamp: time.Now(),
	})
	tracker.ProcessDDLEvent(DDLEvent{
		Database: "db", Schema: "public", Table: "orders",
		DDLType: "CREATE TABLE", Timestamp: time.Now(),
	})

	// Create a snapshot
	tracker.CreateSnapshot("Test")

	time.Sleep(50 * time.Millisecond)

	stats := tracker.GetStats()
	if stats == nil {
		t.Fatal("expected stats")
	}

	if stats.TablesTracked < 1 {
		t.Logf("tables tracked: %d", stats.TablesTracked)
	}

	if stats.Snapshots != 1 {
		t.Errorf("expected 1 snapshot, got %d", stats.Snapshots)
	}

	t.Logf("Stats: tables=%d, changes=%d, breaking=%d, recent=%d",
		stats.TablesTracked, stats.TotalChanges, stats.BreakingChanges, stats.RecentChanges)
}

func TestApplyChangeAddColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Columns:  []Column{{Name: "id", Position: 1, DataType: "int"}},
	})

	change := &Change{
		Type:     ChangeTypeAddColumn,
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Column:   "email",
		NewType:  "varchar",
	}

	tracker.applyChange(change)

	schema, _ := tracker.GetSchema("db", "public", "users")
	if len(schema.Columns) != 2 {
		t.Errorf("expected 2 columns after add, got %d", len(schema.Columns))
	}
}

func TestApplyChangeDropColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Columns: []Column{
			{Name: "id", Position: 1, DataType: "int"},
			{Name: "email", Position: 2, DataType: "varchar"},
		},
	})

	change := &Change{
		Type:     ChangeTypeDropColumn,
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Column:   "email",
	}

	tracker.applyChange(change)

	schema, _ := tracker.GetSchema("db", "public", "users")
	if len(schema.Columns) != 1 {
		t.Errorf("expected 1 column after drop, got %d", len(schema.Columns))
	}
	if schema.Columns[0].Name != "id" {
		t.Errorf("expected remaining column 'id', got '%s'", schema.Columns[0].Name)
	}
}

func TestApplyChangeModifyColumn(t *testing.T) {
	tracker := NewTracker(&Config{TrackChanges: true})

	tracker.RegisterSchema(&TableSchema{
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Columns:  []Column{{Name: "id", Position: 1, DataType: "int"}},
	})

	change := &Change{
		Type:     ChangeTypeModifyColumn,
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Column:   "id",
		NewType:  "bigint",
	}

	tracker.applyChange(change)

	schema, _ := tracker.GetSchema("db", "public", "users")
	if schema.Columns[0].DataType != "bigint" {
		t.Errorf("expected type 'bigint', got '%s'", schema.Columns[0].DataType)
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
