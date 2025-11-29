package hl7v2

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Default HL7 v2.x delimiters
const (
	DefaultFieldSeparator     = "|"
	DefaultComponentSeparator = "^"
	DefaultRepetitionSep      = "~"
	DefaultEscapeCharacter    = "\\"
	DefaultSubcomponentSep    = "&"
	SegmentTerminator         = "\r"
)

// Parser parses HL7 v2.x messages
type Parser struct {
	fieldSep     string
	componentSep string
	repetitionSep string
	escapeChr    string
	subcompSep   string
	strictMode   bool
}

// ParserConfig holds parser configuration
type ParserConfig struct {
	StrictMode bool
}

// NewParser creates a new HL7 v2.x parser
func NewParser(config *ParserConfig) *Parser {
	strictMode := false
	if config != nil {
		strictMode = config.StrictMode
	}
	return &Parser{
		fieldSep:      DefaultFieldSeparator,
		componentSep:  DefaultComponentSeparator,
		repetitionSep: DefaultRepetitionSep,
		escapeChr:     DefaultEscapeCharacter,
		subcompSep:    DefaultSubcomponentSep,
		strictMode:    strictMode,
	}
}

// Parse parses an HL7 v2.x message from raw data
func (p *Parser) Parse(data []byte) (*Message, error) {
	content := string(data)

	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\r")
	content = strings.ReplaceAll(content, "\n", "\r")

	// Split into segments
	segmentStrings := strings.Split(content, SegmentTerminator)
	if len(segmentStrings) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	// Parse MSH segment first to get delimiters
	mshStr := strings.TrimSpace(segmentStrings[0])
	if !strings.HasPrefix(mshStr, "MSH") {
		return nil, fmt.Errorf("message must start with MSH segment")
	}

	// Extract delimiters from MSH
	if len(mshStr) < 8 {
		return nil, fmt.Errorf("invalid MSH segment: too short")
	}
	p.fieldSep = string(mshStr[3])
	encodingChars := mshStr[4:8]
	p.componentSep = string(encodingChars[0])
	p.repetitionSep = string(encodingChars[1])
	p.escapeChr = string(encodingChars[2])
	p.subcompSep = string(encodingChars[3])

	// Parse MSH
	msh, err := p.parseMSH(mshStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MSH: %w", err)
	}

	// Create message
	msg := &Message{
		Type:         MessageType(strings.Split(msh.MessageType, p.componentSep)[0]),
		Version:      msh.VersionID,
		ControlID:    msh.MessageControlID,
		Timestamp:    msh.DateTime,
		SendingApp:   msh.SendingApplication,
		SendingFac:   msh.SendingFacility,
		ReceivingApp: msh.ReceivingApplication,
		ReceivingFac: msh.ReceivingFacility,
		Segments:     []Segment{msh},
		RawData:      data,
	}

	// Extract trigger event
	msgTypeParts := strings.Split(msh.MessageType, p.componentSep)
	if len(msgTypeParts) >= 2 {
		msg.TriggerEvent = TriggerEvent(msgTypeParts[1])
	}

	// Parse remaining segments
	for i := 1; i < len(segmentStrings); i++ {
		segStr := strings.TrimSpace(segmentStrings[i])
		if segStr == "" {
			continue
		}

		seg, err := p.parseSegment(segStr)
		if err != nil {
			if p.strictMode {
				return nil, fmt.Errorf("failed to parse segment %d: %w", i, err)
			}
			continue
		}

		if seg != nil {
			msg.Segments = append(msg.Segments, seg)
		}
	}

	return msg, nil
}

// parseMSH parses an MSH segment
func (p *Parser) parseMSH(data string) (*MSH, error) {
	fields := strings.Split(data, p.fieldSep)

	msh := &MSH{
		FieldSeparator:     p.fieldSep,
		EncodingCharacters: fields[1],
	}

	if len(fields) > 2 {
		msh.SendingApplication = fields[2]
	}
	if len(fields) > 3 {
		msh.SendingFacility = fields[3]
	}
	if len(fields) > 4 {
		msh.ReceivingApplication = fields[4]
	}
	if len(fields) > 5 {
		msh.ReceivingFacility = fields[5]
	}
	if len(fields) > 6 {
		msh.DateTime = p.parseDateTime(fields[6])
	}
	if len(fields) > 7 {
		msh.Security = fields[7]
	}
	if len(fields) > 8 {
		msh.MessageType = fields[8]
	}
	if len(fields) > 9 {
		msh.MessageControlID = fields[9]
	}
	if len(fields) > 10 {
		msh.ProcessingID = fields[10]
	}
	if len(fields) > 11 {
		msh.VersionID = fields[11]
	}
	if len(fields) > 12 {
		msh.SequenceNumber = fields[12]
	}
	if len(fields) > 13 {
		msh.ContinuationPointer = fields[13]
	}
	if len(fields) > 14 {
		msh.AcceptAckType = fields[14]
	}
	if len(fields) > 15 {
		msh.ApplicationAckType = fields[15]
	}
	if len(fields) > 16 {
		msh.CountryCode = fields[16]
	}
	if len(fields) > 17 {
		msh.CharacterSet = fields[17]
	}

	return msh, nil
}

// parseSegment parses a generic segment
func (p *Parser) parseSegment(data string) (Segment, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("segment too short")
	}

	segmentID := data[:3]

	switch segmentID {
	case "MSH":
		return p.parseMSH(data)
	case "PID":
		return p.parsePID(data)
	case "PV1":
		return p.parsePV1(data)
	case "OBR":
		return p.parseOBR(data)
	case "OBX":
		return p.parseOBX(data)
	case "ORC":
		return p.parseORC(data)
	case "EVN":
		return p.parseEVN(data)
	case "NK1":
		return p.parseNK1(data)
	case "DG1":
		return p.parseDG1(data)
	case "AL1":
		return p.parseAL1(data)
	default:
		// Return a generic segment for unknown types
		return &GenericSegment{SegmentID: segmentID, RawData: data}, nil
	}
}

// parsePID parses a PID segment
func (p *Parser) parsePID(data string) (*PID, error) {
	fields := strings.Split(data, p.fieldSep)

	pid := &PID{}

	if len(fields) > 1 {
		pid.SetID = fields[1]
	}
	if len(fields) > 2 {
		pid.PatientID = fields[2]
	}
	if len(fields) > 3 {
		pid.PatientIdentifierList = p.parsePatientIdentifiers(fields[3])
	}
	if len(fields) > 4 {
		pid.AlternatePatientID = fields[4]
	}
	if len(fields) > 5 {
		pid.PatientName = p.parsePersonNames(fields[5])
	}
	if len(fields) > 6 {
		pid.MothersMaidenName = fields[6]
	}
	if len(fields) > 7 {
		pid.DateOfBirth = p.parseDateTime(fields[7])
	}
	if len(fields) > 8 {
		pid.AdministrativeSex = fields[8]
	}
	if len(fields) > 9 {
		pid.PatientAlias = fields[9]
	}
	if len(fields) > 10 {
		pid.Race = fields[10]
	}
	if len(fields) > 11 {
		pid.PatientAddress = p.parseAddresses(fields[11])
	}
	if len(fields) > 12 {
		pid.CountyCode = fields[12]
	}
	if len(fields) > 13 {
		pid.PhoneNumberHome = fields[13]
	}
	if len(fields) > 14 {
		pid.PhoneNumberBusiness = fields[14]
	}
	if len(fields) > 15 {
		pid.PrimaryLanguage = fields[15]
	}
	if len(fields) > 16 {
		pid.MaritalStatus = fields[16]
	}
	if len(fields) > 17 {
		pid.Religion = fields[17]
	}
	if len(fields) > 18 {
		pid.PatientAccountNumber = fields[18]
	}
	if len(fields) > 19 {
		pid.SSNNumber = fields[19]
	}

	return pid, nil
}

// parsePV1 parses a PV1 segment
func (p *Parser) parsePV1(data string) (*PV1, error) {
	fields := strings.Split(data, p.fieldSep)

	pv1 := &PV1{}

	if len(fields) > 1 {
		pv1.SetID = fields[1]
	}
	if len(fields) > 2 {
		pv1.PatientClass = fields[2]
	}
	if len(fields) > 3 {
		pv1.AssignedPatientLoc = p.parseLocation(fields[3])
	}
	if len(fields) > 4 {
		pv1.AdmissionType = fields[4]
	}
	if len(fields) > 5 {
		pv1.PreadmitNumber = fields[5]
	}
	if len(fields) > 6 {
		pv1.PriorPatientLoc = p.parseLocation(fields[6])
	}
	if len(fields) > 7 {
		pv1.AttendingDoctor = p.parseProvider(fields[7])
	}
	if len(fields) > 8 {
		pv1.ReferringDoctor = p.parseProvider(fields[8])
	}
	if len(fields) > 10 {
		pv1.HospitalService = fields[10]
	}
	if len(fields) > 17 {
		pv1.AdmittingDoctor = p.parseProvider(fields[17])
	}
	if len(fields) > 18 {
		pv1.PatientType = fields[18]
	}
	if len(fields) > 19 {
		pv1.VisitNumber = fields[19]
	}
	if len(fields) > 44 {
		pv1.AdmitDateTime = p.parseDateTime(fields[44])
	}
	if len(fields) > 45 && fields[45] != "" {
		dt := p.parseDateTime(fields[45])
		pv1.DischargeDateTime = &dt
	}

	return pv1, nil
}

// parseOBR parses an OBR segment
func (p *Parser) parseOBR(data string) (*OBR, error) {
	fields := strings.Split(data, p.fieldSep)

	obr := &OBR{}

	if len(fields) > 1 {
		obr.SetID = fields[1]
	}
	if len(fields) > 2 {
		obr.PlacerOrderNumber = fields[2]
	}
	if len(fields) > 3 {
		obr.FillerOrderNumber = fields[3]
	}
	if len(fields) > 4 {
		obr.UniversalServiceID = p.parseCodedElement(fields[4])
	}
	if len(fields) > 5 {
		obr.Priority = fields[5]
	}
	if len(fields) > 6 {
		obr.RequestedDateTime = p.parseDateTime(fields[6])
	}
	if len(fields) > 7 {
		obr.ObservationDateTime = p.parseDateTime(fields[7])
	}
	if len(fields) > 16 {
		obr.OrderingProvider = p.parseProviders(fields[16])
	}
	if len(fields) > 25 {
		obr.ResultStatus = fields[25]
	}

	return obr, nil
}

// parseOBX parses an OBX segment
func (p *Parser) parseOBX(data string) (*OBX, error) {
	fields := strings.Split(data, p.fieldSep)

	obx := &OBX{}

	if len(fields) > 1 {
		obx.SetID = fields[1]
	}
	if len(fields) > 2 {
		obx.ValueType = fields[2]
	}
	if len(fields) > 3 {
		obx.ObservationIdentifier = p.parseCodedElement(fields[3])
	}
	if len(fields) > 4 {
		obx.ObservationSubID = fields[4]
	}
	if len(fields) > 5 {
		obx.ObservationValue = p.parseRepetitions(fields[5])
	}
	if len(fields) > 6 {
		obx.Units = p.parseCodedElement(fields[6])
	}
	if len(fields) > 7 {
		obx.ReferencesRange = fields[7]
	}
	if len(fields) > 8 {
		obx.AbnormalFlags = fields[8]
	}
	if len(fields) > 11 {
		obx.ObservationResultStatus = fields[11]
	}
	if len(fields) > 14 {
		obx.DateTimeOfObservation = p.parseDateTime(fields[14])
	}

	return obx, nil
}

// parseORC parses an ORC segment
func (p *Parser) parseORC(data string) (*ORC, error) {
	fields := strings.Split(data, p.fieldSep)

	orc := &ORC{}

	if len(fields) > 1 {
		orc.OrderControl = fields[1]
	}
	if len(fields) > 2 {
		orc.PlacerOrderNumber = fields[2]
	}
	if len(fields) > 3 {
		orc.FillerOrderNumber = fields[3]
	}
	if len(fields) > 4 {
		orc.PlacerGroupNumber = fields[4]
	}
	if len(fields) > 5 {
		orc.OrderStatus = fields[5]
	}
	if len(fields) > 9 {
		orc.DateTimeOfTransaction = p.parseDateTime(fields[9])
	}
	if len(fields) > 10 {
		orc.EnteredBy = p.parseProvider(fields[10])
	}
	if len(fields) > 11 {
		orc.VerifiedBy = p.parseProvider(fields[11])
	}
	if len(fields) > 12 {
		orc.OrderingProvider = p.parseProvider(fields[12])
	}

	return orc, nil
}

// parseEVN parses an EVN segment
func (p *Parser) parseEVN(data string) (*EVN, error) {
	fields := strings.Split(data, p.fieldSep)

	evn := &EVN{}

	if len(fields) > 1 {
		evn.EventTypeCode = fields[1]
	}
	if len(fields) > 2 {
		evn.RecordedDateTime = p.parseDateTime(fields[2])
	}
	if len(fields) > 3 && fields[3] != "" {
		dt := p.parseDateTime(fields[3])
		evn.DateTimePlannedEvt = &dt
	}
	if len(fields) > 4 {
		evn.EventReasonCode = fields[4]
	}
	if len(fields) > 5 {
		evn.OperatorID = p.parseProvider(fields[5])
	}
	if len(fields) > 6 && fields[6] != "" {
		dt := p.parseDateTime(fields[6])
		evn.EventOccurred = &dt
	}
	if len(fields) > 7 {
		evn.EventFacility = fields[7]
	}

	return evn, nil
}

// parseNK1 parses an NK1 segment
func (p *Parser) parseNK1(data string) (*NK1, error) {
	fields := strings.Split(data, p.fieldSep)

	nk1 := &NK1{}

	if len(fields) > 1 {
		nk1.SetID = fields[1]
	}
	if len(fields) > 2 {
		names := p.parsePersonNames(fields[2])
		if len(names) > 0 {
			nk1.Name = names[0]
		}
	}
	if len(fields) > 3 {
		nk1.Relationship = p.parseCodedElement(fields[3])
	}
	if len(fields) > 4 {
		addrs := p.parseAddresses(fields[4])
		if len(addrs) > 0 {
			nk1.Address = addrs[0]
		}
	}
	if len(fields) > 5 {
		nk1.PhoneNumber = fields[5]
	}
	if len(fields) > 6 {
		nk1.BusinessPhone = fields[6]
	}
	if len(fields) > 7 {
		nk1.ContactRole = p.parseCodedElement(fields[7])
	}

	return nk1, nil
}

// parseDG1 parses a DG1 segment
func (p *Parser) parseDG1(data string) (*DG1, error) {
	fields := strings.Split(data, p.fieldSep)

	dg1 := &DG1{}

	if len(fields) > 1 {
		dg1.SetID = fields[1]
	}
	if len(fields) > 2 {
		dg1.DiagnosisCodingMeth = fields[2]
	}
	if len(fields) > 3 {
		dg1.DiagnosisCode = p.parseCodedElement(fields[3])
	}
	if len(fields) > 4 {
		dg1.DiagnosisDescr = fields[4]
	}
	if len(fields) > 5 {
		dg1.DiagnosisDateTime = p.parseDateTime(fields[5])
	}
	if len(fields) > 6 {
		dg1.DiagnosisType = fields[6]
	}
	if len(fields) > 15 {
		dg1.DiagnosisPriority, _ = strconv.Atoi(fields[15])
	}
	if len(fields) > 16 {
		dg1.DiagnosingClinician = p.parseProvider(fields[16])
	}

	return dg1, nil
}

// parseAL1 parses an AL1 segment
func (p *Parser) parseAL1(data string) (*AL1, error) {
	fields := strings.Split(data, p.fieldSep)

	al1 := &AL1{}

	if len(fields) > 1 {
		al1.SetID = fields[1]
	}
	if len(fields) > 2 {
		al1.AllergenTypeCode = fields[2]
	}
	if len(fields) > 3 {
		al1.AllergenCode = p.parseCodedElement(fields[3])
	}
	if len(fields) > 4 {
		al1.AllergySeverity = fields[4]
	}
	if len(fields) > 5 {
		al1.AllergyReaction = fields[5]
	}
	if len(fields) > 6 && fields[6] != "" {
		dt := p.parseDateTime(fields[6])
		al1.IdentificationDate = &dt
	}

	return al1, nil
}

// Helper parsing functions

func (p *Parser) parseDateTime(data string) time.Time {
	if data == "" {
		return time.Time{}
	}

	// Try different HL7 date/time formats
	formats := []string{
		"20060102150405",     // YYYYMMDDHHmmss
		"200601021504",       // YYYYMMDDHHmm
		"2006010215",         // YYYYMMDDHH
		"20060102",           // YYYYMMDD
		"20060102150405.000", // With milliseconds
		"20060102150405-0700", // With timezone
	}

	for _, format := range formats {
		if t, err := time.Parse(format, data); err == nil {
			return t
		}
	}

	return time.Time{}
}

func (p *Parser) parsePatientIdentifiers(data string) []PatientIdentifier {
	if data == "" {
		return nil
	}

	reps := strings.Split(data, p.repetitionSep)
	identifiers := make([]PatientIdentifier, 0, len(reps))

	for _, rep := range reps {
		comps := strings.Split(rep, p.componentSep)
		id := PatientIdentifier{}

		if len(comps) > 0 {
			id.ID = comps[0]
		}
		if len(comps) > 1 {
			id.CheckDigit = comps[1]
		}
		if len(comps) > 2 {
			id.CheckDigitScheme = comps[2]
		}
		if len(comps) > 3 {
			id.AssigningAuth = comps[3]
		}
		if len(comps) > 4 {
			id.IdentifierType = comps[4]
		}
		if len(comps) > 5 {
			id.AssigningFac = comps[5]
		}

		identifiers = append(identifiers, id)
	}

	return identifiers
}

func (p *Parser) parsePersonNames(data string) []PersonName {
	if data == "" {
		return nil
	}

	reps := strings.Split(data, p.repetitionSep)
	names := make([]PersonName, 0, len(reps))

	for _, rep := range reps {
		comps := strings.Split(rep, p.componentSep)
		name := PersonName{}

		if len(comps) > 0 {
			name.FamilyName = comps[0]
		}
		if len(comps) > 1 {
			name.GivenName = comps[1]
		}
		if len(comps) > 2 {
			name.MiddleName = comps[2]
		}
		if len(comps) > 3 {
			name.Suffix = comps[3]
		}
		if len(comps) > 4 {
			name.Prefix = comps[4]
		}
		if len(comps) > 5 {
			name.Degree = comps[5]
		}
		if len(comps) > 6 {
			name.NameTypeCode = comps[6]
		}

		names = append(names, name)
	}

	return names
}

func (p *Parser) parseAddresses(data string) []Address {
	if data == "" {
		return nil
	}

	reps := strings.Split(data, p.repetitionSep)
	addrs := make([]Address, 0, len(reps))

	for _, rep := range reps {
		comps := strings.Split(rep, p.componentSep)
		addr := Address{}

		if len(comps) > 0 {
			addr.StreetAddress = comps[0]
		}
		if len(comps) > 1 {
			addr.OtherDesig = comps[1]
		}
		if len(comps) > 2 {
			addr.City = comps[2]
		}
		if len(comps) > 3 {
			addr.State = comps[3]
		}
		if len(comps) > 4 {
			addr.ZipCode = comps[4]
		}
		if len(comps) > 5 {
			addr.Country = comps[5]
		}
		if len(comps) > 6 {
			addr.AddressType = comps[6]
		}

		addrs = append(addrs, addr)
	}

	return addrs
}

func (p *Parser) parseLocation(data string) Location {
	loc := Location{}
	if data == "" {
		return loc
	}

	comps := strings.Split(data, p.componentSep)

	if len(comps) > 0 {
		loc.PointOfCare = comps[0]
	}
	if len(comps) > 1 {
		loc.Room = comps[1]
	}
	if len(comps) > 2 {
		loc.Bed = comps[2]
	}
	if len(comps) > 3 {
		loc.Facility = comps[3]
	}
	if len(comps) > 4 {
		loc.LocationSt = comps[4]
	}
	if len(comps) > 5 {
		loc.Building = comps[5]
	}
	if len(comps) > 6 {
		loc.Floor = comps[6]
	}

	return loc
}

func (p *Parser) parseProvider(data string) Provider {
	prov := Provider{}
	if data == "" {
		return prov
	}

	comps := strings.Split(data, p.componentSep)

	if len(comps) > 0 {
		prov.ID = comps[0]
	}
	if len(comps) > 1 {
		prov.FamilyName = comps[1]
	}
	if len(comps) > 2 {
		prov.GivenName = comps[2]
	}
	if len(comps) > 3 {
		prov.MiddleName = comps[3]
	}
	if len(comps) > 4 {
		prov.Suffix = comps[4]
	}
	if len(comps) > 5 {
		prov.Prefix = comps[5]
	}
	if len(comps) > 6 {
		prov.Degree = comps[6]
	}
	if len(comps) > 7 {
		prov.SourceTable = comps[7]
	}
	if len(comps) > 8 {
		prov.AssignAuth = comps[8]
	}
	if len(comps) > 9 {
		prov.NameTypeCode = comps[9]
	}

	return prov
}

func (p *Parser) parseProviders(data string) []Provider {
	if data == "" {
		return nil
	}

	reps := strings.Split(data, p.repetitionSep)
	providers := make([]Provider, 0, len(reps))

	for _, rep := range reps {
		providers = append(providers, p.parseProvider(rep))
	}

	return providers
}

func (p *Parser) parseCodedElement(data string) CodedElement {
	ce := CodedElement{}
	if data == "" {
		return ce
	}

	comps := strings.Split(data, p.componentSep)

	if len(comps) > 0 {
		ce.Identifier = comps[0]
	}
	if len(comps) > 1 {
		ce.Text = comps[1]
	}
	if len(comps) > 2 {
		ce.NameOfCodingSys = comps[2]
	}
	if len(comps) > 3 {
		ce.AltIdentifier = comps[3]
	}
	if len(comps) > 4 {
		ce.AltText = comps[4]
	}
	if len(comps) > 5 {
		ce.NameOfAltCodingSys = comps[5]
	}

	return ce
}

func (p *Parser) parseRepetitions(data string) []string {
	if data == "" {
		return nil
	}
	return strings.Split(data, p.repetitionSep)
}

// GenericSegment represents an unknown segment type
type GenericSegment struct {
	SegmentID string
	RawData   string
}

func (g *GenericSegment) ID() string { return g.SegmentID }

func (g *GenericSegment) Encode() (string, error) {
	return g.RawData, nil
}

func (g *GenericSegment) Decode(data string) error {
	g.RawData = data
	return nil
}

// Encode methods for segments

func (m *MSH) Encode() (string, error) {
	fields := []string{
		"MSH",
		m.EncodingCharacters,
		m.SendingApplication,
		m.SendingFacility,
		m.ReceivingApplication,
		m.ReceivingFacility,
		m.DateTime.Format("20060102150405"),
		m.Security,
		m.MessageType,
		m.MessageControlID,
		m.ProcessingID,
		m.VersionID,
		m.SequenceNumber,
		m.ContinuationPointer,
		m.AcceptAckType,
		m.ApplicationAckType,
		m.CountryCode,
		m.CharacterSet,
	}
	return m.FieldSeparator + strings.Join(fields[1:], m.FieldSeparator), nil
}

func (m *MSH) Decode(data string) error {
	// Parsing is done in parseMSH
	return nil
}

func (p *PID) Encode() (string, error) {
	return "", nil // Implement as needed
}

func (p *PID) Decode(data string) error {
	return nil
}

func (p *PV1) Encode() (string, error) {
	return "", nil
}

func (p *PV1) Decode(data string) error {
	return nil
}

func (o *OBR) Encode() (string, error) {
	return "", nil
}

func (o *OBR) Decode(data string) error {
	return nil
}

func (o *OBX) Encode() (string, error) {
	return "", nil
}

func (o *OBX) Decode(data string) error {
	return nil
}

func (o *ORC) Encode() (string, error) {
	return "", nil
}

func (o *ORC) Decode(data string) error {
	return nil
}

func (e *EVN) Encode() (string, error) {
	return "", nil
}

func (e *EVN) Decode(data string) error {
	return nil
}

func (n *NK1) Encode() (string, error) {
	return "", nil
}

func (n *NK1) Decode(data string) error {
	return nil
}

func (i *IN1) Encode() (string, error) {
	return "", nil
}

func (i *IN1) Decode(data string) error {
	return nil
}

func (d *DG1) Encode() (string, error) {
	return "", nil
}

func (d *DG1) Decode(data string) error {
	return nil
}

func (p *PR1) Encode() (string, error) {
	return "", nil
}

func (p *PR1) Decode(data string) error {
	return nil
}

func (a *AL1) Encode() (string, error) {
	return "", nil
}

func (a *AL1) Decode(data string) error {
	return nil
}

func (r *RXA) Encode() (string, error) {
	return "", nil
}

func (r *RXA) Decode(data string) error {
	return nil
}
