package observability

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel represents logging levels
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	SetLevel(level LogLevel)
}

// StructuredLogger implements structured logging
type StructuredLogger struct {
	level      atomic.Int32
	output     io.Writer
	baseFields []Field
	sampler    *LogSampler
	mu         sync.Mutex
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogSampler implements log sampling to reduce log volume
type LogSampler struct {
	initial    int
	thereafter int
	counters   sync.Map // map[string]*samplerCounter
}

type samplerCounter struct {
	count atomic.Int64
}

// NewLogger creates a new structured logger
func NewLogger(output io.Writer, level LogLevel) *StructuredLogger {
	logger := &StructuredLogger{
		output:     output,
		baseFields: []Field{},
	}
	logger.level.Store(int32(level))
	return logger
}

// NewDefaultLogger creates a logger with stdout output
func NewDefaultLogger() *StructuredLogger {
	return NewLogger(os.Stdout, InfoLevel)
}

// SetLevel sets the log level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.level.Store(int32(level))
}

// WithSampling enables log sampling
func (l *StructuredLogger) WithSampling(initial, thereafter int) *StructuredLogger {
	l.sampler = &LogSampler{
		initial:    initial,
		thereafter: thereafter,
	}
	return l
}

// shouldSample determines if a log entry should be sampled
func (s *LogSampler) shouldSample(key string) bool {
	if s == nil {
		return true // No sampling configured
	}

	counterI, _ := s.counters.LoadOrStore(key, &samplerCounter{})
	counter := counterI.(*samplerCounter)
	count := counter.count.Add(1)

	if count <= int64(s.initial) {
		return true
	}

	return (count-int64(s.initial))%int64(s.thereafter) == 0
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	if LogLevel(l.level.Load()) <= DebugLevel {
		l.log(DebugLevel, msg, fields...)
	}
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	if LogLevel(l.level.Load()) <= InfoLevel {
		l.log(InfoLevel, msg, fields...)
	}
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	if LogLevel(l.level.Load()) <= WarnLevel {
		l.log(WarnLevel, msg, fields...)
	}
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	if LogLevel(l.level.Load()) <= ErrorLevel {
		l.log(ErrorLevel, msg, fields...)
	}
}

// With creates a new logger with additional base fields
func (l *StructuredLogger) With(fields ...Field) Logger {
	newLogger := &StructuredLogger{
		output:     l.output,
		baseFields: append(l.baseFields, fields...),
		sampler:    l.sampler,
	}
	newLogger.level.Store(l.level.Load())
	return newLogger
}

// log writes a log entry
func (l *StructuredLogger) log(level LogLevel, msg string, fields ...Field) {
	// Check sampling
	if !l.sampler.shouldSample(msg) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}

	// Add base fields
	for _, f := range l.baseFields {
		entry.Fields[f.Key] = f.Value
	}

	// Add message fields
	for _, f := range fields {
		entry.Fields[f.Key] = f.Value
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	// Write to output
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

func (n *NoOpLogger) Debug(msg string, fields ...Field) {}
func (n *NoOpLogger) Info(msg string, fields ...Field)  {}
func (n *NoOpLogger) Warn(msg string, fields ...Field)  {}
func (n *NoOpLogger) Error(msg string, fields ...Field) {}
func (n *NoOpLogger) With(fields ...Field) Logger       { return n }
func (n *NoOpLogger) SetLevel(level LogLevel)           {}
