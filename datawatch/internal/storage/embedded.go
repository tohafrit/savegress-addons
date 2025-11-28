package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// EmbeddedStorage is a SQLite-based embedded storage for metrics
type EmbeddedStorage struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex

	// In-memory buffer for recent writes
	buffer   []bufferedPoint
	bufferMu sync.Mutex
}

type bufferedPoint struct {
	metric    string
	value     float64
	labels    map[string]string
	timestamp time.Time
}

// NewEmbeddedStorage creates a new embedded storage
func NewEmbeddedStorage(dataPath string) (*EmbeddedStorage, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataPath, "metrics.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &EmbeddedStorage{
		db:     db,
		dbPath: dbPath,
		buffer: make([]bufferedPoint, 0, 1000),
	}

	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Start background flusher
	go s.backgroundFlusher()

	return s, nil
}

func (s *EmbeddedStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		value REAL NOT NULL,
		labels TEXT,
		created_at INTEGER DEFAULT (strftime('%s', 'now'))
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_name_ts ON metrics(name, timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_ts ON metrics(timestamp);

	CREATE TABLE IF NOT EXISTS metric_meta (
		name TEXT PRIMARY KEY,
		description TEXT,
		first_seen INTEGER NOT NULL,
		last_seen INTEGER NOT NULL,
		data_points INTEGER DEFAULT 0
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *EmbeddedStorage) backgroundFlusher() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.flush()
	}
}

func (s *EmbeddedStorage) flush() {
	s.bufferMu.Lock()
	if len(s.buffer) == 0 {
		s.bufferMu.Unlock()
		return
	}
	points := s.buffer
	s.buffer = make([]bufferedPoint, 0, 1000)
	s.bufferMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO metrics (name, timestamp, value, labels) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return
	}
	defer stmt.Close()

	metaStmt, err := tx.Prepare(`
		INSERT INTO metric_meta (name, first_seen, last_seen, data_points)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(name) DO UPDATE SET
			last_seen = excluded.last_seen,
			data_points = data_points + 1
	`)
	if err != nil {
		return
	}
	defer metaStmt.Close()

	for _, p := range points {
		labelsJSON := "{}"
		if len(p.labels) > 0 {
			if data, err := json.Marshal(p.labels); err == nil {
				labelsJSON = string(data)
			}
		}

		ts := p.timestamp.Unix()
		stmt.Exec(p.metric, ts, p.value, labelsJSON)
		metaStmt.Exec(p.metric, ts, ts)
	}

	tx.Commit()
}

// Record records a metric data point
func (s *EmbeddedStorage) Record(ctx context.Context, metric string, value float64, labels map[string]string, ts time.Time) error {
	s.bufferMu.Lock()
	s.buffer = append(s.buffer, bufferedPoint{
		metric:    metric,
		value:     value,
		labels:    labels,
		timestamp: ts,
	})
	s.bufferMu.Unlock()
	return nil
}

// Query queries metric data with aggregation
func (s *EmbeddedStorage) Query(ctx context.Context, metric string, from, to time.Time, aggregation AggregationType) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggFunc := getAggregationSQL(aggregation)
	query := fmt.Sprintf(`
		SELECT %s as value, labels
		FROM metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY labels
	`, aggFunc)

	rows, err := s.db.QueryContext(ctx, query, metric, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &QueryResult{
		Metric:      metric,
		Aggregation: string(aggregation),
		From:        from,
		To:          to,
		Series:      []TimeSeries{},
	}

	for rows.Next() {
		var value float64
		var labelsJSON string
		if err := rows.Scan(&value, &labelsJSON); err != nil {
			continue
		}

		var labels map[string]string
		json.Unmarshal([]byte(labelsJSON), &labels)

		result.Series = append(result.Series, TimeSeries{
			Metric: metric,
			Labels: labels,
			DataPoints: []DataPoint{
				{Timestamp: to, Value: value, Labels: labels},
			},
		})
	}

	return result, nil
}

// QueryRange queries metric data with time steps
func (s *EmbeddedStorage) QueryRange(ctx context.Context, metric string, from, to time.Time, step time.Duration, aggregation AggregationType) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stepSec := int64(step.Seconds())
	aggFunc := getAggregationSQL(aggregation)

	query := fmt.Sprintf(`
		SELECT
			(timestamp / ?) * ? as bucket,
			%s as value,
			labels
		FROM metrics
		WHERE name = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY bucket, labels
		ORDER BY bucket
	`, aggFunc)

	rows, err := s.db.QueryContext(ctx, query, stepSec, stepSec, metric, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group by labels
	seriesMap := make(map[string]*TimeSeries)

	for rows.Next() {
		var bucket int64
		var value float64
		var labelsJSON string
		if err := rows.Scan(&bucket, &value, &labelsJSON); err != nil {
			continue
		}

		var labels map[string]string
		json.Unmarshal([]byte(labelsJSON), &labels)

		key := labelsJSON
		if _, ok := seriesMap[key]; !ok {
			seriesMap[key] = &TimeSeries{
				Metric:     metric,
				Labels:     labels,
				DataPoints: []DataPoint{},
			}
		}

		seriesMap[key].DataPoints = append(seriesMap[key].DataPoints, DataPoint{
			Timestamp: time.Unix(bucket, 0),
			Value:     value,
			Labels:    labels,
		})
	}

	result := &QueryResult{
		Metric:      metric,
		Aggregation: string(aggregation),
		From:        from,
		To:          to,
		Step:        step.String(),
		Series:      make([]TimeSeries, 0, len(seriesMap)),
	}

	for _, series := range seriesMap {
		result.Series = append(result.Series, *series)
	}

	return result, nil
}

// ListMetrics returns all metric names
func (s *EmbeddedStorage) ListMetrics(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, "SELECT name FROM metric_meta ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		metrics = append(metrics, name)
	}

	return metrics, nil
}

// GetMetricMeta returns metadata for a metric
func (s *EmbeddedStorage) GetMetricMeta(ctx context.Context, metric string) (*MetricMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var meta MetricMeta
	var firstSeen, lastSeen int64

	err := s.db.QueryRowContext(ctx, `
		SELECT name, COALESCE(description, ''), first_seen, last_seen, data_points
		FROM metric_meta WHERE name = ?
	`, metric).Scan(&meta.Name, &meta.Description, &firstSeen, &lastSeen, &meta.DataPoints)

	if err != nil {
		return nil, err
	}

	meta.FirstSeen = time.Unix(firstSeen, 0)
	meta.LastSeen = time.Unix(lastSeen, 0)

	// Get unique labels
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT labels FROM metrics WHERE name = ? LIMIT 100
	`, metric)
	if err == nil {
		defer rows.Close()
		labelSet := make(map[string]bool)
		for rows.Next() {
			var labelsJSON string
			if err := rows.Scan(&labelsJSON); err != nil {
				continue
			}
			var labels map[string]string
			if json.Unmarshal([]byte(labelsJSON), &labels) == nil {
				for k := range labels {
					labelSet[k] = true
				}
			}
		}
		for k := range labelSet {
			meta.Labels = append(meta.Labels, k)
		}
		sort.Strings(meta.Labels)
	}

	return &meta, nil
}

// DeleteMetric deletes all data for a metric
func (s *EmbeddedStorage) DeleteMetric(ctx context.Context, metric string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM metrics WHERE name = ?", metric); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM metric_meta WHERE name = ?", metric); err != nil {
		return err
	}

	return tx.Commit()
}

// Cleanup removes old data based on retention policy
func (s *EmbeddedStorage) Cleanup(ctx context.Context, retention time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention).Unix()
	_, err := s.db.ExecContext(ctx, "DELETE FROM metrics WHERE timestamp < ?", cutoff)
	return err
}

// Close closes the storage
func (s *EmbeddedStorage) Close() error {
	s.flush() // Flush any remaining buffered data
	return s.db.Close()
}

func getAggregationSQL(agg AggregationType) string {
	switch agg {
	case AggregationSum:
		return "SUM(value)"
	case AggregationAvg:
		return "AVG(value)"
	case AggregationMin:
		return "MIN(value)"
	case AggregationMax:
		return "MAX(value)"
	case AggregationCount:
		return "COUNT(*)"
	case AggregationLast:
		return "value" // Will need ORDER BY timestamp DESC LIMIT 1
	default:
		return "AVG(value)"
	}
}
