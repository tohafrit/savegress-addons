package audit

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/healthsync/internal/config"
)

func TestNewLogger(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	if logger == nil {
		t.Fatal("expected logger to be created")
	}

	if logger.config != cfg {
		t.Error("config not set correctly")
	}

	if logger.events == nil {
		t.Error("events map not initialized")
	}

	if logger.eventCh == nil {
		t.Error("event channel not initialized")
	}
}

func TestLogger_StartStop(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start logger
	err := logger.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start logger: %v", err)
	}

	if !logger.running {
		t.Error("logger should be running")
	}

	// Starting again should be no-op
	err = logger.Start(ctx)
	if err != nil {
		t.Fatalf("second start should not fail: %v", err)
	}

	// Stop logger
	logger.Stop()

	if logger.running {
		t.Error("logger should not be running after stop")
	}

	// Stopping again should be safe
	logger.Stop()
}

func TestLogger_LogAccess(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	req := &AccessLogRequest{
		UserID:       "user-123",
		UserName:     "John Doe",
		UserRole:     "physician",
		IPAddress:    "192.168.1.100",
		Action:       "R",
		ResourceType: "Patient",
		ResourceID:   "patient-456",
		PatientID:    "patient-456",
		Purpose:      "treatment",
		Outcome:      "0",
		Query:        "name=John",
	}

	event := logger.LogAccess(ctx, req)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.ID == "" {
		t.Error("expected event ID to be set")
	}

	if event.Action != "R" {
		t.Errorf("expected action 'R', got %s", event.Action)
	}

	if event.Outcome != "0" {
		t.Errorf("expected outcome '0', got %s", event.Outcome)
	}

	if event.Type == nil || event.Type.Code != "110110" {
		t.Error("expected patient record type code")
	}

	if len(event.Agent) == 0 {
		t.Fatal("expected at least one agent")
	}

	if event.Agent[0].Name != "John Doe" {
		t.Errorf("expected agent name 'John Doe', got %s", event.Agent[0].Name)
	}

	if event.Agent[0].Network == nil || event.Agent[0].Network.Address != "192.168.1.100" {
		t.Error("expected IP address in network")
	}

	if len(event.Entity) == 0 {
		t.Fatal("expected at least one entity")
	}

	if event.Entity[0].Query != "name=John" {
		t.Errorf("expected query 'name=John', got %s", event.Entity[0].Query)
	}

	// Give time for async processing
	time.Sleep(50 * time.Millisecond)

	// Verify event was stored
	stored, ok := logger.GetEvent(event.ID)
	if !ok {
		t.Error("expected event to be stored")
	}

	if stored.ID != event.ID {
		t.Error("stored event ID mismatch")
	}
}

func TestLogger_LogAccess_Disabled(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: false,
	}
	logger := NewLogger(cfg)

	req := &AccessLogRequest{
		UserID: "user-123",
		Action: "R",
	}

	event := logger.LogAccess(context.Background(), req)

	if event != nil {
		t.Error("expected nil event when logging is disabled")
	}
}

func TestLogger_LogAccess_NoQuery(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	req := &AccessLogRequest{
		UserID:       "user-123",
		Action:       "R",
		ResourceType: "Patient",
		ResourceID:   "123",
	}

	event := logger.LogAccess(ctx, req)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	// Query should be empty when not provided
	if len(event.Entity) > 0 && event.Entity[0].Query != "" {
		t.Error("expected empty query")
	}
}

func TestLogger_LogSecurityEvent(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	req := &SecurityEventRequest{
		EventCode:          "110113",
		EventDisplay:       "Security Alert",
		SubtypeCode:        "110127",
		SubtypeDisplay:     "Emergency Override Started",
		Action:             "E",
		ActorID:            "user-789",
		ActorName:          "Jane Smith",
		IPAddress:          "10.0.0.50",
		Outcome:            "0",
		OutcomeDescription: "Override activated",
	}

	event := logger.LogSecurityEvent(ctx, req)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.Type == nil || event.Type.Code != "110113" {
		t.Error("expected security alert type code")
	}

	if len(event.Subtype) == 0 || event.Subtype[0].Code != "110127" {
		t.Error("expected emergency override subtype")
	}

	if event.OutcomeDesc != "Override activated" {
		t.Error("expected outcome description")
	}
}

func TestLogger_LogSecurityEvent_Disabled(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: false,
	}
	logger := NewLogger(cfg)

	req := &SecurityEventRequest{
		EventCode: "110113",
	}

	event := logger.LogSecurityEvent(context.Background(), req)

	if event != nil {
		t.Error("expected nil event when logging is disabled")
	}
}

func TestLogger_LogLogin_Success(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	event := logger.LogLogin("user-123", "John Doe", "192.168.1.1", true)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.Outcome != "0" {
		t.Errorf("expected outcome '0' for success, got %s", event.Outcome)
	}

	if event.Type.Code != "110114" {
		t.Error("expected user authentication code")
	}

	if len(event.Subtype) == 0 || event.Subtype[0].Code != "110122" {
		t.Error("expected login subtype")
	}
}

func TestLogger_LogLogin_Failure(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	event := logger.LogLogin("user-123", "John Doe", "192.168.1.1", false)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.Outcome != "8" {
		t.Errorf("expected outcome '8' for failure, got %s", event.Outcome)
	}
}

func TestLogger_LogLogout(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	event := logger.LogLogout("user-123", "John Doe", "192.168.1.1")

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.Outcome != "0" {
		t.Errorf("expected outcome '0', got %s", event.Outcome)
	}

	if len(event.Subtype) == 0 || event.Subtype[0].Code != "110123" {
		t.Error("expected logout subtype")
	}
}

func TestLogger_LogExport(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	req := &ExportLogRequest{
		UserID:      "user-123",
		UserName:    "John Doe",
		IPAddress:   "192.168.1.1",
		ExportType:  "PatientList",
		Description: "Monthly patient export",
		RecordCount: "500",
		Format:      "CSV",
	}

	event := logger.LogExport(ctx, req)

	if event == nil {
		t.Fatal("expected event to be created")
	}

	if event.Type.Code != "110106" {
		t.Error("expected export type code")
	}

	if event.Action != "R" {
		t.Error("expected read action for export")
	}

	if len(event.Entity) == 0 {
		t.Fatal("expected entity in export event")
	}

	if event.Entity[0].Name != "PatientList" {
		t.Error("expected export type name")
	}

	if event.Entity[0].Description != "Monthly patient export" {
		t.Error("expected description")
	}

	// Check details
	hasRecordCount := false
	hasFormat := false
	for _, detail := range event.Entity[0].Detail {
		if detail.Type == "RecordCount" && detail.ValueString == "500" {
			hasRecordCount = true
		}
		if detail.Type == "Format" && detail.ValueString == "CSV" {
			hasFormat = true
		}
	}

	if !hasRecordCount {
		t.Error("expected record count detail")
	}

	if !hasFormat {
		t.Error("expected format detail")
	}
}

func TestLogger_GetEvent(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	req := &AccessLogRequest{
		UserID: "user-123",
		Action: "R",
	}

	event := logger.LogAccess(ctx, req)
	time.Sleep(50 * time.Millisecond)

	// Get existing event
	retrieved, ok := logger.GetEvent(event.ID)
	if !ok {
		t.Error("expected to find event")
	}

	if retrieved.ID != event.ID {
		t.Error("retrieved event ID mismatch")
	}

	// Get non-existent event
	_, ok = logger.GetEvent("nonexistent")
	if ok {
		t.Error("expected not to find non-existent event")
	}
}

func TestLogger_GetEvents_NoFilter(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	// Log multiple events
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-2", Action: "C"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-3", Action: "U"})

	time.Sleep(100 * time.Millisecond)

	events := logger.GetEvents(EventFilter{})

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestLogger_GetEvents_FilterByAction(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-2", Action: "R"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-3", Action: "C"})

	time.Sleep(100 * time.Millisecond)

	events := logger.GetEvents(EventFilter{Action: "R"})

	if len(events) != 2 {
		t.Errorf("expected 2 read events, got %d", len(events))
	}

	for _, e := range events {
		if e.Action != "R" {
			t.Error("expected all events to have action R")
		}
	}
}

func TestLogger_GetEvents_FilterByOutcome(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Outcome: "0"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-2", Outcome: "8"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-3", Outcome: "0"})

	time.Sleep(100 * time.Millisecond)

	events := logger.GetEvents(EventFilter{Outcome: "0"})

	if len(events) != 2 {
		t.Errorf("expected 2 successful events, got %d", len(events))
	}
}

func TestLogger_GetEvents_FilterByUserID(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "C"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-2", Action: "R"})

	time.Sleep(100 * time.Millisecond)

	events := logger.GetEvents(EventFilter{UserID: "user-1"})

	if len(events) != 2 {
		t.Errorf("expected 2 events for user-1, got %d", len(events))
	}
}

func TestLogger_GetEvents_FilterByDateRange(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R"})

	time.Sleep(100 * time.Millisecond)

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Events should be within range
	events := logger.GetEvents(EventFilter{StartDate: &past, EndDate: &future})
	if len(events) != 1 {
		t.Errorf("expected 1 event in range, got %d", len(events))
	}

	// Events should be excluded before start date
	veryRecent := now.Add(1 * time.Second)
	events2 := logger.GetEvents(EventFilter{StartDate: &veryRecent})
	if len(events2) != 0 {
		t.Errorf("expected 0 events after start date, got %d", len(events2))
	}
}

func TestLogger_GetStats(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Start(ctx)
	defer logger.Stop()

	// Log various events
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R", Outcome: "0"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-2", Action: "C", Outcome: "0"})
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-3", Action: "R", Outcome: "8"})
	logger.LogLogin("user-4", "Test", "127.0.0.1", true)
	logger.LogLogin("user-5", "Test2", "127.0.0.1", false)

	time.Sleep(100 * time.Millisecond)

	stats := logger.GetStats()

	if stats.TotalEvents != 5 {
		t.Errorf("expected 5 total events, got %d", stats.TotalEvents)
	}

	if stats.FailedEvents != 2 {
		t.Errorf("expected 2 failed events, got %d", stats.FailedEvents)
	}

	if stats.ByAction["R"] != 2 {
		t.Errorf("expected 2 read actions, got %d", stats.ByAction["R"])
	}

	if stats.ByOutcome["0"] != 3 {
		t.Errorf("expected 3 successful outcomes, got %d", stats.ByOutcome["0"])
	}
}

func TestLogger_ProcessEvents_ContextCancellation(t *testing.T) {
	cfg := &config.AuditConfig{
		Enabled: true,
	}
	logger := NewLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	logger.Start(ctx)

	// Log an event
	logger.LogAccess(ctx, &AccessLogRequest{UserID: "user-1", Action: "R"})

	// Cancel context to stop processing
	cancel()

	// Give time for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Should not panic when logger is stopped via context
}
