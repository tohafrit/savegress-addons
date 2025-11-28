package schema

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tracker tracks schema changes
type Tracker struct {
	config     *Config
	schemas    map[string]*TableSchema // table -> schema
	changes    []*Change
	snapshots  map[string]*SchemaSnapshot
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	changeCh   chan *Change
	onBreaking func(*Change)
}

// NewTracker creates a new schema tracker
func NewTracker(cfg *Config) *Tracker {
	return &Tracker{
		config:    cfg,
		schemas:   make(map[string]*TableSchema),
		changes:   make([]*Change, 0),
		snapshots: make(map[string]*SchemaSnapshot),
		stopCh:    make(chan struct{}),
		changeCh:  make(chan *Change, 100),
	}
}

// Start starts the schema tracker
func (t *Tracker) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.mu.Unlock()

	go t.processChanges(ctx)

	return nil
}

// Stop stops the schema tracker
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		close(t.stopCh)
		t.running = false
	}
}

// SetBreakingChangeCallback sets a callback for breaking changes
func (t *Tracker) SetBreakingChangeCallback(fn func(*Change)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onBreaking = fn
}

func (t *Tracker) processChanges(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		case change := <-t.changeCh:
			t.mu.Lock()
			t.changes = append(t.changes, change)
			t.mu.Unlock()

			if change.IsBreaking && t.config.AlertOnBreaking {
				t.mu.RLock()
				cb := t.onBreaking
				t.mu.RUnlock()
				if cb != nil {
					cb(change)
				}
			}
		}
	}
}

// ProcessDDLEvent processes a DDL event from CDC
func (t *Tracker) ProcessDDLEvent(ddl DDLEvent) (*Change, error) {
	if !t.config.TrackChanges {
		return nil, nil
	}

	change := t.parseDDL(ddl)
	if change == nil {
		return nil, nil
	}

	// Determine if breaking
	change.IsBreaking = t.isBreakingChange(change)
	change.Impact = t.assessImpact(change)

	// Update internal schema
	t.applyChange(change)

	// Record change
	select {
	case t.changeCh <- change:
	default:
		// Channel full
	}

	return change, nil
}

// DDLEvent represents a DDL event from CDC
type DDLEvent struct {
	Database     string    `json:"database"`
	Schema       string    `json:"schema"`
	Table        string    `json:"table"`
	DDLType      string    `json:"ddl_type"`
	DDLStatement string    `json:"ddl_statement"`
	Timestamp    time.Time `json:"timestamp"`
}

func (t *Tracker) parseDDL(ddl DDLEvent) *Change {
	change := &Change{
		ID:           fmt.Sprintf("sch_%d", time.Now().UnixNano()),
		Database:     ddl.Database,
		Schema:       ddl.Schema,
		Table:        ddl.Table,
		DDLStatement: ddl.DDLStatement,
		DetectedAt:   ddl.Timestamp,
	}

	// Parse DDL type
	switch ddl.DDLType {
	case "CREATE TABLE":
		change.Type = ChangeTypeAddTable
	case "DROP TABLE":
		change.Type = ChangeTypeDropTable
		change.IsBreaking = true
	case "ALTER TABLE ADD COLUMN":
		change.Type = ChangeTypeAddColumn
	case "ALTER TABLE DROP COLUMN":
		change.Type = ChangeTypeDropColumn
		change.IsBreaking = true
	case "ALTER TABLE MODIFY COLUMN", "ALTER TABLE ALTER COLUMN":
		change.Type = ChangeTypeModifyColumn
	case "ALTER TABLE RENAME COLUMN":
		change.Type = ChangeTypeRenameColumn
		change.IsBreaking = true
	case "CREATE INDEX":
		change.Type = ChangeTypeAddIndex
	case "DROP INDEX":
		change.Type = ChangeTypeDropIndex
	default:
		return nil
	}

	return change
}

func (t *Tracker) isBreakingChange(change *Change) bool {
	switch change.Type {
	case ChangeTypeDropTable, ChangeTypeDropColumn, ChangeTypeRenameColumn, ChangeTypeRenameTable:
		return true
	case ChangeTypeModifyColumn:
		// Check if type change is breaking
		if change.OldType != "" && change.NewType != "" && !t.isCompatibleTypeChange(change.OldType, change.NewType) {
			return true
		}
		// Changing nullable to not nullable is breaking
		if change.WasNullable != nil && change.IsNullable != nil && *change.WasNullable && !*change.IsNullable {
			return true
		}
	case ChangeTypeDropPK, ChangeTypeDropFK:
		return true
	}
	return false
}

func (t *Tracker) isCompatibleTypeChange(oldType, newType string) bool {
	// Simplified compatibility check
	// In reality, this would be more sophisticated
	compatiblePairs := map[string][]string{
		"varchar":   {"text", "varchar"},
		"text":      {"text"},
		"int":       {"bigint", "int"},
		"smallint":  {"int", "bigint"},
		"bigint":    {"bigint"},
		"float":     {"double", "numeric"},
		"double":    {"double", "numeric"},
		"timestamp": {"timestamptz"},
	}

	if compatible, ok := compatiblePairs[oldType]; ok {
		for _, t := range compatible {
			if t == newType {
				return true
			}
		}
	}
	return oldType == newType
}

func (t *Tracker) assessImpact(change *Change) Impact {
	impact := Impact{
		Level: "low",
	}

	switch change.Type {
	case ChangeTypeDropTable:
		impact.Level = "critical"
		impact.Description = fmt.Sprintf("Table '%s' is being dropped. All data will be lost.", change.Table)
		impact.Warnings = []string{
			"All downstream consumers will be affected",
			"Data in this table will be permanently deleted",
			"Dependent views and queries will fail",
		}

	case ChangeTypeDropColumn:
		impact.Level = "high"
		impact.Description = fmt.Sprintf("Column '%s' is being dropped from table '%s'.", change.Column, change.Table)
		impact.Warnings = []string{
			"Applications reading this column will fail",
			"ETL pipelines may need updates",
			"Data in this column will be lost",
		}

	case ChangeTypeModifyColumn:
		if change.IsBreaking {
			impact.Level = "high"
			impact.Description = fmt.Sprintf("Column '%s' type is changing in a potentially incompatible way.", change.Column)
		} else {
			impact.Level = "medium"
			impact.Description = fmt.Sprintf("Column '%s' is being modified.", change.Column)
		}

	case ChangeTypeRenameColumn, ChangeTypeRenameTable:
		impact.Level = "high"
		impact.Description = "Renaming will break existing queries and applications."
		impact.Warnings = []string{
			"All queries using the old name will fail",
			"Update application code and configurations",
		}

	case ChangeTypeAddColumn:
		impact.Level = "low"
		impact.Description = fmt.Sprintf("New column '%s' added to table '%s'.", change.Column, change.Table)

	case ChangeTypeAddIndex, ChangeTypeDropIndex:
		impact.Level = "low"
		impact.Description = "Index change may affect query performance."
	}

	return impact
}

func (t *Tracker) applyChange(change *Change) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s.%s.%s", change.Database, change.Schema, change.Table)

	switch change.Type {
	case ChangeTypeAddTable:
		t.schemas[key] = &TableSchema{
			Database:  change.Database,
			Schema:    change.Schema,
			Table:     change.Table,
			Columns:   []Column{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

	case ChangeTypeDropTable:
		delete(t.schemas, key)

	case ChangeTypeAddColumn:
		if schema, ok := t.schemas[key]; ok {
			schema.Columns = append(schema.Columns, Column{
				Name:     change.Column,
				Position: len(schema.Columns) + 1,
				DataType: change.NewType,
			})
			schema.UpdatedAt = time.Now()
		}

	case ChangeTypeDropColumn:
		if schema, ok := t.schemas[key]; ok {
			newCols := make([]Column, 0, len(schema.Columns)-1)
			for _, col := range schema.Columns {
				if col.Name != change.Column {
					newCols = append(newCols, col)
				}
			}
			schema.Columns = newCols
			schema.UpdatedAt = time.Now()
		}

	case ChangeTypeModifyColumn:
		if schema, ok := t.schemas[key]; ok {
			for i, col := range schema.Columns {
				if col.Name == change.Column {
					schema.Columns[i].DataType = change.NewType
					break
				}
			}
			schema.UpdatedAt = time.Now()
		}
	}
}

// RegisterSchema registers a table schema
func (t *Tracker) RegisterSchema(schema *TableSchema) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s.%s.%s", schema.Database, schema.Schema, schema.Table)
	t.schemas[key] = schema
}

// GetSchema returns the current schema for a table
func (t *Tracker) GetSchema(database, schema, table string) (*TableSchema, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := fmt.Sprintf("%s.%s.%s", database, schema, table)
	s, ok := t.schemas[key]
	return s, ok
}

// ListSchemas returns all tracked schemas
func (t *Tracker) ListSchemas() []*TableSchema {
	t.mu.RLock()
	defer t.mu.RUnlock()

	schemas := make([]*TableSchema, 0, len(t.schemas))
	for _, s := range t.schemas {
		schemas = append(schemas, s)
	}
	return schemas
}

// GetChanges returns schema changes with filters
func (t *Tracker) GetChanges(filter ChangeFilter) []*Change {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []*Change
	count := 0
	for i := len(t.changes) - 1; i >= 0 && (filter.Limit == 0 || count < filter.Limit); i-- {
		c := t.changes[i]
		if t.matchesFilter(c, filter) {
			results = append(results, c)
			count++
		}
	}
	return results
}

// ChangeFilter defines filters for schema changes
type ChangeFilter struct {
	Table      string
	Type       ChangeType
	Breaking   *bool
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
}

func (t *Tracker) matchesFilter(c *Change, filter ChangeFilter) bool {
	if filter.Table != "" && c.Table != filter.Table {
		return false
	}
	if filter.Type != "" && c.Type != filter.Type {
		return false
	}
	if filter.Breaking != nil && c.IsBreaking != *filter.Breaking {
		return false
	}
	if filter.StartTime != nil && c.DetectedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && c.DetectedAt.After(*filter.EndTime) {
		return false
	}
	return true
}

// GetChange returns a change by ID
func (t *Tracker) GetChange(id string) (*Change, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, c := range t.changes {
		if c.ID == id {
			return c, true
		}
	}
	return nil, false
}

// CreateSnapshot creates a schema snapshot
func (t *Tracker) CreateSnapshot(description string) *SchemaSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	snapshot := &SchemaSnapshot{
		ID:          fmt.Sprintf("snap_%d", time.Now().UnixNano()),
		Tables:      make(map[string]*TableSchema),
		Description: description,
		CreatedAt:   time.Now(),
	}

	// Copy current schemas
	for key, schema := range t.schemas {
		snapshot.Tables[key] = schema
		if snapshot.Database == "" {
			snapshot.Database = schema.Database
		}
	}

	t.snapshots[snapshot.ID] = snapshot
	return snapshot
}

// GetSnapshot returns a snapshot by ID
func (t *Tracker) GetSnapshot(id string) (*SchemaSnapshot, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	s, ok := t.snapshots[id]
	return s, ok
}

// ListSnapshots returns all snapshots
func (t *Tracker) ListSnapshots() []*SchemaSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snapshots := make([]*SchemaSnapshot, 0, len(t.snapshots))
	for _, s := range t.snapshots {
		snapshots = append(snapshots, s)
	}
	return snapshots
}

// CompareSchemas compares two table schemas
func (t *Tracker) CompareSchemas(old, new *TableSchema) *SchemaDiff {
	diff := &SchemaDiff{
		Table:     new.Table,
		OldSchema: old,
		NewSchema: new,
		Changes:   make([]*Change, 0),
		DiffedAt:  time.Now(),
	}

	// Create maps for easy comparison
	oldCols := make(map[string]*Column)
	for i := range old.Columns {
		oldCols[old.Columns[i].Name] = &old.Columns[i]
	}

	newCols := make(map[string]*Column)
	for i := range new.Columns {
		newCols[new.Columns[i].Name] = &new.Columns[i]
	}

	// Find added columns
	for name, col := range newCols {
		if _, exists := oldCols[name]; !exists {
			change := &Change{
				ID:         fmt.Sprintf("diff_%d", time.Now().UnixNano()),
				Type:       ChangeTypeAddColumn,
				Table:      new.Table,
				Column:     name,
				NewType:    col.DataType,
				DetectedAt: time.Now(),
			}
			diff.Changes = append(diff.Changes, change)
		}
	}

	// Find removed columns
	for name := range oldCols {
		if _, exists := newCols[name]; !exists {
			change := &Change{
				ID:         fmt.Sprintf("diff_%d", time.Now().UnixNano()),
				Type:       ChangeTypeDropColumn,
				Table:      new.Table,
				Column:     name,
				IsBreaking: true,
				DetectedAt: time.Now(),
			}
			diff.Changes = append(diff.Changes, change)
			diff.HasBreaking = true
		}
	}

	// Find modified columns
	for name, newCol := range newCols {
		if oldCol, exists := oldCols[name]; exists {
			if oldCol.DataType != newCol.DataType {
				change := &Change{
					ID:         fmt.Sprintf("diff_%d", time.Now().UnixNano()),
					Type:       ChangeTypeModifyColumn,
					Table:      new.Table,
					Column:     name,
					OldType:    oldCol.DataType,
					NewType:    newCol.DataType,
					DetectedAt: time.Now(),
				}
				change.IsBreaking = !t.isCompatibleTypeChange(oldCol.DataType, newCol.DataType)
				if change.IsBreaking {
					diff.HasBreaking = true
				}
				diff.Changes = append(diff.Changes, change)
			}
		}
	}

	return diff
}

// GetStats returns tracker statistics
func (t *Tracker) GetStats() *TrackerStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &TrackerStats{
		TablesTracked:   len(t.schemas),
		TotalChanges:    len(t.changes),
		Snapshots:       len(t.snapshots),
		ChangesByType:   make(map[string]int),
		ChangesByTable:  make(map[string]int),
	}

	for _, c := range t.changes {
		stats.ChangesByType[string(c.Type)]++
		stats.ChangesByTable[c.Table]++
		if c.IsBreaking {
			stats.BreakingChanges++
		}
	}

	// Recent changes (last 24h)
	yesterday := time.Now().Add(-24 * time.Hour)
	for _, c := range t.changes {
		if c.DetectedAt.After(yesterday) {
			stats.RecentChanges++
		}
	}

	return stats
}

// TrackerStats contains tracker statistics
type TrackerStats struct {
	TablesTracked   int            `json:"tables_tracked"`
	TotalChanges    int            `json:"total_changes"`
	BreakingChanges int            `json:"breaking_changes"`
	RecentChanges   int            `json:"recent_changes_24h"`
	Snapshots       int            `json:"snapshots"`
	ChangesByType   map[string]int `json:"changes_by_type"`
	ChangesByTable  map[string]int `json:"changes_by_table"`
}
