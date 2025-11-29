package observability

import (
	"context"
	"fmt"
	"time"
)

// Span represents a trace span
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]interface{}
	Events     []SpanEvent
	Status     SpanStatus
}

// SpanEvent represents an event in a span
type SpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]interface{}
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    StatusCode
	Message string
}

// StatusCode represents a span status code
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// Tracer interface for distributed tracing
type Tracer interface {
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
	End(span Span)
}

// SpanOption configures a span
type SpanOption func(*Span)

// WithAttributes sets span attributes
func WithAttributes(attrs map[string]interface{}) SpanOption {
	return func(s *Span) {
		if s.Attributes == nil {
			s.Attributes = make(map[string]interface{})
		}
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// SimpleTracer is a basic tracer implementation
type SimpleTracer struct {
	serviceName string
	exporter    SpanExporter
}

// SpanExporter exports spans
type SpanExporter interface {
	ExportSpan(span Span) error
}

// NewSimpleTracer creates a new simple tracer
func NewSimpleTracer(serviceName string, exporter SpanExporter) *SimpleTracer {
	return &SimpleTracer{
		serviceName: serviceName,
		exporter:    exporter,
	}
}

// Start starts a new span
func (t *SimpleTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	span := Span{
		TraceID:    generateTraceID(ctx),
		SpanID:     generateSpanID(),
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     []SpanEvent{},
		Status: SpanStatus{
			Code: StatusCodeUnset,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(&span)
	}

	// Add service name
	span.Attributes["service.name"] = t.serviceName

	// Extract parent span from context
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		span.ParentID = parentSpan.SpanID
		span.TraceID = parentSpan.TraceID
	}

	// Store span in context
	ctx = ContextWithSpan(ctx, &span)

	return ctx, span
}

// End ends a span
func (t *SimpleTracer) End(span Span) {
	span.EndTime = time.Now()

	// Export span
	if t.exporter != nil {
		if err := t.exporter.ExportSpan(span); err != nil {
			// Log error (simplified)
			fmt.Printf("Failed to export span: %v\n", err)
		}
	}
}

// RecordError records an error in the span
func (s *Span) RecordError(err error) {
	s.Status.Code = StatusCodeError
	s.Status.Message = err.Error()
	s.AddEvent("error", map[string]interface{}{
		"error.message": err.Error(),
	})
}

// SetStatus sets the span status
func (s *Span) SetStatus(code StatusCode, message string) {
	s.Status.Code = code
	s.Status.Message = message
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attrs map[string]interface{}) {
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// Context keys for span storage
type spanContextKey struct{}

// ContextWithSpan stores a span in context
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey{}, span)
}

// SpanFromContext retrieves a span from context
func SpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(spanContextKey{}).(*Span)
	return span
}

// NoOpTracer is a tracer that does nothing
type NoOpTracer struct{}

func (n *NoOpTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, Span{}
}

func (n *NoOpTracer) End(span Span) {}

// InMemorySpanExporter exports spans to memory (for testing/debugging)
type InMemorySpanExporter struct {
	spans []Span
}

func (e *InMemorySpanExporter) ExportSpan(span Span) error {
	e.spans = append(e.spans, span)
	return nil
}

func (e *InMemorySpanExporter) GetSpans() []Span {
	return e.spans
}

// Helper functions

var (
	traceIDCounter uint64
	spanIDCounter  uint64
)

func generateTraceID(ctx context.Context) string {
	// Try to get existing trace ID from context
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}
