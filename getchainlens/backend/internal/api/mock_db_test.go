package api

import (
	"context"
	"sync"
	"time"

	"getchainlens.com/chainlens/backend/internal/database"
)

// MockDB implements database methods for testing
type MockDB struct {
	mu sync.RWMutex

	// Users storage
	users          map[string]*database.User
	usersByEmail   map[string]*database.User
	usersByStripe  map[string]*database.User

	// Refresh tokens
	refreshTokens map[string]*database.RefreshToken

	// Licenses
	licenses map[string]*database.License

	// Projects
	projects map[string]*database.Project

	// Contracts
	contracts map[string]*database.Contract

	// Monitors
	monitors map[string]*database.Monitor

	// Alerts
	alerts map[string]*database.Alert

	// Error injection for testing error paths
	CreateUserErr       error
	GetUserByIDErr      error
	GetUserByEmailErr   error
	UpdateUserErr       error
	CreateRefreshErr    error
	GetRefreshErr       error
	CreateProjectErr    error
	GetProjectsErr      error
	CreateContractErr   error
	GetContractsErr     error
	CreateMonitorErr    error
	GetMonitorsErr      error
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		users:          make(map[string]*database.User),
		usersByEmail:   make(map[string]*database.User),
		usersByStripe:  make(map[string]*database.User),
		refreshTokens:  make(map[string]*database.RefreshToken),
		licenses:       make(map[string]*database.License),
		projects:       make(map[string]*database.Project),
		contracts:      make(map[string]*database.Contract),
		monitors:       make(map[string]*database.Monitor),
		alerts:         make(map[string]*database.Alert),
	}
}

// CreateUser creates a new user
func (m *MockDB) CreateUser(ctx context.Context, email, passwordHash, name string) (*database.User, error) {
	if m.CreateUserErr != nil {
		return nil, m.CreateUserErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.usersByEmail[email]; exists {
		return nil, database.ErrDuplicateEmail
	}

	user := &database.User{
		ID:              generateMockID("usr"),
		Email:           email,
		PasswordHash:    passwordHash,
		Name:            name,
		Plan:            "free",
		APICallsUsed:    0,
		APICallsResetAt: time.Now().AddDate(0, 1, 0),
		EmailVerified:   false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	m.users[user.ID] = user
	m.usersByEmail[email] = user

	return user, nil
}

// GetUserByID retrieves a user by ID
func (m *MockDB) GetUserByID(ctx context.Context, id string) (*database.User, error) {
	if m.GetUserByIDErr != nil {
		return nil, m.GetUserByIDErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, database.ErrNotFound
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	if m.GetUserByEmailErr != nil {
		return nil, m.GetUserByEmailErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, database.ErrNotFound
	}

	return user, nil
}

// UpdateUser updates user information
func (m *MockDB) UpdateUser(ctx context.Context, id, name string) error {
	if m.UpdateUserErr != nil {
		return m.UpdateUserErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return database.ErrNotFound
	}

	user.Name = name
	user.UpdatedAt = time.Now()

	return nil
}

// UpdateUserPassword updates user password
func (m *MockDB) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return database.ErrNotFound
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()

	return nil
}

// UpdateUserPlan updates user subscription plan
func (m *MockDB) UpdateUserPlan(ctx context.Context, id, plan string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return database.ErrNotFound
	}

	user.Plan = plan
	user.UpdatedAt = time.Now()

	return nil
}

// UpdateUserStripe updates Stripe customer info
func (m *MockDB) UpdateUserStripe(ctx context.Context, id, customerID, subscriptionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return database.ErrNotFound
	}

	user.StripeCustomerID = &customerID
	user.StripeSubscID = &subscriptionID
	user.UpdatedAt = time.Now()

	m.usersByStripe[customerID] = user

	return nil
}

// GetUserByStripeCustomerID retrieves a user by Stripe customer ID
func (m *MockDB) GetUserByStripeCustomerID(ctx context.Context, customerID string) (*database.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByStripe[customerID]
	if !ok {
		return nil, database.ErrNotFound
	}

	return user, nil
}

// IncrementAPIUsage increments API call counter
func (m *MockDB) IncrementAPIUsage(ctx context.Context, userID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return 0, database.ErrNotFound
	}

	user.APICallsUsed++
	return user.APICallsUsed, nil
}

// CreateRefreshToken creates a new refresh token
func (m *MockDB) CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) (*database.RefreshToken, error) {
	if m.CreateRefreshErr != nil {
		return nil, m.CreateRefreshErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rt := &database.RefreshToken{
		ID:        generateMockID("rt"),
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	m.refreshTokens[token] = rt

	return rt, nil
}

// GetRefreshToken retrieves a refresh token
func (m *MockDB) GetRefreshToken(ctx context.Context, token string) (*database.RefreshToken, error) {
	if m.GetRefreshErr != nil {
		return nil, m.GetRefreshErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	rt, ok := m.refreshTokens[token]
	if !ok || rt.ExpiresAt.Before(time.Now()) {
		return nil, database.ErrInvalidToken
	}

	return rt, nil
}

// DeleteRefreshToken deletes a refresh token
func (m *MockDB) DeleteRefreshToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.refreshTokens, token)
	return nil
}

// DeleteUserRefreshTokens deletes all refresh tokens for a user
func (m *MockDB) DeleteUserRefreshTokens(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token, rt := range m.refreshTokens {
		if rt.UserID == userID {
			delete(m.refreshTokens, token)
		}
	}

	return nil
}

// CreateLicense creates a new license
func (m *MockDB) CreateLicense(ctx context.Context, userID, tier string, features []string, maxDevices int) (*database.License, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lic := &database.License{
		ID:         generateMockID("lic"),
		UserID:     userID,
		LicenseKey: generateMockLicenseKey(),
		Tier:       tier,
		Features:   features,
		MaxDevices: maxDevices,
		CreatedAt:  time.Now(),
	}

	m.licenses[lic.LicenseKey] = lic

	return lic, nil
}

// GetLicenseByKey retrieves a license by key
func (m *MockDB) GetLicenseByKey(ctx context.Context, licenseKey string) (*database.License, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lic, ok := m.licenses[licenseKey]
	if !ok {
		return nil, database.ErrNotFound
	}

	return lic, nil
}

// CreateProject creates a new project
func (m *MockDB) CreateProject(ctx context.Context, userID, name, description string) (*database.Project, error) {
	if m.CreateProjectErr != nil {
		return nil, m.CreateProjectErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	project := &database.Project{
		ID:          generateMockID("prj"),
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.projects[project.ID] = project

	return project, nil
}

// GetProjectsByUser retrieves all projects for a user
func (m *MockDB) GetProjectsByUser(ctx context.Context, userID string) ([]*database.Project, error) {
	if m.GetProjectsErr != nil {
		return nil, m.GetProjectsErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var projects []*database.Project
	for _, p := range m.projects {
		if p.UserID == userID {
			projects = append(projects, p)
		}
	}

	return projects, nil
}

// GetProjectByID retrieves a project by ID
func (m *MockDB) GetProjectByID(ctx context.Context, id string) (*database.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	project, ok := m.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}

	return project, nil
}

// UpdateProject updates a project
func (m *MockDB) UpdateProject(ctx context.Context, id, name, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project, ok := m.projects[id]
	if !ok {
		return database.ErrNotFound
	}

	project.Name = name
	project.Description = description
	project.UpdatedAt = time.Now()

	return nil
}

// DeleteProject deletes a project
func (m *MockDB) DeleteProject(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.projects, id)
	return nil
}

// CreateContract creates a new contract
func (m *MockDB) CreateContract(ctx context.Context, projectID, userID, address, chain, name string) (*database.Contract, error) {
	if m.CreateContractErr != nil {
		return nil, m.CreateContractErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	contract := &database.Contract{
		ID:         generateMockID("ctr"),
		ProjectID:  projectID,
		UserID:     userID,
		Address:    address,
		Chain:      chain,
		Name:       name,
		IsVerified: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.contracts[contract.ID] = contract

	return contract, nil
}

// GetContractsByProject retrieves all contracts for a project
func (m *MockDB) GetContractsByProject(ctx context.Context, projectID string) ([]*database.Contract, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var contracts []*database.Contract
	for _, c := range m.contracts {
		if c.ProjectID == projectID {
			contracts = append(contracts, c)
		}
	}

	return contracts, nil
}

// GetContractsByUser retrieves all contracts for a user
func (m *MockDB) GetContractsByUser(ctx context.Context, userID string) ([]*database.Contract, error) {
	if m.GetContractsErr != nil {
		return nil, m.GetContractsErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var contracts []*database.Contract
	for _, c := range m.contracts {
		if c.UserID == userID {
			contracts = append(contracts, c)
		}
	}

	return contracts, nil
}

// DeleteContract deletes a contract
func (m *MockDB) DeleteContract(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.contracts, id)
	return nil
}

// CreateMonitor creates a new monitor
func (m *MockDB) CreateMonitor(ctx context.Context, userID, contractID, name, webhookURL string, eventFilters []string) (*database.Monitor, error) {
	if m.CreateMonitorErr != nil {
		return nil, m.CreateMonitorErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	monitor := &database.Monitor{
		ID:           generateMockID("mon"),
		UserID:       userID,
		ContractID:   contractID,
		Name:         name,
		EventFilters: eventFilters,
		WebhookURL:   webhookURL,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.monitors[monitor.ID] = monitor

	return monitor, nil
}

// GetMonitorsByUser retrieves all monitors for a user
func (m *MockDB) GetMonitorsByUser(ctx context.Context, userID string) ([]*database.Monitor, error) {
	if m.GetMonitorsErr != nil {
		return nil, m.GetMonitorsErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var monitors []*database.Monitor
	for _, mon := range m.monitors {
		if mon.UserID == userID {
			monitors = append(monitors, mon)
		}
	}

	return monitors, nil
}

// GetMonitorByID retrieves a monitor by ID
func (m *MockDB) GetMonitorByID(ctx context.Context, id string) (*database.Monitor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	monitor, ok := m.monitors[id]
	if !ok {
		return nil, database.ErrNotFound
	}

	return monitor, nil
}

// UpdateMonitor updates a monitor
func (m *MockDB) UpdateMonitor(ctx context.Context, id, name, webhookURL string, eventFilters []string, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	monitor, ok := m.monitors[id]
	if !ok {
		return database.ErrNotFound
	}

	monitor.Name = name
	monitor.WebhookURL = webhookURL
	monitor.EventFilters = eventFilters
	monitor.IsActive = isActive
	monitor.UpdatedAt = time.Now()

	return nil
}

// DeleteMonitor deletes a monitor
func (m *MockDB) DeleteMonitor(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.monitors, id)
	return nil
}

// CreateAlert creates a new alert
func (m *MockDB) CreateAlert(ctx context.Context, monitorID, userID, alertType, severity, message, data string) (*database.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert := &database.Alert{
		ID:        generateMockID("alt"),
		MonitorID: monitorID,
		UserID:    userID,
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Data:      data,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	m.alerts[alert.ID] = alert

	return alert, nil
}

// GetAlertsByUser retrieves alerts for a user
func (m *MockDB) GetAlertsByUser(ctx context.Context, userID string, limit int) ([]*database.Alert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var alerts []*database.Alert
	count := 0
	for _, a := range m.alerts {
		if a.UserID == userID {
			alerts = append(alerts, a)
			count++
			if count >= limit {
				break
			}
		}
	}

	return alerts, nil
}

// MarkAlertRead marks an alert as read
func (m *MockDB) MarkAlertRead(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[id]
	if !ok {
		return database.ErrNotFound
	}

	alert.IsRead = true
	return nil
}

// TrackAPIUsage tracks API usage
func (m *MockDB) TrackAPIUsage(ctx context.Context, userID, endpoint, method string, statusCode int, durationMs int64) error {
	// No-op for mock
	return nil
}

// ResetAPIUsage resets API usage for a user
func (m *MockDB) ResetAPIUsage(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}

	user.APICallsUsed = 0
	user.APICallsResetAt = time.Now().AddDate(0, 1, 0)

	return nil
}

// GetAPIUsageStats retrieves API usage stats
func (m *MockDB) GetAPIUsageStats(ctx context.Context, userID string, since time.Time) (int64, error) {
	return 100, nil // Mock value
}

// SaveAnalysisResult saves an analysis result
func (m *MockDB) SaveAnalysisResult(ctx context.Context, userID, sourceHash, status, issuesJSON, gasJSON, scoreJSON string, durationMs int64) (*database.AnalysisResult, error) {
	return &database.AnalysisResult{
		ID:         generateMockID("anl"),
		UserID:     userID,
		SourceHash: sourceHash,
		Status:     status,
		IssuesJSON: issuesJSON,
		GasJSON:    gasJSON,
		ScoreJSON:  scoreJSON,
		DurationMs: durationMs,
		CreatedAt:  time.Now(),
	}, nil
}

// Helper to add a test user
func (m *MockDB) AddTestUser(id, email, passwordHash, name, plan string) *database.User {
	m.mu.Lock()
	defer m.mu.Unlock()

	user := &database.User{
		ID:              id,
		Email:           email,
		PasswordHash:    passwordHash,
		Name:            name,
		Plan:            plan,
		APICallsUsed:    0,
		APICallsResetAt: time.Now().AddDate(0, 1, 0),
		EmailVerified:   true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	m.users[id] = user
	m.usersByEmail[email] = user

	return user
}

// Helper functions
var mockIDCounter int

func generateMockID(prefix string) string {
	mockIDCounter++
	return prefix + "_mock_" + time.Now().Format("20060102150405") + "_" + string(rune('0'+mockIDCounter%10))
}

func generateMockLicenseKey() string {
	return "MOCK-1234-5678-ABCD"
}
