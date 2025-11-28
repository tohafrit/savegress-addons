package devices

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

func newTestDevicesConfig() *config.DevicesConfig {
	return &config.DevicesConfig{
		HeartbeatInterval: 100 * time.Millisecond,
		OfflineThreshold:  200 * time.Millisecond,
		MaxDevices:        100,
		AutoDiscovery:     false,
	}
}

func TestNewRegistry(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.config != cfg {
		t.Error("config not set correctly")
	}
	if r.devices == nil {
		t.Error("devices should be initialized")
	}
	if r.groups == nil {
		t.Error("groups should be initialized")
	}
	if r.commands == nil {
		t.Error("commands should be initialized")
	}
	if r.events == nil {
		t.Error("events should be initialized")
	}
}

func TestRegistry_StartStop(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := r.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Start again should be idempotent
	err = r.Start(ctx)
	if err != nil {
		t.Fatalf("Second Start should not fail: %v", err)
	}

	r.Stop()

	// Stop again should be idempotent
	r.Stop()
}

func TestRegistry_SetStatusChangeCallback(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.SetStatusChangeCallback(func(device *models.Device, oldStatus, newStatus models.DeviceStatus) {
		// Callback set
	})

	if r.onStatusChange == nil {
		t.Error("expected callback to be set")
	}
}

func TestRegistry_Register(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{
		Name: "Test Device",
		Type: models.DeviceTypeSensor,
	}

	err := r.Register(device)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if device.ID == "" {
		t.Error("ID should be generated")
	}
	if device.Status != models.DeviceStatusOnline {
		t.Error("Status should be online")
	}
	if device.LastSeen == nil {
		t.Error("LastSeen should be set")
	}
	if device.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestRegistry_Register_WithID(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{
		ID:   "custom-id",
		Name: "Test Device",
	}

	err := r.Register(device)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if device.ID != "custom-id" {
		t.Error("Custom ID should be preserved")
	}
}

func TestRegistry_Register_MaxDevices(t *testing.T) {
	cfg := &config.DevicesConfig{
		MaxDevices: 2,
	}
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "d1"})
	r.Register(&models.Device{ID: "d2"})

	err := r.Register(&models.Device{ID: "d3"})
	if err != ErrMaxDevicesReached {
		t.Errorf("expected ErrMaxDevicesReached, got %v", err)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{ID: "device-001"}
	r.Register(device)

	err := r.Unregister("device-001")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	_, ok := r.Get("device-001")
	if ok {
		t.Error("device should be removed")
	}
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.Unregister("nonexistent")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	// Not found
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}

	// Found
	device := &models.Device{ID: "device-001", Name: "Test"}
	r.Register(device)

	got, ok := r.Get("device-001")
	if !ok {
		t.Fatal("expected to find device")
	}
	if got.Name != "Test" {
		t.Errorf("Name = %s, want Test", got.Name)
	}
}

func TestRegistry_Update(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{ID: "device-001", Name: "Original"}
	r.Register(device)

	device.Name = "Updated"
	err := r.Update(device)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := r.Get("device-001")
	if got.Name != "Updated" {
		t.Errorf("Name = %s, want Updated", got.Name)
	}
}

func TestRegistry_Update_NotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.Update(&models.Device{ID: "nonexistent"})
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "d1", Type: models.DeviceTypeSensor, Tags: map[string]string{"env": "prod"}})
	r.Register(&models.Device{ID: "d2", Type: models.DeviceTypeSensor, Tags: map[string]string{"env": "dev"}})
	r.Register(&models.Device{ID: "d3", Type: models.DeviceTypeActuator, Tags: map[string]string{"env": "prod"}})

	// List all
	all := r.List(DeviceFilter{})
	if len(all) != 3 {
		t.Errorf("expected 3 devices, got %d", len(all))
	}

	// Filter by type
	sensors := r.List(DeviceFilter{Type: models.DeviceTypeSensor})
	if len(sensors) != 2 {
		t.Errorf("expected 2 sensors, got %d", len(sensors))
	}

	// Filter by tag key
	prodDevices := r.List(DeviceFilter{TagKey: "env", TagValue: "prod"})
	if len(prodDevices) != 2 {
		t.Errorf("expected 2 prod devices, got %d", len(prodDevices))
	}

	// Filter by tag key only
	envDevices := r.List(DeviceFilter{TagKey: "env"})
	if len(envDevices) != 3 {
		t.Errorf("expected 3 devices with env tag, got %d", len(envDevices))
	}
}

func TestRegistry_List_ByStatus(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "d1"})
	r.Register(&models.Device{ID: "d2"})
	r.devices["d2"].Status = models.DeviceStatusOffline

	online := r.List(DeviceFilter{Status: models.DeviceStatusOnline})
	if len(online) != 1 {
		t.Errorf("expected 1 online device, got %d", len(online))
	}
}

func TestRegistry_Heartbeat(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{ID: "device-001"}
	r.Register(device)

	// Set to offline
	r.devices["device-001"].Status = models.DeviceStatusOffline

	var callbackCalled bool
	r.SetStatusChangeCallback(func(d *models.Device, old, new models.DeviceStatus) {
		callbackCalled = true
	})

	err := r.Heartbeat("device-001")
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Wait for callback goroutine

	if !callbackCalled {
		t.Error("status change callback should be called")
	}

	got, _ := r.Get("device-001")
	if got.Status != models.DeviceStatusOnline {
		t.Errorf("Status = %s, want online", got.Status)
	}
}

func TestRegistry_Heartbeat_NotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.Heartbeat("nonexistent")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestRegistry_UpdateTelemetryTime(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	device := &models.Device{ID: "device-001"}
	r.Register(device)
	r.devices["device-001"].Status = models.DeviceStatusOffline

	r.UpdateTelemetryTime("device-001")

	got, _ := r.Get("device-001")
	if got.LastTelemetry == nil {
		t.Error("LastTelemetry should be set")
	}
	if got.Status != models.DeviceStatusOnline {
		t.Error("Status should be online after telemetry")
	}
}

func TestRegistry_UpdateTelemetryTime_NonExistent(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	// Should not panic
	r.UpdateTelemetryTime("nonexistent")
}

func TestRegistry_CreateGroup(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	group := &models.DeviceGroup{
		Name:        "Test Group",
		Description: "A test group",
	}

	err := r.CreateGroup(group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group.ID == "" {
		t.Error("ID should be generated")
	}
	if group.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestRegistry_GetGroup(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	// Not found
	_, ok := r.GetGroup("nonexistent")
	if ok {
		t.Error("expected not found")
	}

	// Found
	group := &models.DeviceGroup{ID: "group-001", Name: "Test"}
	r.CreateGroup(group)

	got, ok := r.GetGroup("group-001")
	if !ok {
		t.Fatal("expected to find group")
	}
	if got.Name != "Test" {
		t.Errorf("Name = %s, want Test", got.Name)
	}
}

func TestRegistry_ListGroups(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.CreateGroup(&models.DeviceGroup{ID: "g1", Name: "Group 1"})
	r.CreateGroup(&models.DeviceGroup{ID: "g2", Name: "Group 2"})

	groups := r.ListGroups()
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestRegistry_AddToGroup(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})
	r.CreateGroup(&models.DeviceGroup{ID: "group-001"})

	err := r.AddToGroup("group-001", "device-001")
	if err != nil {
		t.Fatalf("AddToGroup failed: %v", err)
	}

	// Add again should be idempotent
	err = r.AddToGroup("group-001", "device-001")
	if err != nil {
		t.Fatalf("Second AddToGroup should not fail: %v", err)
	}

	group, _ := r.GetGroup("group-001")
	if len(group.DeviceIDs) != 1 {
		t.Errorf("expected 1 device in group, got %d", len(group.DeviceIDs))
	}
}

func TestRegistry_AddToGroup_GroupNotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})

	err := r.AddToGroup("nonexistent", "device-001")
	if err != ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestRegistry_AddToGroup_DeviceNotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.CreateGroup(&models.DeviceGroup{ID: "group-001"})

	err := r.AddToGroup("group-001", "nonexistent")
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestRegistry_RemoveFromGroup(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})
	r.CreateGroup(&models.DeviceGroup{ID: "group-001"})
	r.AddToGroup("group-001", "device-001")

	err := r.RemoveFromGroup("group-001", "device-001")
	if err != nil {
		t.Fatalf("RemoveFromGroup failed: %v", err)
	}

	// Remove non-member should succeed
	err = r.RemoveFromGroup("group-001", "device-001")
	if err != nil {
		t.Fatalf("Removing non-member should not fail: %v", err)
	}
}

func TestRegistry_RemoveFromGroup_GroupNotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.RemoveFromGroup("nonexistent", "device-001")
	if err != ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestRegistry_GetGroupDevices(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "d1", Name: "Device 1"})
	r.Register(&models.Device{ID: "d2", Name: "Device 2"})
	r.CreateGroup(&models.DeviceGroup{ID: "group-001"})
	r.AddToGroup("group-001", "d1")
	r.AddToGroup("group-001", "d2")

	devices := r.GetGroupDevices("group-001")
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}

	// Non-existent group
	devices = r.GetGroupDevices("nonexistent")
	if devices != nil {
		t.Error("expected nil for non-existent group")
	}
}

func TestRegistry_SendCommand(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})

	cmd := &models.Command{
		DeviceID: "device-001",
		Name:     "reboot",
	}

	err := r.SendCommand(cmd)
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	if cmd.ID == "" {
		t.Error("ID should be generated")
	}
	if cmd.Status != models.CommandStatusPending {
		t.Error("Status should be pending")
	}
	if cmd.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestRegistry_SendCommand_DeviceNotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.SendCommand(&models.Command{DeviceID: "nonexistent"})
	if err != ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestRegistry_GetCommand(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	// Not found
	_, ok := r.GetCommand("nonexistent")
	if ok {
		t.Error("expected not found")
	}

	// Found
	r.Register(&models.Device{ID: "device-001"})
	cmd := &models.Command{ID: "cmd-001", DeviceID: "device-001", Name: "reboot"}
	r.SendCommand(cmd)

	got, ok := r.GetCommand("cmd-001")
	if !ok {
		t.Fatal("expected to find command")
	}
	if got.Name != "reboot" {
		t.Errorf("Name = %s, want reboot", got.Name)
	}
}

func TestRegistry_UpdateCommandStatus(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})
	r.SendCommand(&models.Command{ID: "cmd-001", DeviceID: "device-001"})

	// Update to sent
	err := r.UpdateCommandStatus("cmd-001", models.CommandStatusSent, nil, "")
	if err != nil {
		t.Fatalf("UpdateCommandStatus failed: %v", err)
	}

	cmd, _ := r.GetCommand("cmd-001")
	if cmd.SentAt == nil {
		t.Error("SentAt should be set")
	}

	// Update to acked
	err = r.UpdateCommandStatus("cmd-001", models.CommandStatusAcked, nil, "")
	if err != nil {
		t.Fatalf("UpdateCommandStatus failed: %v", err)
	}

	cmd, _ = r.GetCommand("cmd-001")
	if cmd.AckedAt == nil {
		t.Error("AckedAt should be set")
	}

	// Update to completed
	err = r.UpdateCommandStatus("cmd-001", models.CommandStatusCompleted, map[string]string{"result": "ok"}, "")
	if err != nil {
		t.Fatalf("UpdateCommandStatus failed: %v", err)
	}

	cmd, _ = r.GetCommand("cmd-001")
	if cmd.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
	if cmd.Result == nil {
		t.Error("Result should be set")
	}

	// Update to failed
	r.SendCommand(&models.Command{ID: "cmd-002", DeviceID: "device-001"})
	err = r.UpdateCommandStatus("cmd-002", models.CommandStatusFailed, nil, "error message")
	if err != nil {
		t.Fatalf("UpdateCommandStatus failed: %v", err)
	}

	cmd, _ = r.GetCommand("cmd-002")
	if cmd.Error != "error message" {
		t.Errorf("Error = %s, want 'error message'", cmd.Error)
	}
}

func TestRegistry_UpdateCommandStatus_NotFound(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	err := r.UpdateCommandStatus("nonexistent", models.CommandStatusCompleted, nil, "")
	if err != ErrCommandNotFound {
		t.Errorf("expected ErrCommandNotFound, got %v", err)
	}
}

func TestRegistry_GetPendingCommands(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "device-001"})
	r.Register(&models.Device{ID: "device-002"})

	r.SendCommand(&models.Command{ID: "cmd-001", DeviceID: "device-001"})
	r.SendCommand(&models.Command{ID: "cmd-002", DeviceID: "device-001"})
	r.SendCommand(&models.Command{ID: "cmd-003", DeviceID: "device-002"})
	r.UpdateCommandStatus("cmd-002", models.CommandStatusCompleted, nil, "")

	pending := r.GetPendingCommands("device-001")
	if len(pending) != 1 {
		t.Errorf("expected 1 pending command, got %d", len(pending))
	}
}

func TestRegistry_RecordEvent(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	event := &models.DeviceEvent{
		DeviceID:  "device-001",
		EventType: "startup",
		Message:   "Device started",
	}

	r.RecordEvent(event)

	if event.ID == "" {
		t.Error("ID should be generated")
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestRegistry_GetDeviceEvents(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.RecordEvent(&models.DeviceEvent{ID: "e1", DeviceID: "device-001", EventType: "event1", Timestamp: time.Now()})
	r.RecordEvent(&models.DeviceEvent{ID: "e2", DeviceID: "device-001", EventType: "event2", Timestamp: time.Now()})
	r.RecordEvent(&models.DeviceEvent{ID: "e3", DeviceID: "device-001", EventType: "event3", Timestamp: time.Now()})

	// Get all
	events := r.GetDeviceEvents("device-001", 0)
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	// Get with limit
	events = r.GetDeviceEvents("device-001", 2)
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(events))
	}
}

func TestRegistry_GetStats(t *testing.T) {
	cfg := newTestDevicesConfig()
	r := NewRegistry(cfg)

	r.Register(&models.Device{ID: "d1", Type: models.DeviceTypeSensor})
	r.Register(&models.Device{ID: "d2", Type: models.DeviceTypeSensor})
	r.Register(&models.Device{ID: "d3", Type: models.DeviceTypeActuator})
	r.devices["d3"].Status = models.DeviceStatusOffline

	stats := r.GetStats()

	if stats.TotalDevices != 3 {
		t.Errorf("TotalDevices = %d, want 3", stats.TotalDevices)
	}
	if stats.OnlineDevices != 2 {
		t.Errorf("OnlineDevices = %d, want 2", stats.OnlineDevices)
	}
	if stats.OfflineDevices != 1 {
		t.Errorf("OfflineDevices = %d, want 1", stats.OfflineDevices)
	}
	if stats.DevicesByType["sensor"] != 2 {
		t.Errorf("DevicesByType[sensor] = %d, want 2", stats.DevicesByType["sensor"])
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		err  *Error
		want string
	}{
		{ErrDeviceNotFound, "Device not found"},
		{ErrGroupNotFound, "Group not found"},
		{ErrCommandNotFound, "Command not found"},
		{ErrMaxDevicesReached, "Maximum number of devices reached"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Code, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDeviceFilter_Struct(t *testing.T) {
	filter := DeviceFilter{
		Type:     models.DeviceTypeSensor,
		Status:   models.DeviceStatusOnline,
		GroupID:  "group-001",
		TagKey:   "env",
		TagValue: "prod",
		Limit:    10,
	}

	if filter.Type != models.DeviceTypeSensor {
		t.Error("Type not set")
	}
	if filter.Limit != 10 {
		t.Error("Limit not set")
	}
}
