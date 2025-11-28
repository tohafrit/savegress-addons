package audit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/pkg/models"
)

// Logger handles HIPAA-compliant audit logging
type Logger struct {
	config    *config.AuditConfig
	events    map[string]*models.AuditEvent
	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}
	eventCh   chan *models.AuditEvent
}

// NewLogger creates a new audit logger
func NewLogger(cfg *config.AuditConfig) *Logger {
	return &Logger{
		config:  cfg,
		events:  make(map[string]*models.AuditEvent),
		stopCh:  make(chan struct{}),
		eventCh: make(chan *models.AuditEvent, 1000),
	}
}

// Start starts the audit logger
func (l *Logger) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = true
	l.mu.Unlock()

	go l.processEvents(ctx)
	return nil
}

// Stop stops the audit logger
func (l *Logger) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		close(l.stopCh)
		l.running = false
	}
}

func (l *Logger) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopCh:
			return
		case event := <-l.eventCh:
			l.mu.Lock()
			l.events[event.ID] = event
			l.mu.Unlock()
		}
	}
}

// LogAccess logs a PHI access event
func (l *Logger) LogAccess(ctx context.Context, req *AccessLogRequest) *models.AuditEvent {
	if !l.config.Enabled {
		return nil
	}

	event := &models.AuditEvent{
		ID: uuid.New().String(),
		Type: &models.Coding{
			System:  "http://dicom.nema.org/resources/ontology/DCM",
			Code:    "110110",
			Display: "Patient Record",
		},
		Action:   req.Action,
		Recorded: time.Now(),
		Outcome:  req.Outcome,
		Agent: []models.AuditEventAgent{
			{
				Who: &models.Reference{
					Reference: req.UserID,
					Display:   req.UserName,
				},
				Name:      req.UserName,
				Requestor: true,
				Network: &models.AuditEventNetwork{
					Address: req.IPAddress,
					Type:    "2", // IP Address
				},
				PurposeOfUse: []models.CodeableConcept{
					{
						Coding: []models.Coding{
							{
								System:  "http://terminology.hl7.org/CodeSystem/v3-ActReason",
								Code:    req.Purpose,
								Display: req.Purpose,
							},
						},
					},
				},
			},
		},
		Source: &models.AuditEventSource{
			Site: "HealthSync",
			Observer: &models.Reference{
				Reference: "Device/healthsync-server",
				Display:   "HealthSync Server",
			},
			Type: []models.Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/security-source-type",
					Code:    "4",
					Display: "Application Server",
				},
			},
		},
		Entity: []models.AuditEventEntity{
			{
				What: &models.Reference{
					Reference: req.ResourceType + "/" + req.ResourceID,
				},
				Type: &models.Coding{
					System:  "http://terminology.hl7.org/CodeSystem/audit-entity-type",
					Code:    "1",
					Display: "Person",
				},
				Role: &models.Coding{
					System:  "http://terminology.hl7.org/CodeSystem/object-role",
					Code:    "1",
					Display: "Patient",
				},
				Name: req.PatientID,
			},
		},
	}

	if req.Query != "" {
		event.Entity[0].Query = req.Query
	}

	l.eventCh <- event
	return event
}

// AccessLogRequest contains parameters for access logging
type AccessLogRequest struct {
	UserID       string
	UserName     string
	UserRole     string
	IPAddress    string
	Action       string // C, R, U, D, E (Create, Read, Update, Delete, Execute)
	ResourceType string
	ResourceID   string
	PatientID    string
	Purpose      string
	Outcome      string // 0=success, 4=minor failure, 8=serious failure, 12=major failure
	Query        string
}

// LogSecurityEvent logs a security event
func (l *Logger) LogSecurityEvent(ctx context.Context, req *SecurityEventRequest) *models.AuditEvent {
	if !l.config.Enabled {
		return nil
	}

	event := &models.AuditEvent{
		ID: uuid.New().String(),
		Type: &models.Coding{
			System:  "http://dicom.nema.org/resources/ontology/DCM",
			Code:    req.EventCode,
			Display: req.EventDisplay,
		},
		Subtype: []models.Coding{
			{
				System:  "http://dicom.nema.org/resources/ontology/DCM",
				Code:    req.SubtypeCode,
				Display: req.SubtypeDisplay,
			},
		},
		Action:      req.Action,
		Recorded:    time.Now(),
		Outcome:     req.Outcome,
		OutcomeDesc: req.OutcomeDescription,
		Agent: []models.AuditEventAgent{
			{
				Who: &models.Reference{
					Reference: req.ActorID,
					Display:   req.ActorName,
				},
				Name:      req.ActorName,
				Requestor: true,
				Network: &models.AuditEventNetwork{
					Address: req.IPAddress,
					Type:    "2",
				},
			},
		},
		Source: &models.AuditEventSource{
			Site: "HealthSync",
			Observer: &models.Reference{
				Reference: "Device/healthsync-server",
			},
		},
	}

	l.eventCh <- event
	return event
}

// SecurityEventRequest contains parameters for security event logging
type SecurityEventRequest struct {
	EventCode          string
	EventDisplay       string
	SubtypeCode        string
	SubtypeDisplay     string
	Action             string
	ActorID            string
	ActorName          string
	IPAddress          string
	Outcome            string
	OutcomeDescription string
}

// LogLogin logs a login event
func (l *Logger) LogLogin(userID, userName, ipAddress string, success bool) *models.AuditEvent {
	outcome := "0"
	if !success {
		outcome = "8"
	}

	return l.LogSecurityEvent(context.Background(), &SecurityEventRequest{
		EventCode:    "110114",
		EventDisplay: "User Authentication",
		SubtypeCode:  "110122",
		SubtypeDisplay: "Login",
		Action:       "E",
		ActorID:      userID,
		ActorName:    userName,
		IPAddress:    ipAddress,
		Outcome:      outcome,
	})
}

// LogLogout logs a logout event
func (l *Logger) LogLogout(userID, userName, ipAddress string) *models.AuditEvent {
	return l.LogSecurityEvent(context.Background(), &SecurityEventRequest{
		EventCode:    "110114",
		EventDisplay: "User Authentication",
		SubtypeCode:  "110123",
		SubtypeDisplay: "Logout",
		Action:       "E",
		ActorID:      userID,
		ActorName:    userName,
		IPAddress:    ipAddress,
		Outcome:      "0",
	})
}

// LogExport logs a data export event
func (l *Logger) LogExport(ctx context.Context, req *ExportLogRequest) *models.AuditEvent {
	event := &models.AuditEvent{
		ID: uuid.New().String(),
		Type: &models.Coding{
			System:  "http://dicom.nema.org/resources/ontology/DCM",
			Code:    "110106",
			Display: "Export",
		},
		Action:   "R",
		Recorded: time.Now(),
		Outcome:  "0",
		Agent: []models.AuditEventAgent{
			{
				Who: &models.Reference{
					Reference: req.UserID,
					Display:   req.UserName,
				},
				Requestor: true,
				Network: &models.AuditEventNetwork{
					Address: req.IPAddress,
					Type:    "2",
				},
			},
		},
		Source: &models.AuditEventSource{
			Site: "HealthSync",
			Observer: &models.Reference{
				Reference: "Device/healthsync-server",
			},
		},
		Entity: []models.AuditEventEntity{
			{
				Name:        req.ExportType,
				Description: req.Description,
				Detail: []models.AuditEventDetail{
					{
						Type:        "RecordCount",
						ValueString: req.RecordCount,
					},
					{
						Type:        "Format",
						ValueString: req.Format,
					},
				},
			},
		},
	}

	l.eventCh <- event
	return event
}

// ExportLogRequest contains parameters for export logging
type ExportLogRequest struct {
	UserID      string
	UserName    string
	IPAddress   string
	ExportType  string
	Description string
	RecordCount string
	Format      string
}

// GetEvent retrieves an audit event by ID
func (l *Logger) GetEvent(id string) (*models.AuditEvent, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	event, ok := l.events[id]
	return event, ok
}

// GetEvents retrieves audit events with filters
func (l *Logger) GetEvents(filter EventFilter) []*models.AuditEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []*models.AuditEvent
	for _, event := range l.events {
		if l.matchesFilter(event, filter) {
			results = append(results, event)
		}
	}
	return results
}

// EventFilter defines filters for event queries
type EventFilter struct {
	Action    string
	Outcome   string
	UserID    string
	PatientID string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

func (l *Logger) matchesFilter(event *models.AuditEvent, filter EventFilter) bool {
	if filter.Action != "" && event.Action != filter.Action {
		return false
	}
	if filter.Outcome != "" && event.Outcome != filter.Outcome {
		return false
	}
	if filter.StartDate != nil && event.Recorded.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && event.Recorded.After(*filter.EndDate) {
		return false
	}
	if filter.UserID != "" {
		found := false
		for _, agent := range event.Agent {
			if agent.Who != nil && agent.Who.Reference == filter.UserID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// GetStats returns audit statistics
func (l *Logger) GetStats() *AuditStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := &AuditStats{
		ByAction:  make(map[string]int),
		ByOutcome: make(map[string]int),
		ByType:    make(map[string]int),
	}

	for _, event := range l.events {
		stats.TotalEvents++
		stats.ByAction[event.Action]++
		stats.ByOutcome[event.Outcome]++
		if event.Type != nil {
			stats.ByType[event.Type.Code]++
		}

		if event.Outcome != "0" {
			stats.FailedEvents++
		}
	}

	return stats
}

// AuditStats contains audit statistics
type AuditStats struct {
	TotalEvents  int            `json:"total_events"`
	FailedEvents int            `json:"failed_events"`
	ByAction     map[string]int `json:"by_action"`
	ByOutcome    map[string]int `json:"by_outcome"`
	ByType       map[string]int `json:"by_type"`
}
