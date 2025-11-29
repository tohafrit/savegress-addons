package hl7v2

import (
	"fmt"
	"time"

	"github.com/savegress/healthsync/pkg/models"
)

// Converter converts between HL7 v2.x and FHIR resources
type Converter struct{}

// NewConverter creates a new converter
func NewConverter() *Converter {
	return &Converter{}
}

// HL7ToFHIRPatient converts HL7 PID segment to FHIR Patient
func (c *Converter) HL7ToFHIRPatient(pid *PID) *models.Patient {
	patient := &models.Patient{
		ResourceType: models.ResourceTypePatient,
	}

	// Set identifiers
	for _, id := range pid.PatientIdentifierList {
		identifier := models.Identifier{
			Value: id.ID,
		}
		if id.AssigningAuth != "" {
			identifier.System = id.AssigningAuth
		}
		if id.IdentifierType != "" {
			identifier.Type = &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: id.IdentifierType},
				},
			}
		}
		patient.Identifier = append(patient.Identifier, identifier)
	}

	// Set names
	for _, name := range pid.PatientName {
		humanName := models.HumanName{
			Family: name.FamilyName,
		}
		if name.GivenName != "" {
			humanName.Given = append(humanName.Given, name.GivenName)
		}
		if name.MiddleName != "" {
			humanName.Given = append(humanName.Given, name.MiddleName)
		}
		if name.Prefix != "" {
			humanName.Prefix = append(humanName.Prefix, name.Prefix)
		}
		if name.Suffix != "" {
			humanName.Suffix = append(humanName.Suffix, name.Suffix)
		}
		patient.Name = append(patient.Name, humanName)
	}

	// Set birth date
	if !pid.DateOfBirth.IsZero() {
		patient.BirthDate = pid.DateOfBirth.Format("2006-01-02")
	}

	// Set gender
	switch pid.AdministrativeSex {
	case "M":
		patient.Gender = "male"
	case "F":
		patient.Gender = "female"
	case "O":
		patient.Gender = "other"
	case "U":
		patient.Gender = "unknown"
	}

	// Set addresses
	for _, addr := range pid.PatientAddress {
		fhirAddr := models.Address{
			City:       addr.City,
			State:      addr.State,
			PostalCode: addr.ZipCode,
			Country:    addr.Country,
		}
		if addr.StreetAddress != "" {
			fhirAddr.Line = append(fhirAddr.Line, addr.StreetAddress)
		}
		if addr.OtherDesig != "" {
			fhirAddr.Line = append(fhirAddr.Line, addr.OtherDesig)
		}
		switch addr.AddressType {
		case "H":
			fhirAddr.Use = "home"
		case "B":
			fhirAddr.Use = "work"
		case "C":
			fhirAddr.Use = "temp"
		}
		patient.Address = append(patient.Address, fhirAddr)
	}

	// Set telecom
	if pid.PhoneNumberHome != "" {
		patient.Telecom = append(patient.Telecom, models.ContactPoint{
			System: "phone",
			Value:  pid.PhoneNumberHome,
			Use:    "home",
		})
	}
	if pid.PhoneNumberBusiness != "" {
		patient.Telecom = append(patient.Telecom, models.ContactPoint{
			System: "phone",
			Value:  pid.PhoneNumberBusiness,
			Use:    "work",
		})
	}

	// Set marital status
	if pid.MaritalStatus != "" {
		patient.MaritalStatus = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "http://terminology.hl7.org/CodeSystem/v3-MaritalStatus",
					Code:   c.mapMaritalStatus(pid.MaritalStatus),
				},
			},
		}
	}

	// Set deceased
	if pid.PatientDeathIndicator == "Y" {
		deceased := true
		patient.DeceasedBoolean = &deceased
		if pid.PatientDeathDateTime != nil {
			patient.DeceasedDateTime = pid.PatientDeathDateTime.Format(time.RFC3339)
		}
	}

	return patient
}

// FHIRToHL7Patient converts FHIR Patient to HL7 PID segment
func (c *Converter) FHIRToHL7Patient(patient *models.Patient) *PID {
	pid := &PID{
		SetID: "1",
	}

	// Set identifiers
	for _, id := range patient.Identifier {
		hl7ID := PatientIdentifier{
			ID:            id.Value,
			AssigningAuth: id.System,
		}
		if id.Type != nil && len(id.Type.Coding) > 0 {
			hl7ID.IdentifierType = id.Type.Coding[0].Code
		}
		pid.PatientIdentifierList = append(pid.PatientIdentifierList, hl7ID)
	}

	// Set names
	for _, name := range patient.Name {
		hl7Name := PersonName{
			FamilyName: name.Family,
		}
		if len(name.Given) > 0 {
			hl7Name.GivenName = name.Given[0]
		}
		if len(name.Given) > 1 {
			hl7Name.MiddleName = name.Given[1]
		}
		if len(name.Prefix) > 0 {
			hl7Name.Prefix = name.Prefix[0]
		}
		if len(name.Suffix) > 0 {
			hl7Name.Suffix = name.Suffix[0]
		}
		pid.PatientName = append(pid.PatientName, hl7Name)
	}

	// Set birth date
	if patient.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", patient.BirthDate); err == nil {
			pid.DateOfBirth = t
		}
	}

	// Set gender
	switch patient.Gender {
	case "male":
		pid.AdministrativeSex = "M"
	case "female":
		pid.AdministrativeSex = "F"
	case "other":
		pid.AdministrativeSex = "O"
	case "unknown":
		pid.AdministrativeSex = "U"
	}

	// Set addresses
	for _, addr := range patient.Address {
		hl7Addr := Address{
			City:    addr.City,
			State:   addr.State,
			ZipCode: addr.PostalCode,
			Country: addr.Country,
		}
		if len(addr.Line) > 0 {
			hl7Addr.StreetAddress = addr.Line[0]
		}
		if len(addr.Line) > 1 {
			hl7Addr.OtherDesig = addr.Line[1]
		}
		switch addr.Use {
		case "home":
			hl7Addr.AddressType = "H"
		case "work":
			hl7Addr.AddressType = "B"
		case "temp":
			hl7Addr.AddressType = "C"
		}
		pid.PatientAddress = append(pid.PatientAddress, hl7Addr)
	}

	// Set telecom
	for _, telecom := range patient.Telecom {
		if telecom.System == "phone" {
			switch telecom.Use {
			case "home":
				pid.PhoneNumberHome = telecom.Value
			case "work":
				pid.PhoneNumberBusiness = telecom.Value
			}
		}
	}

	return pid
}

// HL7ToFHIREncounter converts HL7 PV1 segment to FHIR Encounter
func (c *Converter) HL7ToFHIREncounter(pv1 *PV1, patientRef string) *models.Encounter {
	encounter := &models.Encounter{
		ResourceType: models.ResourceTypeEncounter,
	}

	// Set status based on patient class and discharge
	if pv1.DischargeDateTime != nil {
		encounter.Status = "finished"
	} else {
		encounter.Status = "in-progress"
	}

	// Set class
	encounter.Class = models.Coding{
		System: "http://terminology.hl7.org/CodeSystem/v3-ActCode",
		Code:   c.mapPatientClass(pv1.PatientClass),
	}

	// Set subject
	encounter.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set period
	encounter.Period = &models.Period{}
	if !pv1.AdmitDateTime.IsZero() {
		start := pv1.AdmitDateTime.Format(time.RFC3339)
		encounter.Period.Start = start
	}
	if pv1.DischargeDateTime != nil {
		end := pv1.DischargeDateTime.Format(time.RFC3339)
		encounter.Period.End = end
	}

	// Set location
	if pv1.AssignedPatientLoc.PointOfCare != "" {
		encounter.Location = append(encounter.Location, models.EncounterLocation{
			Location: models.Reference{
				Display: fmt.Sprintf("%s-%s-%s",
					pv1.AssignedPatientLoc.PointOfCare,
					pv1.AssignedPatientLoc.Room,
					pv1.AssignedPatientLoc.Bed),
			},
			Status: "active",
		})
	}

	// Set participants
	if pv1.AttendingDoctor.ID != "" {
		encounter.Participant = append(encounter.Participant, models.EncounterParticipant{
			Type: []models.CodeableConcept{
				{
					Coding: []models.Coding{
						{
							System: "http://terminology.hl7.org/CodeSystem/v3-ParticipationType",
							Code:   "ATND",
						},
					},
				},
			},
			Individual: &models.Reference{
				Display: fmt.Sprintf("%s %s", pv1.AttendingDoctor.GivenName, pv1.AttendingDoctor.FamilyName),
			},
		})
	}

	// Set service type
	if pv1.HospitalService != "" {
		encounter.ServiceType = &models.CodeableConcept{
			Coding: []models.Coding{
				{Code: pv1.HospitalService},
			},
		}
	}

	// Set identifiers
	if pv1.VisitNumber != "" {
		encounter.Identifier = append(encounter.Identifier, models.Identifier{
			Value: pv1.VisitNumber,
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "VN"},
				},
			},
		})
	}

	return encounter
}

// HL7ToFHIRObservation converts HL7 OBX segment to FHIR Observation
func (c *Converter) HL7ToFHIRObservation(obx *OBX, patientRef, encounterRef string) *models.Observation {
	obs := &models.Observation{
		ResourceType: models.ResourceTypeObservation,
	}

	// Set status
	switch obx.ObservationResultStatus {
	case "F":
		obs.Status = "final"
	case "P":
		obs.Status = "preliminary"
	case "C":
		obs.Status = "corrected"
	case "X":
		obs.Status = "cancelled"
	default:
		obs.Status = "unknown"
	}

	// Set code
	obs.Code = models.CodeableConcept{
		Coding: []models.Coding{
			{
				Code:    obx.ObservationIdentifier.Identifier,
				Display: obx.ObservationIdentifier.Text,
				System:  c.mapCodingSystem(obx.ObservationIdentifier.NameOfCodingSys),
			},
		},
	}

	// Set subject
	obs.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set encounter
	if encounterRef != "" {
		obs.Encounter = &models.Reference{
			Reference: encounterRef,
		}
	}

	// Set effective date
	if !obx.DateTimeOfObservation.IsZero() {
		obs.EffectiveDateTime = obx.DateTimeOfObservation.Format(time.RFC3339)
	}

	// Set value based on type
	if len(obx.ObservationValue) > 0 {
		c.setObservationValue(obs, obx.ValueType, obx.ObservationValue[0], obx.Units)
	}

	// Set reference range
	if obx.ReferencesRange != "" {
		obs.ReferenceRange = append(obs.ReferenceRange, models.ObservationReferenceRange{
			Text: obx.ReferencesRange,
		})
	}

	// Set interpretation
	if obx.AbnormalFlags != "" {
		obs.Interpretation = append(obs.Interpretation, models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
					Code:   c.mapAbnormalFlag(obx.AbnormalFlags),
				},
			},
		})
	}

	return obs
}

// setObservationValue sets the observation value based on HL7 value type
func (c *Converter) setObservationValue(obs *models.Observation, valueType, value string, units CodedElement) {
	switch valueType {
	case "NM": // Numeric
		obs.ValueQuantity = &models.Quantity{
			Value: c.parseFloat(value),
			Unit:  units.Text,
			Code:  units.Identifier,
		}
	case "ST", "TX": // String, Text
		obs.ValueString = value
	case "CE", "CWE": // Coded element
		obs.ValueCodeableConcept = &models.CodeableConcept{
			Coding: []models.Coding{
				{Code: value},
			},
		}
	case "DT": // Date
		obs.ValueDateTime = value
	case "TM": // Time
		obs.ValueTime = value
	default:
		obs.ValueString = value
	}
}

// Helper functions

func (c *Converter) mapMaritalStatus(hl7Status string) string {
	mapping := map[string]string{
		"A": "A", // Annulled
		"D": "D", // Divorced
		"I": "I", // Interlocutory
		"L": "L", // Legally Separated
		"M": "M", // Married
		"P": "P", // Polygamous
		"S": "S", // Never Married
		"T": "T", // Domestic partner
		"U": "U", // Unknown
		"W": "W", // Widowed
	}
	if mapped, ok := mapping[hl7Status]; ok {
		return mapped
	}
	return "U"
}

func (c *Converter) mapPatientClass(hl7Class string) string {
	mapping := map[string]string{
		"I": "IMP",   // Inpatient
		"O": "AMB",   // Outpatient/Ambulatory
		"E": "EMER",  // Emergency
		"P": "PRENC", // Pre-admission
		"R": "SS",    // Recurring patient
		"B": "OBSENC", // Obstetrics
	}
	if mapped, ok := mapping[hl7Class]; ok {
		return mapped
	}
	return "AMB"
}

func (c *Converter) mapCodingSystem(hl7System string) string {
	mapping := map[string]string{
		"LN":   "http://loinc.org",
		"SCT":  "http://snomed.info/sct",
		"I9":   "http://hl7.org/fhir/sid/icd-9-cm",
		"I10":  "http://hl7.org/fhir/sid/icd-10",
		"CPT":  "http://www.ama-assn.org/go/cpt",
		"HCPCS": "http://www.cms.gov/Medicare/Coding/HCPCSReleaseCodeSets",
		"NDC":  "http://hl7.org/fhir/sid/ndc",
		"RxNorm": "http://www.nlm.nih.gov/research/umls/rxnorm",
	}
	if mapped, ok := mapping[hl7System]; ok {
		return mapped
	}
	return hl7System
}

func (c *Converter) mapAbnormalFlag(flag string) string {
	mapping := map[string]string{
		"L":  "L",  // Low
		"H":  "H",  // High
		"LL": "LL", // Critical low
		"HH": "HH", // Critical high
		"<":  "L",  // Below absolute low
		">":  "H",  // Above absolute high
		"N":  "N",  // Normal
		"A":  "A",  // Abnormal
		"AA": "AA", // Critical abnormal
		"U":  "U",  // Significant change up
		"D":  "D",  // Significant change down
		"B":  "B",  // Better
		"W":  "W",  // Worse
	}
	if mapped, ok := mapping[flag]; ok {
		return mapped
	}
	return flag
}

func (c *Converter) parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// HL7ToFHIRDiagnosticReport converts HL7 OBR+OBXs to FHIR DiagnosticReport
func (c *Converter) HL7ToFHIRDiagnosticReport(obr *OBR, observations []*OBX, patientRef string) *models.DiagnosticReport {
	report := &models.DiagnosticReport{
		ResourceType: models.ResourceTypeDiagnosticReport,
	}

	// Set status
	switch obr.ResultStatus {
	case "F":
		report.Status = "final"
	case "P":
		report.Status = "preliminary"
	case "C":
		report.Status = "corrected"
	case "X":
		report.Status = "cancelled"
	default:
		report.Status = "unknown"
	}

	// Set code
	report.Code = models.CodeableConcept{
		Coding: []models.Coding{
			{
				Code:    obr.UniversalServiceID.Identifier,
				Display: obr.UniversalServiceID.Text,
				System:  c.mapCodingSystem(obr.UniversalServiceID.NameOfCodingSys),
			},
		},
	}

	// Set subject
	report.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set effective date
	if !obr.ObservationDateTime.IsZero() {
		report.EffectiveDateTime = obr.ObservationDateTime.Format(time.RFC3339)
	}

	// Set issued date
	if !obr.ObservationDateTime.IsZero() {
		report.Issued = obr.ObservationDateTime.Format(time.RFC3339)
	}

	// Set identifiers
	if obr.FillerOrderNumber != "" {
		report.Identifier = append(report.Identifier, models.Identifier{
			Value: obr.FillerOrderNumber,
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "FILL"},
				},
			},
		})
	}
	if obr.PlacerOrderNumber != "" {
		report.Identifier = append(report.Identifier, models.Identifier{
			Value: obr.PlacerOrderNumber,
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "PLAC"},
				},
			},
		})
	}

	return report
}

// HL7ToFHIRServiceRequest converts HL7 ORC+OBR to FHIR ServiceRequest
func (c *Converter) HL7ToFHIRServiceRequest(orc *ORC, obr *OBR, patientRef string) *models.ServiceRequest {
	sr := &models.ServiceRequest{
		ResourceType: models.ResourceTypeServiceRequest,
	}

	// Set status
	switch orc.OrderStatus {
	case "IP":
		sr.Status = "active"
	case "CM":
		sr.Status = "completed"
	case "CA":
		sr.Status = "revoked"
	case "DC":
		sr.Status = "revoked"
	case "HD":
		sr.Status = "on-hold"
	default:
		sr.Status = "active"
	}

	// Set intent based on order control
	switch orc.OrderControl {
	case "NW":
		sr.Intent = "order"
	case "SC":
		sr.Intent = "order"
	case "RF":
		sr.Intent = "reflex-order"
	default:
		sr.Intent = "order"
	}

	// Set code
	sr.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				Code:    obr.UniversalServiceID.Identifier,
				Display: obr.UniversalServiceID.Text,
				System:  c.mapCodingSystem(obr.UniversalServiceID.NameOfCodingSys),
			},
		},
	}

	// Set subject
	sr.Subject = models.Reference{
		Reference: patientRef,
	}

	// Set requester
	if orc.OrderingProvider.ID != "" {
		sr.Requester = &models.Reference{
			Display: fmt.Sprintf("%s %s", orc.OrderingProvider.GivenName, orc.OrderingProvider.FamilyName),
		}
	}

	// Set authored on
	if !orc.DateTimeOfTransaction.IsZero() {
		sr.AuthoredOn = orc.DateTimeOfTransaction.Format(time.RFC3339)
	}

	// Set priority
	switch obr.Priority {
	case "S":
		sr.Priority = "stat"
	case "A":
		sr.Priority = "asap"
	case "R":
		sr.Priority = "routine"
	default:
		sr.Priority = "routine"
	}

	// Set identifiers
	if orc.PlacerOrderNumber != "" {
		sr.Identifier = append(sr.Identifier, models.Identifier{
			Value: orc.PlacerOrderNumber,
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "PLAC"},
				},
			},
		})
	}
	if orc.FillerOrderNumber != "" {
		sr.Identifier = append(sr.Identifier, models.Identifier{
			Value: orc.FillerOrderNumber,
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "FILL"},
				},
			},
		})
	}

	return sr
}

// HL7ToFHIRCondition converts HL7 DG1 segment to FHIR Condition
func (c *Converter) HL7ToFHIRCondition(dg1 *DG1, patientRef, encounterRef string) *models.Condition {
	cond := &models.Condition{
		ResourceType: models.ResourceTypeCondition,
	}

	// Set clinical status
	cond.ClinicalStatus = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/condition-clinical",
				Code:   "active",
			},
		},
	}

	// Set verification status
	switch dg1.DiagnosisType {
	case "F":
		cond.VerificationStatus = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "http://terminology.hl7.org/CodeSystem/condition-ver-status",
					Code:   "confirmed",
				},
			},
		}
	case "W":
		cond.VerificationStatus = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "http://terminology.hl7.org/CodeSystem/condition-ver-status",
					Code:   "provisional",
				},
			},
		}
	default:
		cond.VerificationStatus = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "http://terminology.hl7.org/CodeSystem/condition-ver-status",
					Code:   "unconfirmed",
				},
			},
		}
	}

	// Set code
	cond.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				Code:    dg1.DiagnosisCode.Identifier,
				Display: dg1.DiagnosisCode.Text,
				System:  c.mapCodingSystem(dg1.DiagnosisCode.NameOfCodingSys),
			},
		},
	}
	if dg1.DiagnosisDescr != "" {
		cond.Code.Text = dg1.DiagnosisDescr
	}

	// Set subject
	cond.Subject = models.Reference{
		Reference: patientRef,
	}

	// Set encounter
	if encounterRef != "" {
		cond.Encounter = &models.Reference{
			Reference: encounterRef,
		}
	}

	// Set recorded date
	if !dg1.DiagnosisDateTime.IsZero() {
		cond.RecordedDate = dg1.DiagnosisDateTime.Format(time.RFC3339)
	}

	// Set asserter
	if dg1.DiagnosingClinician.ID != "" {
		cond.Asserter = &models.Reference{
			Display: fmt.Sprintf("%s %s", dg1.DiagnosingClinician.GivenName, dg1.DiagnosingClinician.FamilyName),
		}
	}

	return cond
}

// HL7ToFHIRAllergyIntolerance converts HL7 AL1 segment to FHIR AllergyIntolerance
func (c *Converter) HL7ToFHIRAllergyIntolerance(al1 *AL1, patientRef string) *models.AllergyIntolerance {
	allergy := &models.AllergyIntolerance{
		ResourceType: models.ResourceTypeAllergyIntolerance,
	}

	// Set clinical status
	allergy.ClinicalStatus = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical",
				Code:   "active",
			},
		},
	}

	// Set verification status
	allergy.VerificationStatus = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/allergyintolerance-verification",
				Code:   "confirmed",
			},
		},
	}

	// Set type based on allergen type code
	switch al1.AllergenTypeCode {
	case "DA":
		allergy.Type = "allergy"
		allergy.Category = append(allergy.Category, "medication")
	case "FA":
		allergy.Type = "allergy"
		allergy.Category = append(allergy.Category, "food")
	case "EA":
		allergy.Type = "allergy"
		allergy.Category = append(allergy.Category, "environment")
	default:
		allergy.Type = "allergy"
	}

	// Set criticality based on severity
	switch al1.AllergySeverity {
	case "SV":
		allergy.Criticality = "high"
	case "MO":
		allergy.Criticality = "low"
	case "MI":
		allergy.Criticality = "low"
	}

	// Set code
	allergy.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				Code:    al1.AllergenCode.Identifier,
				Display: al1.AllergenCode.Text,
				System:  c.mapCodingSystem(al1.AllergenCode.NameOfCodingSys),
			},
		},
	}

	// Set patient
	allergy.Patient = models.Reference{
		Reference: patientRef,
	}

	// Set recorded date
	if al1.IdentificationDate != nil {
		allergy.RecordedDate = al1.IdentificationDate.Format(time.RFC3339)
	}

	// Set reaction
	if al1.AllergyReaction != "" {
		allergy.Reaction = append(allergy.Reaction, models.AllergyReaction{
			Manifestation: []models.CodeableConcept{
				{Text: al1.AllergyReaction},
			},
		})
	}

	return allergy
}
