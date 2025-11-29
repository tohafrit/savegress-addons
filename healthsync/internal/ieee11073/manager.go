package ieee11073

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeviceManager manages IEEE 11073 medical devices
type DeviceManager struct {
	mu               sync.RWMutex
	devices          map[string]*ManagedDevice
	handlers         map[DeviceCategory]MeasurementHandler
	alertHandlers    []AlertHandler
	connectionState  map[string]AssociationState
	config           *ManagerConfig
}

// ManagedDevice represents a connected device
type ManagedDevice struct {
	Config           DeviceConfiguration
	State            AssociationState
	LastSeen         time.Time
	Measurements     []Measurement
	Alerts           []Alert
	Transport        DeviceTransport
}

// MeasurementHandler handles new measurements from devices
type MeasurementHandler func(ctx context.Context, measurement *Measurement) error

// AlertHandler handles device alerts
type AlertHandler func(ctx context.Context, alert *Alert) error

// ManagerConfig holds manager configuration
type ManagerConfig struct {
	EnableBluetooth   bool
	EnableBluetoothLE bool
	EnableUSB         bool
	ScanInterval      time.Duration
	ConnectionTimeout time.Duration
	MeasurementBuffer int
}

// DeviceTransport interface for device communication
type DeviceTransport interface {
	Connect(ctx context.Context, deviceID string) error
	Disconnect(deviceID string) error
	Send(deviceID string, data []byte) error
	Receive(deviceID string) ([]byte, error)
	IsConnected(deviceID string) bool
}

// NewDeviceManager creates a new device manager
func NewDeviceManager(config *ManagerConfig) *DeviceManager {
	if config == nil {
		config = &ManagerConfig{
			EnableBluetooth:   true,
			EnableBluetoothLE: true,
			ScanInterval:      30 * time.Second,
			ConnectionTimeout: 10 * time.Second,
			MeasurementBuffer: 1000,
		}
	}

	return &DeviceManager{
		devices:         make(map[string]*ManagedDevice),
		handlers:        make(map[DeviceCategory]MeasurementHandler),
		alertHandlers:   make([]AlertHandler, 0),
		connectionState: make(map[string]AssociationState),
		config:          config,
	}
}

// RegisterDevice registers a new device
func (m *DeviceManager) RegisterDevice(config DeviceConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deviceID := config.DeviceID.ID
	if deviceID == "" {
		return fmt.Errorf("device ID is required")
	}

	if _, exists := m.devices[deviceID]; exists {
		return fmt.Errorf("device already registered: %s", deviceID)
	}

	m.devices[deviceID] = &ManagedDevice{
		Config:       config,
		State:        StateDisconnected,
		LastSeen:     time.Now(),
		Measurements: make([]Measurement, 0, m.config.MeasurementBuffer),
		Alerts:       make([]Alert, 0),
	}

	m.connectionState[deviceID] = StateDisconnected

	return nil
}

// UnregisterDevice removes a device
func (m *DeviceManager) UnregisterDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Disconnect if connected
	if device.Transport != nil && device.Transport.IsConnected(deviceID) {
		device.Transport.Disconnect(deviceID)
	}

	delete(m.devices, deviceID)
	delete(m.connectionState, deviceID)

	return nil
}

// Connect establishes connection to a device
func (m *DeviceManager) Connect(ctx context.Context, deviceID string) error {
	m.mu.Lock()
	device, exists := m.devices[deviceID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("device not found: %s", deviceID)
	}
	m.connectionState[deviceID] = StateConnecting
	m.mu.Unlock()

	// Create appropriate transport based on device type
	transport, err := m.createTransport(device.Config.TransportType)
	if err != nil {
		m.setDeviceState(deviceID, StateDisconnected)
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// Connect with timeout
	ctx, cancel := context.WithTimeout(ctx, m.config.ConnectionTimeout)
	defer cancel()

	if err := transport.Connect(ctx, deviceID); err != nil {
		m.setDeviceState(deviceID, StateDisconnected)
		return fmt.Errorf("failed to connect: %w", err)
	}

	m.mu.Lock()
	device.Transport = transport
	device.State = StateConnected
	device.LastSeen = time.Now()
	m.connectionState[deviceID] = StateConnected
	m.mu.Unlock()

	// Start association procedure
	go m.associate(ctx, deviceID)

	return nil
}

// Disconnect closes connection to a device
func (m *DeviceManager) Disconnect(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if device.Transport != nil {
		if err := device.Transport.Disconnect(deviceID); err != nil {
			return fmt.Errorf("failed to disconnect: %w", err)
		}
	}

	device.State = StateDisconnected
	device.Transport = nil
	m.connectionState[deviceID] = StateDisconnected

	return nil
}

// GetDeviceState returns the current state of a device
func (m *DeviceManager) GetDeviceState(deviceID string) (AssociationState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.connectionState[deviceID]
	if !exists {
		return StateDisconnected, fmt.Errorf("device not found: %s", deviceID)
	}

	return state, nil
}

// RegisterMeasurementHandler registers a handler for a device category
func (m *DeviceManager) RegisterMeasurementHandler(category DeviceCategory, handler MeasurementHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[category] = handler
}

// RegisterAlertHandler registers an alert handler
func (m *DeviceManager) RegisterAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertHandlers = append(m.alertHandlers, handler)
}

// ProcessMeasurement processes an incoming measurement
func (m *DeviceManager) ProcessMeasurement(ctx context.Context, measurement *Measurement) error {
	m.mu.Lock()
	device, exists := m.devices[measurement.DeviceID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("device not found: %s", measurement.DeviceID)
	}

	// Store measurement
	device.Measurements = append(device.Measurements, *measurement)
	if len(device.Measurements) > m.config.MeasurementBuffer {
		device.Measurements = device.Measurements[1:]
	}
	device.LastSeen = time.Now()

	// Get handler
	handler := m.handlers[device.Config.Category]
	m.mu.Unlock()

	// Call handler if registered
	if handler != nil {
		if err := handler(ctx, measurement); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}

	return nil
}

// ProcessAlert processes a device alert
func (m *DeviceManager) ProcessAlert(ctx context.Context, alert *Alert) error {
	m.mu.Lock()
	device, exists := m.devices[alert.DeviceID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("device not found: %s", alert.DeviceID)
	}

	device.Alerts = append(device.Alerts, *alert)
	handlers := make([]AlertHandler, len(m.alertHandlers))
	copy(handlers, m.alertHandlers)
	m.mu.Unlock()

	// Call all alert handlers
	for _, handler := range handlers {
		if err := handler(ctx, alert); err != nil {
			// Log but continue
			continue
		}
	}

	return nil
}

// GetMeasurements returns measurements for a device
func (m *DeviceManager) GetMeasurements(deviceID string, since time.Time, limit int) ([]Measurement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	var result []Measurement
	for _, meas := range device.Measurements {
		if meas.Timestamp.After(since) {
			result = append(result, meas)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// GetLatestMeasurement returns the most recent measurement for a specific code
func (m *DeviceManager) GetLatestMeasurement(deviceID string, code NomenclatureCode) (*Measurement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	for i := len(device.Measurements) - 1; i >= 0; i-- {
		if device.Measurements[i].Code == code {
			meas := device.Measurements[i]
			return &meas, nil
		}
	}

	return nil, fmt.Errorf("no measurement found for code: %d", code)
}

// ListDevices returns all registered devices
func (m *DeviceManager) ListDevices() []DeviceConfiguration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]DeviceConfiguration, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, device.Config)
	}

	return devices
}

// GetDevice returns a specific device
func (m *DeviceManager) GetDevice(deviceID string) (*ManagedDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return device, nil
}

// Helper methods

func (m *DeviceManager) setDeviceState(deviceID string, state AssociationState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if device, exists := m.devices[deviceID]; exists {
		device.State = state
	}
	m.connectionState[deviceID] = state
}

func (m *DeviceManager) createTransport(transportType TransportType) (DeviceTransport, error) {
	switch transportType {
	case TransportBluetooth:
		return NewBluetoothTransport(), nil
	case TransportBluetoothLE:
		return NewBluetoothLETransport(), nil
	case TransportUSB:
		return NewUSBTransport(), nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}

func (m *DeviceManager) associate(ctx context.Context, deviceID string) {
	m.setDeviceState(deviceID, StateAssociating)

	// Association procedure (simplified)
	// In full implementation, this would follow IEEE 11073-20601 protocol

	// 1. Send Association Request
	// 2. Wait for Association Response
	// 3. Exchange configuration

	// For now, simulate successful association
	time.Sleep(100 * time.Millisecond)

	m.setDeviceState(deviceID, StateConfiguring)

	// Configuration phase
	time.Sleep(50 * time.Millisecond)

	m.setDeviceState(deviceID, StateOperating)
}

// BluetoothTransport implements Bluetooth Classic transport
type BluetoothTransport struct {
	mu          sync.RWMutex
	connections map[string]bool
}

func NewBluetoothTransport() *BluetoothTransport {
	return &BluetoothTransport{
		connections: make(map[string]bool),
	}
}

func (t *BluetoothTransport) Connect(ctx context.Context, deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connections[deviceID] = true
	return nil
}

func (t *BluetoothTransport) Disconnect(deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.connections, deviceID)
	return nil
}

func (t *BluetoothTransport) Send(deviceID string, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (t *BluetoothTransport) Receive(deviceID string) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return nil, fmt.Errorf("not connected")
	}
	return nil, nil
}

func (t *BluetoothTransport) IsConnected(deviceID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connections[deviceID]
}

// BluetoothLETransport implements Bluetooth Low Energy transport
type BluetoothLETransport struct {
	mu          sync.RWMutex
	connections map[string]bool
}

func NewBluetoothLETransport() *BluetoothLETransport {
	return &BluetoothLETransport{
		connections: make(map[string]bool),
	}
}

func (t *BluetoothLETransport) Connect(ctx context.Context, deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connections[deviceID] = true
	return nil
}

func (t *BluetoothLETransport) Disconnect(deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.connections, deviceID)
	return nil
}

func (t *BluetoothLETransport) Send(deviceID string, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (t *BluetoothLETransport) Receive(deviceID string) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return nil, fmt.Errorf("not connected")
	}
	return nil, nil
}

func (t *BluetoothLETransport) IsConnected(deviceID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connections[deviceID]
}

// USBTransport implements USB transport
type USBTransport struct {
	mu          sync.RWMutex
	connections map[string]bool
}

func NewUSBTransport() *USBTransport {
	return &USBTransport{
		connections: make(map[string]bool),
	}
}

func (t *USBTransport) Connect(ctx context.Context, deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connections[deviceID] = true
	return nil
}

func (t *USBTransport) Disconnect(deviceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.connections, deviceID)
	return nil
}

func (t *USBTransport) Send(deviceID string, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (t *USBTransport) Receive(deviceID string) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.connections[deviceID] {
		return nil, fmt.Errorf("not connected")
	}
	return nil, nil
}

func (t *USBTransport) IsConnected(deviceID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connections[deviceID]
}
