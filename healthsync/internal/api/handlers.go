package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/healthsync/internal/anonymization"
	"github.com/savegress/healthsync/internal/audit"
	"github.com/savegress/healthsync/internal/compliance"
	"github.com/savegress/healthsync/internal/consent"
	"github.com/savegress/healthsync/pkg/models"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	compliance    *compliance.Engine
	audit         *audit.Logger
	anonymization *anonymization.Engine
	consent       *consent.Manager
	// In-memory storage for demo (would use database in production)
	patients     map[string]*models.Patient
	observations map[string]*models.Observation
	encounters   map[string]*models.Encounter
}

// NewHandlers creates new handlers
func NewHandlers(comp *compliance.Engine, auditLog *audit.Logger, anon *anonymization.Engine, consentMgr *consent.Manager) *Handlers {
	return &Handlers{
		compliance:    comp,
		audit:         auditLog,
		anonymization: anon,
		consent:       consentMgr,
		patients:      make(map[string]*models.Patient),
		observations:  make(map[string]*models.Observation),
		encounters:    make(map[string]*models.Encounter),
	}
}

// HealthCheck handles health check requests
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "healthsync",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// FHIR Patient handlers

// SearchPatients searches patients
func (h *Handlers) SearchPatients(w http.ResponseWriter, r *http.Request) {
	var results []*models.Patient
	for _, p := range h.patients {
		results = append(results, p)
	}
	respond(w, http.StatusOK, results)
}

// CreatePatient creates a patient
func (h *Handlers) CreatePatient(w http.ResponseWriter, r *http.Request) {
	var patient models.Patient
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if patient.ID == "" {
		patient.ID = generateID("patient")
	}
	patient.ResourceType = models.ResourceTypePatient

	// Validate compliance
	result := h.compliance.ValidateResource(&patient, models.ResourceTypePatient)
	if !result.Valid {
		respond(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "Compliance validation failed",
			"violations": result.Violations,
		})
		return
	}

	h.patients[patient.ID] = &patient

	// Log access
	h.audit.LogAccess(r.Context(), &audit.AccessLogRequest{
		UserID:       r.Header.Get("X-User-ID"),
		UserName:     r.Header.Get("X-User-Name"),
		IPAddress:    r.RemoteAddr,
		Action:       "C",
		ResourceType: "Patient",
		ResourceID:   patient.ID,
		PatientID:    patient.ID,
		Purpose:      r.Header.Get("X-Purpose"),
		Outcome:      "0",
	})

	respond(w, http.StatusCreated, patient)
}

// GetPatient gets a patient by ID
func (h *Handlers) GetPatient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	patient, ok := h.patients[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Patient not found")
		return
	}

	// Check consent
	accessResult := h.consent.CheckAccess(&consent.AccessCheckRequest{
		PatientID:    id,
		ResourceType: "Patient",
		ResourceID:   id,
		RequestorID:  r.Header.Get("X-User-ID"),
		Purpose:      r.Header.Get("X-Purpose"),
		Action:       "read",
	})

	if !accessResult.Allowed {
		h.audit.LogAccess(r.Context(), &audit.AccessLogRequest{
			UserID:       r.Header.Get("X-User-ID"),
			IPAddress:    r.RemoteAddr,
			Action:       "R",
			ResourceType: "Patient",
			ResourceID:   id,
			PatientID:    id,
			Purpose:      r.Header.Get("X-Purpose"),
			Outcome:      "8", // Serious failure
		})

		respondError(w, http.StatusForbidden, "Access denied: "+accessResult.Reason)
		return
	}

	// Log successful access
	h.audit.LogAccess(r.Context(), &audit.AccessLogRequest{
		UserID:       r.Header.Get("X-User-ID"),
		UserName:     r.Header.Get("X-User-Name"),
		IPAddress:    r.RemoteAddr,
		Action:       "R",
		ResourceType: "Patient",
		ResourceID:   id,
		PatientID:    id,
		Purpose:      r.Header.Get("X-Purpose"),
		Outcome:      "0",
	})

	respond(w, http.StatusOK, patient)
}

// UpdatePatient updates a patient
func (h *Handlers) UpdatePatient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, ok := h.patients[id]; !ok {
		respondError(w, http.StatusNotFound, "Patient not found")
		return
	}

	var patient models.Patient
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	patient.ID = id
	patient.ResourceType = models.ResourceTypePatient

	// Validate compliance
	result := h.compliance.ValidateResource(&patient, models.ResourceTypePatient)
	if !result.Valid {
		respond(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "Compliance validation failed",
			"violations": result.Violations,
		})
		return
	}

	h.patients[id] = &patient

	// Log access
	h.audit.LogAccess(r.Context(), &audit.AccessLogRequest{
		UserID:       r.Header.Get("X-User-ID"),
		IPAddress:    r.RemoteAddr,
		Action:       "U",
		ResourceType: "Patient",
		ResourceID:   id,
		PatientID:    id,
		Purpose:      r.Header.Get("X-Purpose"),
		Outcome:      "0",
	})

	respond(w, http.StatusOK, patient)
}

// DeletePatient deletes a patient
func (h *Handlers) DeletePatient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, ok := h.patients[id]; !ok {
		respondError(w, http.StatusNotFound, "Patient not found")
		return
	}

	delete(h.patients, id)

	// Log access
	h.audit.LogAccess(r.Context(), &audit.AccessLogRequest{
		UserID:       r.Header.Get("X-User-ID"),
		IPAddress:    r.RemoteAddr,
		Action:       "D",
		ResourceType: "Patient",
		ResourceID:   id,
		PatientID:    id,
		Purpose:      r.Header.Get("X-Purpose"),
		Outcome:      "0",
	})

	w.WriteHeader(http.StatusNoContent)
}

// Observation handlers

// SearchObservations searches observations
func (h *Handlers) SearchObservations(w http.ResponseWriter, r *http.Request) {
	var results []*models.Observation
	for _, o := range h.observations {
		results = append(results, o)
	}
	respond(w, http.StatusOK, results)
}

// CreateObservation creates an observation
func (h *Handlers) CreateObservation(w http.ResponseWriter, r *http.Request) {
	var obs models.Observation
	if err := json.NewDecoder(r.Body).Decode(&obs); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if obs.ID == "" {
		obs.ID = generateID("obs")
	}
	obs.ResourceType = models.ResourceTypeObservation

	h.observations[obs.ID] = &obs
	respond(w, http.StatusCreated, obs)
}

// GetObservation gets an observation by ID
func (h *Handlers) GetObservation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	obs, ok := h.observations[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Observation not found")
		return
	}

	respond(w, http.StatusOK, obs)
}

// Encounter handlers

// SearchEncounters searches encounters
func (h *Handlers) SearchEncounters(w http.ResponseWriter, r *http.Request) {
	var results []*models.Encounter
	for _, e := range h.encounters {
		results = append(results, e)
	}
	respond(w, http.StatusOK, results)
}

// CreateEncounter creates an encounter
func (h *Handlers) CreateEncounter(w http.ResponseWriter, r *http.Request) {
	var enc models.Encounter
	if err := json.NewDecoder(r.Body).Decode(&enc); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if enc.ID == "" {
		enc.ID = generateID("enc")
	}
	enc.ResourceType = models.ResourceTypeEncounter

	h.encounters[enc.ID] = &enc
	respond(w, http.StatusCreated, enc)
}

// GetEncounter gets an encounter by ID
func (h *Handlers) GetEncounter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	enc, ok := h.encounters[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Encounter not found")
		return
	}

	respond(w, http.StatusOK, enc)
}

// Compliance handlers

// ListViolations lists compliance violations
func (h *Handlers) ListViolations(w http.ResponseWriter, r *http.Request) {
	filter := compliance.ViolationFilter{}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}
	violations := h.compliance.GetViolations(filter)
	respond(w, http.StatusOK, violations)
}

// GetViolation gets a violation by ID
func (h *Handlers) GetViolation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	violation, ok := h.compliance.GetViolation(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Violation not found")
		return
	}

	respond(w, http.StatusOK, violation)
}

// ResolveViolation resolves a violation
func (h *Handlers) ResolveViolation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.compliance.ResolveViolation(id, req.Resolution); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// ValidateResource validates a resource for compliance
func (h *Handlers) ValidateResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string          `json:"resource_type"`
		Resource     json.RawMessage `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var resource interface{}
	if err := json.Unmarshal(req.Resource, &resource); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid resource")
		return
	}

	result := h.compliance.ValidateResource(resource, models.ResourceType(req.ResourceType))
	respond(w, http.StatusOK, result)
}

// ScanForPHI scans a resource for PHI
func (h *Handlers) ScanForPHI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string          `json:"resource_type"`
		Resource     json.RawMessage `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var resource interface{}
	if err := json.Unmarshal(req.Resource, &resource); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid resource")
		return
	}

	result := h.compliance.ScanResourceForPHI(resource, models.ResourceType(req.ResourceType))
	respond(w, http.StatusOK, result)
}

// CheckMinimumNecessary checks minimum necessary principle
func (h *Handlers) CheckMinimumNecessary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType    string   `json:"resource_type"`
		RequestedFields []string `json:"requested_fields"`
		Purpose         string   `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result := h.compliance.CheckMinimumNecessary(req.RequestedFields, req.Purpose, models.ResourceType(req.ResourceType))
	respond(w, http.StatusOK, result)
}

// GetComplianceStats gets compliance statistics
func (h *Handlers) GetComplianceStats(w http.ResponseWriter, r *http.Request) {
	stats := h.compliance.GetStats()
	respond(w, http.StatusOK, stats)
}

// Audit handlers

// ListAuditEvents lists audit events
func (h *Handlers) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	filter := audit.EventFilter{}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	events := h.audit.GetEvents(filter)
	respond(w, http.StatusOK, events)
}

// GetAuditEvent gets an audit event by ID
func (h *Handlers) GetAuditEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	event, ok := h.audit.GetEvent(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Audit event not found")
		return
	}

	respond(w, http.StatusOK, event)
}

// GetAuditStats gets audit statistics
func (h *Handlers) GetAuditStats(w http.ResponseWriter, r *http.Request) {
	stats := h.audit.GetStats()
	respond(w, http.StatusOK, stats)
}

// Consent handlers

// ListConsents lists consents
func (h *Handlers) ListConsents(w http.ResponseWriter, r *http.Request) {
	patientID := r.URL.Query().Get("patient")
	if patientID != "" {
		consents := h.consent.GetPatientConsents(patientID)
		respond(w, http.StatusOK, consents)
		return
	}
	respond(w, http.StatusOK, []models.Consent{})
}

// CreateConsent creates a consent
func (h *Handlers) CreateConsent(w http.ResponseWriter, r *http.Request) {
	var consentRecord models.Consent
	if err := json.NewDecoder(r.Body).Decode(&consentRecord); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.consent.CreateConsent(&consentRecord); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, consentRecord)
}

// GetConsent gets a consent by ID
func (h *Handlers) GetConsent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	consentRecord, ok := h.consent.GetConsent(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Consent not found")
		return
	}

	respond(w, http.StatusOK, consentRecord)
}

// RevokeConsent revokes a consent
func (h *Handlers) RevokeConsent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.consent.RevokeConsent(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// CheckAccess checks consent-based access
func (h *Handlers) CheckAccess(w http.ResponseWriter, r *http.Request) {
	var req consent.AccessCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result := h.consent.CheckAccess(&req)
	respond(w, http.StatusOK, result)
}

// GetConsentStats gets consent statistics
func (h *Handlers) GetConsentStats(w http.ResponseWriter, r *http.Request) {
	stats := h.consent.GetStats()
	respond(w, http.StatusOK, stats)
}

// Access Request handlers

// ListAccessRequests lists access requests
func (h *Handlers) ListAccessRequests(w http.ResponseWriter, r *http.Request) {
	requests := h.consent.GetPendingRequests()
	respond(w, http.StatusOK, requests)
}

// CreateAccessRequest creates an access request
func (h *Handlers) CreateAccessRequest(w http.ResponseWriter, r *http.Request) {
	var req models.AccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.consent.CreateAccessRequest(r.Context(), &req); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, req)
}

// GetAccessRequest gets an access request by ID
func (h *Handlers) GetAccessRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req, ok := h.consent.GetAccessRequest(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Access request not found")
		return
	}

	respond(w, http.StatusOK, req)
}

// ApproveAccessRequest approves an access request
func (h *Handlers) ApproveAccessRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		ApproverID     string `json:"approver_id"`
		ExpirationDays int    `json:"expiration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.consent.ApproveAccessRequest(id, req.ApproverID, req.ExpirationDays); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "approved"})
}

// DenyAccessRequest denies an access request
func (h *Handlers) DenyAccessRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.consent.DenyAccessRequest(id, req.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "denied"})
}

// Anonymization handlers

// AnonymizePatient anonymizes a patient
func (h *Handlers) AnonymizePatient(w http.ResponseWriter, r *http.Request) {
	var patient models.Patient
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	anonymized := h.anonymization.AnonymizePatient(&patient)
	respond(w, http.StatusOK, anonymized)
}

// AnonymizeResource anonymizes a generic resource
func (h *Handlers) AnonymizeResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string          `json:"resource_type"`
		PatientID    string          `json:"patient_id"`
		Resource     json.RawMessage `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	anonymized, err := h.anonymization.AnonymizeJSON(req.Resource, models.ResourceType(req.ResourceType), req.PatientID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var result interface{}
	json.Unmarshal(anonymized, &result)
	respond(w, http.StatusOK, result)
}

// RedactText redacts PHI from text
func (h *Handlers) RedactText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	redacted := h.anonymization.RedactText(req.Text)
	respond(w, http.StatusOK, map[string]string{"redacted_text": redacted})
}

// CheckKAnonymity checks k-anonymity
func (h *Handlers) CheckKAnonymity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Records          []map[string]interface{} `json:"records"`
		QuasiIdentifiers []string                 `json:"quasi_identifiers"`
		K                int                      `json:"k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result := h.anonymization.CheckKAnonymity(req.Records, req.QuasiIdentifiers, req.K)
	respond(w, http.StatusOK, result)
}

// GetOverallStats gets overall system statistics
func (h *Handlers) GetOverallStats(w http.ResponseWriter, r *http.Request) {
	complianceStats := h.compliance.GetStats()
	auditStats := h.audit.GetStats()
	consentStats := h.consent.GetStats()

	respond(w, http.StatusOK, map[string]interface{}{
		"compliance": complianceStats,
		"audit":      auditStats,
		"consent":    consentStats,
		"resources": map[string]int{
			"patients":     len(h.patients),
			"observations": len(h.observations),
			"encounters":   len(h.encounters),
		},
	})
}

// Helper functions

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405")
}
