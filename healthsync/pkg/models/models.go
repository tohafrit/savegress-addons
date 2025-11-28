package models

import (
	"time"
)

// ResourceType represents FHIR resource types
type ResourceType string

const (
	ResourceTypePatient          ResourceType = "Patient"
	ResourceTypePractitioner     ResourceType = "Practitioner"
	ResourceTypeOrganization     ResourceType = "Organization"
	ResourceTypeEncounter        ResourceType = "Encounter"
	ResourceTypeObservation      ResourceType = "Observation"
	ResourceTypeCondition        ResourceType = "Condition"
	ResourceTypeMedication       ResourceType = "Medication"
	ResourceTypeMedicationRequest ResourceType = "MedicationRequest"
	ResourceTypeProcedure        ResourceType = "Procedure"
	ResourceTypeDiagnosticReport ResourceType = "DiagnosticReport"
	ResourceTypeImmunization     ResourceType = "Immunization"
	ResourceTypeAllergyIntolerance ResourceType = "AllergyIntolerance"
	ResourceTypeDocumentReference ResourceType = "DocumentReference"
)

// FHIRResource represents a base FHIR resource
type FHIRResource struct {
	ResourceType ResourceType           `json:"resourceType"`
	ID           string                 `json:"id"`
	Meta         *ResourceMeta          `json:"meta,omitempty"`
	Text         *Narrative             `json:"text,omitempty"`
	Extension    []Extension            `json:"extension,omitempty"`
	Identifier   []Identifier           `json:"identifier,omitempty"`
}

// ResourceMeta contains metadata about a resource
type ResourceMeta struct {
	VersionID   string    `json:"versionId,omitempty"`
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
	Source      string    `json:"source,omitempty"`
	Profile     []string  `json:"profile,omitempty"`
	Security    []Coding  `json:"security,omitempty"`
	Tag         []Coding  `json:"tag,omitempty"`
}

// Narrative contains human-readable summary
type Narrative struct {
	Status string `json:"status"`
	Div    string `json:"div"`
}

// Extension represents FHIR extensions
type Extension struct {
	URL   string      `json:"url"`
	Value interface{} `json:"value,omitempty"`
}

// Identifier represents a business identifier
type Identifier struct {
	Use      string   `json:"use,omitempty"`
	Type     *CodeableConcept `json:"type,omitempty"`
	System   string   `json:"system,omitempty"`
	Value    string   `json:"value,omitempty"`
	Period   *Period  `json:"period,omitempty"`
	Assigner *Reference `json:"assigner,omitempty"`
}

// Coding represents a code from a code system
type Coding struct {
	System       string `json:"system,omitempty"`
	Version      string `json:"version,omitempty"`
	Code         string `json:"code,omitempty"`
	Display      string `json:"display,omitempty"`
	UserSelected bool   `json:"userSelected,omitempty"`
}

// CodeableConcept represents a concept with coding and text
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Reference represents a reference to another resource
type Reference struct {
	Reference  string     `json:"reference,omitempty"`
	Type       string     `json:"type,omitempty"`
	Identifier *Identifier `json:"identifier,omitempty"`
	Display    string     `json:"display,omitempty"`
}

// Period represents a time period
type Period struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// HumanName represents a person's name
type HumanName struct {
	Use    string   `json:"use,omitempty"`
	Text   string   `json:"text,omitempty"`
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
	Prefix []string `json:"prefix,omitempty"`
	Suffix []string `json:"suffix,omitempty"`
	Period *Period  `json:"period,omitempty"`
}

// Address represents a physical address
type Address struct {
	Use        string   `json:"use,omitempty"`
	Type       string   `json:"type,omitempty"`
	Text       string   `json:"text,omitempty"`
	Line       []string `json:"line,omitempty"`
	City       string   `json:"city,omitempty"`
	District   string   `json:"district,omitempty"`
	State      string   `json:"state,omitempty"`
	PostalCode string   `json:"postalCode,omitempty"`
	Country    string   `json:"country,omitempty"`
	Period     *Period  `json:"period,omitempty"`
}

// ContactPoint represents contact information
type ContactPoint struct {
	System string  `json:"system,omitempty"`
	Value  string  `json:"value,omitempty"`
	Use    string  `json:"use,omitempty"`
	Rank   int     `json:"rank,omitempty"`
	Period *Period `json:"period,omitempty"`
}

// Patient represents a FHIR Patient resource
type Patient struct {
	FHIRResource
	Active           bool            `json:"active,omitempty"`
	Name             []HumanName     `json:"name,omitempty"`
	Telecom          []ContactPoint  `json:"telecom,omitempty"`
	Gender           string          `json:"gender,omitempty"`
	BirthDate        string          `json:"birthDate,omitempty"`
	DeceasedBoolean  *bool           `json:"deceasedBoolean,omitempty"`
	DeceasedDateTime *time.Time      `json:"deceasedDateTime,omitempty"`
	Address          []Address       `json:"address,omitempty"`
	MaritalStatus    *CodeableConcept `json:"maritalStatus,omitempty"`
	Contact          []PatientContact `json:"contact,omitempty"`
	Communication    []PatientCommunication `json:"communication,omitempty"`
	GeneralPractitioner []Reference  `json:"generalPractitioner,omitempty"`
	ManagingOrganization *Reference  `json:"managingOrganization,omitempty"`
}

// PatientContact represents a contact person for a patient
type PatientContact struct {
	Relationship []CodeableConcept `json:"relationship,omitempty"`
	Name         *HumanName        `json:"name,omitempty"`
	Telecom      []ContactPoint    `json:"telecom,omitempty"`
	Address      *Address          `json:"address,omitempty"`
	Gender       string            `json:"gender,omitempty"`
	Organization *Reference        `json:"organization,omitempty"`
	Period       *Period           `json:"period,omitempty"`
}

// PatientCommunication represents language and preference
type PatientCommunication struct {
	Language  *CodeableConcept `json:"language"`
	Preferred bool             `json:"preferred,omitempty"`
}

// Observation represents a FHIR Observation resource
type Observation struct {
	FHIRResource
	Status          string           `json:"status"`
	Category        []CodeableConcept `json:"category,omitempty"`
	Code            *CodeableConcept `json:"code"`
	Subject         *Reference       `json:"subject,omitempty"`
	Encounter       *Reference       `json:"encounter,omitempty"`
	EffectiveDateTime *time.Time     `json:"effectiveDateTime,omitempty"`
	Issued          *time.Time       `json:"issued,omitempty"`
	Performer       []Reference      `json:"performer,omitempty"`
	ValueQuantity   *Quantity        `json:"valueQuantity,omitempty"`
	ValueString     string           `json:"valueString,omitempty"`
	ValueBoolean    *bool            `json:"valueBoolean,omitempty"`
	ValueCodeableConcept *CodeableConcept `json:"valueCodeableConcept,omitempty"`
	Interpretation  []CodeableConcept `json:"interpretation,omitempty"`
	Note            []Annotation     `json:"note,omitempty"`
	ReferenceRange  []ObservationReferenceRange `json:"referenceRange,omitempty"`
}

// Quantity represents a measured amount
type Quantity struct {
	Value      float64 `json:"value,omitempty"`
	Comparator string  `json:"comparator,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	System     string  `json:"system,omitempty"`
	Code       string  `json:"code,omitempty"`
}

// Annotation represents a text note
type Annotation struct {
	AuthorReference *Reference `json:"authorReference,omitempty"`
	AuthorString    string     `json:"authorString,omitempty"`
	Time            *time.Time `json:"time,omitempty"`
	Text            string     `json:"text"`
}

// ObservationReferenceRange represents reference range for observation
type ObservationReferenceRange struct {
	Low         *Quantity         `json:"low,omitempty"`
	High        *Quantity         `json:"high,omitempty"`
	Type        *CodeableConcept  `json:"type,omitempty"`
	AppliesTo   []CodeableConcept `json:"appliesTo,omitempty"`
	Age         *Range            `json:"age,omitempty"`
	Text        string            `json:"text,omitempty"`
}

// Range represents a range of values
type Range struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
}

// Encounter represents a FHIR Encounter resource
type Encounter struct {
	FHIRResource
	Status           string            `json:"status"`
	Class            *Coding           `json:"class"`
	Type             []CodeableConcept `json:"type,omitempty"`
	ServiceType      *CodeableConcept  `json:"serviceType,omitempty"`
	Priority         *CodeableConcept  `json:"priority,omitempty"`
	Subject          *Reference        `json:"subject,omitempty"`
	Participant      []EncounterParticipant `json:"participant,omitempty"`
	Period           *Period           `json:"period,omitempty"`
	ReasonCode       []CodeableConcept `json:"reasonCode,omitempty"`
	Diagnosis        []EncounterDiagnosis `json:"diagnosis,omitempty"`
	ServiceProvider  *Reference        `json:"serviceProvider,omitempty"`
}

// EncounterParticipant represents a participant in an encounter
type EncounterParticipant struct {
	Type       []CodeableConcept `json:"type,omitempty"`
	Period     *Period           `json:"period,omitempty"`
	Individual *Reference        `json:"individual,omitempty"`
}

// EncounterDiagnosis represents a diagnosis in an encounter
type EncounterDiagnosis struct {
	Condition *Reference       `json:"condition"`
	Use       *CodeableConcept `json:"use,omitempty"`
	Rank      int              `json:"rank,omitempty"`
}

// Consent represents patient consent for data sharing
type Consent struct {
	FHIRResource
	Status       string            `json:"status"`
	Scope        *CodeableConcept  `json:"scope"`
	Category     []CodeableConcept `json:"category"`
	Patient      *Reference        `json:"patient,omitempty"`
	DateTime     *time.Time        `json:"dateTime,omitempty"`
	Performer    []Reference       `json:"performer,omitempty"`
	Organization []Reference       `json:"organization,omitempty"`
	PolicyRule   *CodeableConcept  `json:"policyRule,omitempty"`
	Provision    *ConsentProvision `json:"provision,omitempty"`
}

// ConsentProvision represents consent rules
type ConsentProvision struct {
	Type       string               `json:"type,omitempty"`
	Period     *Period              `json:"period,omitempty"`
	Actor      []ConsentActor       `json:"actor,omitempty"`
	Action     []CodeableConcept    `json:"action,omitempty"`
	SecurityLabel []Coding          `json:"securityLabel,omitempty"`
	Purpose    []Coding             `json:"purpose,omitempty"`
	Class      []Coding             `json:"class,omitempty"`
	Code       []CodeableConcept    `json:"code,omitempty"`
	DataPeriod *Period              `json:"dataPeriod,omitempty"`
	Data       []ConsentData        `json:"data,omitempty"`
	Provision  []ConsentProvision   `json:"provision,omitempty"`
}

// ConsentActor represents who the consent applies to
type ConsentActor struct {
	Role      *CodeableConcept `json:"role"`
	Reference *Reference       `json:"reference"`
}

// ConsentData represents data covered by consent
type ConsentData struct {
	Meaning   string     `json:"meaning"`
	Reference *Reference `json:"reference"`
}

// AuditEvent represents a HIPAA audit event
type AuditEvent struct {
	ID              string            `json:"id"`
	Type            *Coding           `json:"type"`
	Subtype         []Coding          `json:"subtype,omitempty"`
	Action          string            `json:"action"`
	Period          *Period           `json:"period,omitempty"`
	Recorded        time.Time         `json:"recorded"`
	Outcome         string            `json:"outcome"`
	OutcomeDesc     string            `json:"outcomeDesc,omitempty"`
	PurposeOfEvent  []CodeableConcept `json:"purposeOfEvent,omitempty"`
	Agent           []AuditEventAgent `json:"agent"`
	Source          *AuditEventSource `json:"source"`
	Entity          []AuditEventEntity `json:"entity,omitempty"`
}

// AuditEventAgent represents who performed the action
type AuditEventAgent struct {
	Type        *CodeableConcept `json:"type,omitempty"`
	Role        []CodeableConcept `json:"role,omitempty"`
	Who         *Reference       `json:"who,omitempty"`
	AltID       string           `json:"altId,omitempty"`
	Name        string           `json:"name,omitempty"`
	Requestor   bool             `json:"requestor"`
	Location    *Reference       `json:"location,omitempty"`
	Policy      []string         `json:"policy,omitempty"`
	Network     *AuditEventNetwork `json:"network,omitempty"`
	PurposeOfUse []CodeableConcept `json:"purposeOfUse,omitempty"`
}

// AuditEventNetwork represents network details
type AuditEventNetwork struct {
	Address string `json:"address,omitempty"`
	Type    string `json:"type,omitempty"`
}

// AuditEventSource represents the audit event source
type AuditEventSource struct {
	Site     string    `json:"site,omitempty"`
	Observer *Reference `json:"observer"`
	Type     []Coding  `json:"type,omitempty"`
}

// AuditEventEntity represents what was accessed
type AuditEventEntity struct {
	What        *Reference       `json:"what,omitempty"`
	Type        *Coding          `json:"type,omitempty"`
	Role        *Coding          `json:"role,omitempty"`
	Lifecycle   *Coding          `json:"lifecycle,omitempty"`
	SecurityLabel []Coding       `json:"securityLabel,omitempty"`
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Query       string           `json:"query,omitempty"`
	Detail      []AuditEventDetail `json:"detail,omitempty"`
}

// AuditEventDetail represents additional details
type AuditEventDetail struct {
	Type        string `json:"type"`
	ValueString string `json:"valueString,omitempty"`
	ValueBase64Binary string `json:"valueBase64Binary,omitempty"`
}

// PHIField represents a field containing Protected Health Information
type PHIField struct {
	FieldName   string `json:"field_name"`
	FieldPath   string `json:"field_path"`
	PHICategory string `json:"phi_category"`
	Sensitivity string `json:"sensitivity"`
}

// PHI Categories per HIPAA Safe Harbor
const (
	PHICategoryName          = "name"
	PHICategoryAddress       = "address"
	PHICategoryDates         = "dates"
	PHICategoryPhone         = "phone"
	PHICategoryFax           = "fax"
	PHICategoryEmail         = "email"
	PHICategorySSN           = "ssn"
	PHICategoryMRN           = "medical_record_number"
	PHICategoryHealthPlan    = "health_plan_beneficiary"
	PHICategoryAccount       = "account_number"
	PHICategoryCertificate   = "certificate_license"
	PHICategoryVehicle       = "vehicle_identifier"
	PHICategoryDevice        = "device_identifier"
	PHICategoryURL           = "web_url"
	PHICategoryIP            = "ip_address"
	PHICategoryBiometric     = "biometric"
	PHICategoryPhoto         = "photo"
	PHICategoryOther         = "other_unique"
)

// ComplianceViolation represents a HIPAA compliance violation
type ComplianceViolation struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	Field        string    `json:"field"`
	Description  string    `json:"description"`
	Remediation  string    `json:"remediation"`
	DetectedAt   time.Time `json:"detected_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	Status       string    `json:"status"`
}

// AccessRequest represents a request to access PHI
type AccessRequest struct {
	ID            string    `json:"id"`
	RequestorID   string    `json:"requestor_id"`
	RequestorType string    `json:"requestor_type"`
	PatientID     string    `json:"patient_id"`
	ResourceType  string    `json:"resource_type"`
	Purpose       string    `json:"purpose"`
	Status        string    `json:"status"`
	RequestedAt   time.Time `json:"requested_at"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	ApprovedBy    string    `json:"approved_by,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}
