package schema

import (
	"time"
)

// ChangeType defines the type of schema change
type ChangeType string

const (
	ChangeTypeAddColumn    ChangeType = "add_column"
	ChangeTypeDropColumn   ChangeType = "drop_column"
	ChangeTypeModifyColumn ChangeType = "modify_column"
	ChangeTypeRenameColumn ChangeType = "rename_column"
	ChangeTypeAddIndex     ChangeType = "add_index"
	ChangeTypeDropIndex    ChangeType = "drop_index"
	ChangeTypeAddTable     ChangeType = "add_table"
	ChangeTypeDropTable    ChangeType = "drop_table"
	ChangeTypeRenameTable  ChangeType = "rename_table"
	ChangeTypeAddFK        ChangeType = "add_foreign_key"
	ChangeTypeDropFK       ChangeType = "drop_foreign_key"
	ChangeTypeAddPK        ChangeType = "add_primary_key"
	ChangeTypeDropPK       ChangeType = "drop_primary_key"
)

// Change represents a schema change
type Change struct {
	ID          string                 `json:"id"`
	Type        ChangeType             `json:"type"`
	Database    string                 `json:"database"`
	Schema      string                 `json:"schema"`
	Table       string                 `json:"table"`
	Column      string                 `json:"column,omitempty"`
	OldName     string                 `json:"old_name,omitempty"`
	NewName     string                 `json:"new_name,omitempty"`
	OldType     string                 `json:"old_type,omitempty"`
	NewType     string                 `json:"new_type,omitempty"`
	OldDefault  interface{}            `json:"old_default,omitempty"`
	NewDefault  interface{}            `json:"new_default,omitempty"`
	IsNullable  *bool                  `json:"is_nullable,omitempty"`
	WasNullable *bool                  `json:"was_nullable,omitempty"`
	IsBreaking  bool                   `json:"is_breaking"`
	Impact      Impact                 `json:"impact"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	AppliedAt   *time.Time             `json:"applied_at,omitempty"`
	DDLStatement string                `json:"ddl_statement,omitempty"`
}

// Impact describes the impact of a schema change
type Impact struct {
	Level       string   `json:"level"` // low, medium, high, critical
	Description string   `json:"description"`
	Affected    []string `json:"affected"` // affected downstream systems/tables
	Warnings    []string `json:"warnings"`
}

// TableSchema represents the schema of a table
type TableSchema struct {
	Database    string                 `json:"database"`
	Schema      string                 `json:"schema"`
	Table       string                 `json:"table"`
	Columns     []Column               `json:"columns"`
	PrimaryKey  []string               `json:"primary_key"`
	Indexes     []Index                `json:"indexes"`
	ForeignKeys []ForeignKey           `json:"foreign_keys"`
	RowCount    int64                  `json:"row_count,omitempty"`
	SizeBytes   int64                  `json:"size_bytes,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Column represents a table column
type Column struct {
	Name         string      `json:"name"`
	Position     int         `json:"position"`
	DataType     string      `json:"data_type"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty"`
	NumPrecision *int        `json:"numeric_precision,omitempty"`
	NumScale     *int        `json:"numeric_scale,omitempty"`
	Comment      string      `json:"comment,omitempty"`
}

// Index represents a table index
type Index struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
	Type      string   `json:"type,omitempty"` // btree, hash, gin, etc.
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name            string   `json:"name"`
	Columns         []string `json:"columns"`
	RefTable        string   `json:"referenced_table"`
	RefColumns      []string `json:"referenced_columns"`
	OnDelete        string   `json:"on_delete,omitempty"`
	OnUpdate        string   `json:"on_update,omitempty"`
}

// SchemaDiff represents the difference between two schemas
type SchemaDiff struct {
	Table       string    `json:"table"`
	OldSchema   *TableSchema `json:"old_schema,omitempty"`
	NewSchema   *TableSchema `json:"new_schema,omitempty"`
	Changes     []*Change `json:"changes"`
	HasBreaking bool      `json:"has_breaking"`
	DiffedAt    time.Time `json:"diffed_at"`
}

// SchemaSnapshot represents a point-in-time snapshot of the schema
type SchemaSnapshot struct {
	ID          string                  `json:"id"`
	Database    string                  `json:"database"`
	Tables      map[string]*TableSchema `json:"tables"`
	Version     string                  `json:"version,omitempty"`
	Description string                  `json:"description,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
}

// Config contains schema tracking configuration
type Config struct {
	TrackChanges    bool `json:"track_changes"`
	AlertOnBreaking bool `json:"alert_on_breaking"`
}
