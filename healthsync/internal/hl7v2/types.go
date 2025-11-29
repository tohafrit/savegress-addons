package hl7v2

import (
	"time"
)

// MessageType represents HL7 v2.x message types
type MessageType string

const (
	// ADT - Admit, Discharge, Transfer messages
	MessageTypeADT MessageType = "ADT"
	// ORM - Order messages
	MessageTypeORM MessageType = "ORM"
	// ORU - Observation Result messages
	MessageTypeORU MessageType = "ORU"
	// ACK - Acknowledgment
	MessageTypeACK MessageType = "ACK"
	// MDM - Medical Document Management
	MessageTypeMDM MessageType = "MDM"
	// SIU - Scheduling Information Unsolicited
	MessageTypeSIU MessageType = "SIU"
	// RDE - Pharmacy/Treatment Encoded Order
	MessageTypeRDE MessageType = "RDE"
)

// TriggerEvent represents HL7 v2.x trigger events
type TriggerEvent string

// ADT Trigger Events
const (
	TriggerA01 TriggerEvent = "A01" // Admit/Visit Notification
	TriggerA02 TriggerEvent = "A02" // Transfer a Patient
	TriggerA03 TriggerEvent = "A03" // Discharge/End Visit
	TriggerA04 TriggerEvent = "A04" // Register a Patient
	TriggerA05 TriggerEvent = "A05" // Pre-admit a Patient
	TriggerA06 TriggerEvent = "A06" // Change an Outpatient to Inpatient
	TriggerA07 TriggerEvent = "A07" // Change an Inpatient to Outpatient
	TriggerA08 TriggerEvent = "A08" // Update Patient Information
	TriggerA11 TriggerEvent = "A11" // Cancel Admit
	TriggerA12 TriggerEvent = "A12" // Cancel Transfer
	TriggerA13 TriggerEvent = "A13" // Cancel Discharge
	TriggerA28 TriggerEvent = "A28" // Add Person Information
	TriggerA31 TriggerEvent = "A31" // Update Person Information
	TriggerA40 TriggerEvent = "A40" // Merge Patient
)

// ORM Trigger Events
const (
	TriggerO01 TriggerEvent = "O01" // Order Message
	TriggerO02 TriggerEvent = "O02" // Order Response
)

// ORU Trigger Events
const (
	TriggerR01 TriggerEvent = "R01" // Unsolicited Observation Result
	TriggerR03 TriggerEvent = "R03" // Display-oriented Results
	TriggerR30 TriggerEvent = "R30" // Unsolicited Point-of-Care Observation
	TriggerR31 TriggerEvent = "R31" // Unsolicited New Point-of-Care Observation
)

// Segment represents an HL7 segment
type Segment interface {
	ID() string
	Encode() (string, error)
	Decode(data string) error
}

// Message represents an HL7 v2.x message
type Message struct {
	Type         MessageType
	TriggerEvent TriggerEvent
	Version      string
	ControlID    string
	Timestamp    time.Time
	SendingApp   string
	SendingFac   string
	ReceivingApp string
	ReceivingFac string
	Segments     []Segment
	RawData      []byte
}

// MSH - Message Header Segment
type MSH struct {
	FieldSeparator        string
	EncodingCharacters    string
	SendingApplication    string
	SendingFacility       string
	ReceivingApplication  string
	ReceivingFacility     string
	DateTime              time.Time
	Security              string
	MessageType           string
	MessageControlID      string
	ProcessingID          string
	VersionID             string
	SequenceNumber        string
	ContinuationPointer   string
	AcceptAckType         string
	ApplicationAckType    string
	CountryCode           string
	CharacterSet          string
}

func (m *MSH) ID() string { return "MSH" }

// PID - Patient Identification Segment
type PID struct {
	SetID                    string
	PatientID                string
	PatientIdentifierList    []PatientIdentifier
	AlternatePatientID       string
	PatientName              []PersonName
	MothersMaidenName        string
	DateOfBirth              time.Time
	AdministrativeSex        string
	PatientAlias             string
	Race                     string
	PatientAddress           []Address
	CountyCode               string
	PhoneNumberHome          string
	PhoneNumberBusiness      string
	PrimaryLanguage          string
	MaritalStatus            string
	Religion                 string
	PatientAccountNumber     string
	SSNNumber                string
	DriversLicenseNumber     string
	MothersIdentifier        string
	EthnicGroup              string
	BirthPlace               string
	MultipleBirthIndicator   string
	BirthOrder               string
	Citizenship              string
	VeteransMilitaryStatus   string
	Nationality              string
	PatientDeathDateTime     *time.Time
	PatientDeathIndicator    string
}

func (p *PID) ID() string { return "PID" }

// PatientIdentifier represents a patient identifier
type PatientIdentifier struct {
	ID               string
	CheckDigit       string
	CheckDigitScheme string
	AssigningAuth    string
	IdentifierType   string
	AssigningFac     string
}

// PersonName represents a person's name
type PersonName struct {
	FamilyName   string
	GivenName    string
	MiddleName   string
	Suffix       string
	Prefix       string
	Degree       string
	NameTypeCode string
}

// Address represents an address
type Address struct {
	StreetAddress string
	OtherDesig    string
	City          string
	State         string
	ZipCode       string
	Country       string
	AddressType   string
}

// PV1 - Patient Visit Segment
type PV1 struct {
	SetID                string
	PatientClass         string // I=Inpatient, O=Outpatient, E=Emergency, P=Preadmit
	AssignedPatientLoc   Location
	AdmissionType        string
	PreadmitNumber       string
	PriorPatientLoc      Location
	AttendingDoctor      Provider
	ReferringDoctor      Provider
	ConsultingDoctor     []Provider
	HospitalService      string
	TempLocation         Location
	PreadmitTestInd      string
	ReadmissionIndicator string
	AdmitSource          string
	AmbulatoryStatus     string
	VIPIndicator         string
	AdmittingDoctor      Provider
	PatientType          string
	VisitNumber          string
	FinancialClass       string
	ChargePriceIndicator string
	CourtesyCode         string
	CreditRating         string
	ContractCode         string
	ContractEffectiveDate time.Time
	ContractAmount       float64
	ContractPeriod       int
	InterestCode         string
	TransferToBadDebt    string
	TransferToBadDebtDate *time.Time
	BadDebtAgencyCode    string
	BadDebtTransferAmt   float64
	BadDebtRecoveryAmt   float64
	DeleteAccountInd     string
	DeleteAccountDate    *time.Time
	DischargeDisposition string
	DischargedToLocation string
	DietType             string
	ServicingFacility    string
	BedStatus            string
	AccountStatus        string
	PendingLocation      Location
	PriorTempLocation    Location
	AdmitDateTime        time.Time
	DischargeDateTime    *time.Time
	CurrentPatientBal    float64
	TotalCharges         float64
	TotalAdjustments     float64
	TotalPayments        float64
}

func (p *PV1) ID() string { return "PV1" }

// Location represents a patient location
type Location struct {
	PointOfCare string
	Room        string
	Bed         string
	Facility    string
	LocationSt  string
	Building    string
	Floor       string
}

// Provider represents a healthcare provider
type Provider struct {
	ID          string
	FamilyName  string
	GivenName   string
	MiddleName  string
	Suffix      string
	Prefix      string
	Degree      string
	SourceTable string
	AssignAuth  string
	NameTypeCode string
}

// OBR - Observation Request Segment
type OBR struct {
	SetID                    string
	PlacerOrderNumber        string
	FillerOrderNumber        string
	UniversalServiceID       CodedElement
	Priority                 string
	RequestedDateTime        time.Time
	ObservationDateTime      time.Time
	ObservationEndDateTime   *time.Time
	CollectionVolume         string
	CollectorIdentifier      []Provider
	SpecimenActionCode       string
	DangerCode               string
	RelevantClinicalInfo     string
	SpecimenReceivedDateTime *time.Time
	SpecimenSource           string
	OrderingProvider         []Provider
	OrderCallbackPhone       string
	PlacerField1             string
	PlacerField2             string
	FillerField1             string
	FillerField2             string
	ResultsRptStatusChg      *time.Time
	ChargeToPractice         string
	DiagnosticServSectID     string
	ResultStatus             string
	ParentResult             string
	QuantityTiming           string
	ResultCopiesTo           []Provider
	Parent                   string
	TransportationMode       string
	ReasonForStudy           string
	PrincipalResultInterp    Provider
	AssistantResultInterp    []Provider
	Technician               []Provider
	Transcriptionist         []Provider
	ScheduledDateTime        *time.Time
	NumberOfSampleContainers int
	TransportLogisticsOfColl string
	CollectorComment         string
	TransportArrangement     string
	TransportArranged        string
	EscortRequired           string
	PlannedPatientTransport  string
}

func (o *OBR) ID() string { return "OBR" }

// OBX - Observation/Result Segment
type OBX struct {
	SetID                  string
	ValueType              string // NM, ST, TX, CE, etc.
	ObservationIdentifier  CodedElement
	ObservationSubID       string
	ObservationValue       []string
	Units                  CodedElement
	ReferencesRange        string
	AbnormalFlags          string
	Probability            string
	NatureOfAbnormalTest   string
	ObservationResultStatus string // F=Final, P=Preliminary, C=Correction
	EffectiveDateLastObs   *time.Time
	UserDefinedAccessChecks string
	DateTimeOfObservation  time.Time
	ProducersID            string
	ResponsibleObserver    Provider
	ObservationMethod      CodedElement
	EquipmentInstanceID    string
	DateTimeOfAnalysis     *time.Time
}

func (o *OBX) ID() string { return "OBX" }

// CodedElement represents a coded element (CE data type)
type CodedElement struct {
	Identifier       string
	Text             string
	NameOfCodingSys  string
	AltIdentifier    string
	AltText          string
	NameOfAltCodingSys string
}

// ORC - Common Order Segment
type ORC struct {
	OrderControl          string // NW=New, CA=Cancel, XO=Change, etc.
	PlacerOrderNumber     string
	FillerOrderNumber     string
	PlacerGroupNumber     string
	OrderStatus           string
	ResponseFlag          string
	QuantityTiming        string
	Parent                string
	DateTimeOfTransaction time.Time
	EnteredBy             Provider
	VerifiedBy            Provider
	OrderingProvider      Provider
	EnterersLocation      Location
	CallBackPhoneNumber   string
	OrderEffectiveDateTime *time.Time
	OrderControlCodeReason string
	EnteringOrganization  string
	EnteringDevice        string
	ActionBy              Provider
	AdvancedBeneficiary   string
	OrderingFacilityName  string
	OrderingFacilityAddr  Address
	OrderingFacilityPhone string
	OrderingProviderAddr  Address
}

func (o *ORC) ID() string { return "ORC" }

// NK1 - Next of Kin Segment
type NK1 struct {
	SetID        string
	Name         PersonName
	Relationship CodedElement
	Address      Address
	PhoneNumber  string
	BusinessPhone string
	ContactRole  CodedElement
	StartDate    *time.Time
	EndDate      *time.Time
	JobTitle     string
	JobCode      string
	EmployeeNum  string
	OrgName      string
	MaritalStatus string
	Sex          string
	DateOfBirth  *time.Time
	LivingDepend string
	AmbulatoryStatus string
	Citizenship  string
	PrimaryLang  string
	LivingArrang string
	PublicityCode string
	ProtectionInd string
}

func (n *NK1) ID() string { return "NK1" }

// IN1 - Insurance Segment
type IN1 struct {
	SetID                  string
	InsurancePlanID        CodedElement
	InsuranceCompanyID     string
	InsuranceCompanyName   string
	InsuranceCompanyAddr   Address
	InsuranceCoContactPers string
	InsuranceCoPhoneNumber string
	GroupNumber            string
	GroupName              string
	InsuredsGroupEmpID     string
	InsuredsGroupEmpName   string
	PlanEffectiveDate      *time.Time
	PlanExpirationDate     *time.Time
	AuthorizationInfo      string
	PlanType               string
	NameOfInsured          PersonName
	InsuredRelationship    string
	InsuredDateOfBirth     *time.Time
	InsuredAddress         Address
	AssignmentOfBenefits   string
	CoordinationOfBenefits string
	CoordOfBenPriority     string
	NoticeOfAdmissionFlag  string
	NoticeOfAdmissionDate  *time.Time
	ReportOfEligibilityFlag string
	ReportOfEligibilityDate *time.Time
	ReleaseInfoCode        string
	PreAdmitCert           string
	VerificationDateTime   *time.Time
	VerificationBy         Provider
	TypeOfAgreementCode    string
	BillingStatus          string
	LifetimeReserveDays    int
	DelayBeforeLRDay       int
	CompanyPlanCode        string
	PolicyNumber           string
	PolicyDeductible       float64
	PolicyLimitAmount      float64
	PolicyLimitDays        int
	RoomRateSemiPrivate    float64
	RoomRatePrivate        float64
	InsuredsEmploymentStatus string
	InsuredsSex            string
	InsuredsEmployerAddr   Address
	VerificationStatus     string
	PriorInsurancePlanID   string
	CoverageType           string
	Handicap               string
	InsuredsIDNumber       string
}

func (i *IN1) ID() string { return "IN1" }

// DG1 - Diagnosis Segment
type DG1 struct {
	SetID              string
	DiagnosisCodingMeth string
	DiagnosisCode      CodedElement
	DiagnosisDescr     string
	DiagnosisDateTime  time.Time
	DiagnosisType      string // A=Admitting, W=Working, F=Final
	MajorDiagnosticCat CodedElement
	DiagnosticRelGroup CodedElement
	DRGApprovalInd     string
	DRGGrouperReviewCode string
	OutlierType        string
	OutlierDays        int
	OutlierCost        float64
	GrouperVersionCode string
	DiagnosisPriority  int
	DiagnosingClinician Provider
	DiagnosisClassif   string
	ConfidentialInd    string
	AttestationDateTime *time.Time
}

func (d *DG1) ID() string { return "DG1" }

// PR1 - Procedures Segment
type PR1 struct {
	SetID             string
	ProcedureCodingMeth string
	ProcedureCode     CodedElement
	ProcedureDescr    string
	ProcedureDateTime time.Time
	ProcedureFuncType string
	ProcedureMinutes  int
	Anesthesiologist  Provider
	AnesthesiaCode    string
	AnesthesiaMinutes int
	Surgeon           Provider
	ProcedurePract    Provider
	ConsentCode       string
	ProcedurePriority int
	AssocDiagnosisCode CodedElement
	ProcedureCodeMod  string
	ProcedureDRGType  string
	TissueTypeCode    string
}

func (p *PR1) ID() string { return "PR1" }

// EVN - Event Type Segment
type EVN struct {
	EventTypeCode      string
	RecordedDateTime   time.Time
	DateTimePlannedEvt *time.Time
	EventReasonCode    string
	OperatorID         Provider
	EventOccurred      *time.Time
	EventFacility      string
}

func (e *EVN) ID() string { return "EVN" }

// AL1 - Patient Allergy Information
type AL1 struct {
	SetID              string
	AllergenTypeCode   string // DA=Drug allergy, FA=Food allergy, etc.
	AllergenCode       CodedElement
	AllergySeverity    string // MI=Mild, MO=Moderate, SV=Severe
	AllergyReaction    string
	IdentificationDate *time.Time
}

func (a *AL1) ID() string { return "AL1" }

// RXA - Pharmacy/Treatment Administration
type RXA struct {
	GiveSubIDCounter       int
	AdminSubIDCounter      int
	DateTimeStartOfAdmin   time.Time
	DateTimeEndOfAdmin     *time.Time
	AdminCode              CodedElement
	AdminAmount            float64
	AdminUnits             CodedElement
	AdminDosageForm        CodedElement
	AdminNotes             string
	AdminProvider          Provider
	AdminLocation          Location
	AdminPer               string
	AdminPerTimeUnit       string
	AdminStrength          float64
	AdminStrengthUnits     CodedElement
	SubstanceLotNumber     string
	SubstanceExpiration    *time.Time
	SubstanceManufacturer  string
	SubstanceRefusalReason string
	Indication             CodedElement
	CompletionStatus       string
	ActionCode             string
	SystemEntryDateTime    time.Time
}

func (r *RXA) ID() string { return "RXA" }
