package devices

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/pkg/models"
)

// Registry manages device registration and status
type Registry struct {
	config   *config.DevicesConfig
	devices  map[string]*models.Device
	groups   map[string]*models.DeviceGroup
	commands map[string]*models.Command
	events   map[string][]*models.DeviceEvent
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	onStatusChange func(device *models.Device, oldStatus, newStatus models.DeviceStatus)
}

// NewRegistry creates a new device registry
func NewRegistry(cfg *config.DevicesConfig) *Registry {
	return &Registry{
		config:   cfg,
		devices:  make(map[string]*models.Device),
		groups:   make(map[string]*models.DeviceGroup),
		commands: make(map[string]*models.Command),
		events:   make(map[string][]*models.DeviceEvent),
		stopCh:   make(chan struct{}),
	}
}

// Start starts the registry
func (r *Registry) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = true
	r.mu.Unlock()

	go r.monitorLoop(ctx)
	return nil
}

// Stop stops the registry
func (r *Registry) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		close(r.stopCh)
		r.running = false
	}
}

// SetStatusChangeCallback sets a callback for status changes
func (r *Registry) SetStatusChangeCallback(cb func(device *models.Device, oldStatus, newStatus models.DeviceStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStatusChange = cb
}

func (r *Registry) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(r.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkDeviceStatus()
		}
	}
}

func (r *Registry) checkDeviceStatus() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, device := range r.devices {
		if device.LastSeen == nil {
			continue
		}

		oldStatus := device.Status
		if now.Sub(*device.LastSeen) > r.config.OfflineThreshold {
			if device.Status == models.DeviceStatusOnline {
				device.Status = models.DeviceStatusOffline
				device.UpdatedAt = now

				if r.onStatusChange != nil {
					go r.onStatusChange(device, oldStatus, device.Status)
				}
			}
		}
	}
}

// Register registers a new device
func (r *Registry) Register(device *models.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.devices) >= r.config.MaxDevices {
		return ErrMaxDevicesReached
	}

	if device.ID == "" {
		device.ID = uuid.New().String()
	}

	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	device.Status = models.DeviceStatusOnline
	device.LastSeen = &now

	r.devices[device.ID] = device
	return nil
}

// Unregister removes a device
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.devices[id]; !ok {
		return ErrDeviceNotFound
	}

	delete(r.devices, id)
	delete(r.events, id)
	return nil
}

// Get retrieves a device by ID
func (r *Registry) Get(id string) (*models.Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	device, ok := r.devices[id]
	return device, ok
}

// Update updates a device
func (r *Registry) Update(device *models.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.devices[device.ID]; !ok {
		return ErrDeviceNotFound
	}

	device.UpdatedAt = time.Now()
	r.devices[device.ID] = device
	return nil
}

// List lists devices with optional filters
func (r *Registry) List(filter DeviceFilter) []*models.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.Device
	for _, device := range r.devices {
		if r.matchesFilter(device, filter) {
			results = append(results, device)
		}
	}
	return results
}

// DeviceFilter defines filters for device queries
type DeviceFilter struct {
	Type     models.DeviceType
	Status   models.DeviceStatus
	GroupID  string
	TagKey   string
	TagValue string
	Limit    int
}

func (r *Registry) matchesFilter(device *models.Device, filter DeviceFilter) bool {
	if filter.Type != "" && device.Type != filter.Type {
		return false
	}
	if filter.Status != "" && device.Status != filter.Status {
		return false
	}
	if filter.TagKey != "" {
		if val, ok := device.Tags[filter.TagKey]; !ok || (filter.TagValue != "" && val != filter.TagValue) {
			return false
		}
	}
	return true
}

// Heartbeat updates device last seen time
func (r *Registry) Heartbeat(deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	device, ok := r.devices[deviceID]
	if !ok {
		return ErrDeviceNotFound
	}

	now := time.Now()
	oldStatus := device.Status
	device.LastSeen = &now
	device.UpdatedAt = now

	if device.Status == models.DeviceStatusOffline {
		device.Status = models.DeviceStatusOnline
		if r.onStatusChange != nil {
			go r.onStatusChange(device, oldStatus, device.Status)
		}
	}

	return nil
}

// UpdateTelemetryTime updates the last telemetry timestamp
func (r *Registry) UpdateTelemetryTime(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if device, ok := r.devices[deviceID]; ok {
		now := time.Now()
		device.LastTelemetry = &now
		device.LastSeen = &now
		device.UpdatedAt = now

		if device.Status == models.DeviceStatusOffline {
			device.Status = models.DeviceStatusOnline
		}
	}
}

// Group Management

// CreateGroup creates a device group
func (r *Registry) CreateGroup(group *models.DeviceGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group.ID == "" {
		group.ID = uuid.New().String()
	}

	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	r.groups[group.ID] = group
	return nil
}

// GetGroup retrieves a group by ID
func (r *Registry) GetGroup(id string) (*models.DeviceGroup, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[id]
	return group, ok
}

// ListGroups lists all groups
func (r *Registry) ListGroups() []*models.DeviceGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.DeviceGroup
	for _, group := range r.groups {
		results = append(results, group)
	}
	return results
}

// AddToGroup adds a device to a group
func (r *Registry) AddToGroup(groupID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	group, ok := r.groups[groupID]
	if !ok {
		return ErrGroupNotFound
	}

	if _, ok := r.devices[deviceID]; !ok {
		return ErrDeviceNotFound
	}

	for _, id := range group.DeviceIDs {
		if id == deviceID {
			return nil // Already in group
		}
	}

	group.DeviceIDs = append(group.DeviceIDs, deviceID)
	group.UpdatedAt = time.Now()
	return nil
}

// RemoveFromGroup removes a device from a group
func (r *Registry) RemoveFromGroup(groupID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	group, ok := r.groups[groupID]
	if !ok {
		return ErrGroupNotFound
	}

	for i, id := range group.DeviceIDs {
		if id == deviceID {
			group.DeviceIDs = append(group.DeviceIDs[:i], group.DeviceIDs[i+1:]...)
			group.UpdatedAt = time.Now()
			return nil
		}
	}

	return nil
}

// GetGroupDevices gets all devices in a group
func (r *Registry) GetGroupDevices(groupID string) []*models.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	group, ok := r.groups[groupID]
	if !ok {
		return nil
	}

	var devices []*models.Device
	for _, deviceID := range group.DeviceIDs {
		if device, ok := r.devices[deviceID]; ok {
			devices = append(devices, device)
		}
	}
	return devices
}

// Command Management

// SendCommand sends a command to a device
func (r *Registry) SendCommand(cmd *models.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.devices[cmd.DeviceID]; !ok {
		return ErrDeviceNotFound
	}

	if cmd.ID == "" {
		cmd.ID = uuid.New().String()
	}

	now := time.Now()
	cmd.CreatedAt = now
	cmd.Status = models.CommandStatusPending

	r.commands[cmd.ID] = cmd
	return nil
}

// GetCommand retrieves a command by ID
func (r *Registry) GetCommand(id string) (*models.Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[id]
	return cmd, ok
}

// UpdateCommandStatus updates command status
func (r *Registry) UpdateCommandStatus(id string, status models.CommandStatus, result interface{}, err string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, ok := r.commands[id]
	if !ok {
		return ErrCommandNotFound
	}

	now := time.Now()
	cmd.Status = status
	cmd.Result = result
	cmd.Error = err

	switch status {
	case models.CommandStatusSent:
		cmd.SentAt = &now
	case models.CommandStatusAcked:
		cmd.AckedAt = &now
	case models.CommandStatusCompleted, models.CommandStatusFailed:
		cmd.CompletedAt = &now
	}

	return nil
}

// GetPendingCommands gets pending commands for a device
func (r *Registry) GetPendingCommands(deviceID string) []*models.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pending []*models.Command
	for _, cmd := range r.commands {
		if cmd.DeviceID == deviceID && cmd.Status == models.CommandStatusPending {
			pending = append(pending, cmd)
		}
	}
	return pending
}

// Event Management

// RecordEvent records a device event
func (r *Registry) RecordEvent(event *models.DeviceEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	r.events[event.DeviceID] = append(r.events[event.DeviceID], event)
}

// GetDeviceEvents gets events for a device
func (r *Registry) GetDeviceEvents(deviceID string, limit int) []*models.DeviceEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.events[deviceID]
	if limit > 0 && len(events) > limit {
		return events[len(events)-limit:]
	}
	return events
}

// GetStats returns registry statistics
func (r *Registry) GetStats() *models.SystemStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &models.SystemStats{
		DevicesByType:   make(map[string]int),
		DevicesByStatus: make(map[string]int),
	}

	for _, device := range r.devices {
		stats.TotalDevices++
		stats.DevicesByType[string(device.Type)]++
		stats.DevicesByStatus[string(device.Status)]++

		switch device.Status {
		case models.DeviceStatusOnline:
			stats.OnlineDevices++
		case models.DeviceStatusOffline:
			stats.OfflineDevices++
		}
	}

	return stats
}

// Errors
var (
	ErrDeviceNotFound    = &Error{Code: "DEVICE_NOT_FOUND", Message: "Device not found"}
	ErrGroupNotFound     = &Error{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	ErrCommandNotFound   = &Error{Code: "COMMAND_NOT_FOUND", Message: "Command not found"}
	ErrMaxDevicesReached = &Error{Code: "MAX_DEVICES_REACHED", Message: "Maximum number of devices reached"}
)

// Error represents a registry error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
