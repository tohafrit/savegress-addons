package ieee11073

import (
	"fmt"
	"time"

	"github.com/savegress/healthsync/pkg/models"
)

// FHIRConverter converts IEEE 11073 measurements to FHIR resources
type FHIRConverter struct{}

// NewFHIRConverter creates a new FHIR converter
func NewFHIRConverter() *FHIRConverter {
	return &FHIRConverter{}
}

// MeasurementToObservation converts an IEEE 11073 measurement to a FHIR Observation
func (c *FHIRConverter) MeasurementToObservation(m *Measurement, patientRef string) *models.Observation {
	obs := &models.Observation{
		ResourceType: models.ResourceTypeObservation,
	}

	// Set status
	switch m.Status {
	case MeasStatusValid:
		obs.Status = "final"
	case MeasStatusQuestionable:
		obs.Status = "preliminary"
	case MeasStatusMeasurementOngoing:
		obs.Status = "preliminary"
	case MeasStatusCalibrating:
		obs.Status = "registered"
	case MeasStatusNotAvailable, MeasStatusNoData:
		obs.Status = "cancelled"
	default:
		obs.Status = "unknown"
	}

	// Set category based on device category
	obs.Category = append(obs.Category, models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/observation-category",
				Code:   c.getCategoryCode(m.Category),
			},
		},
	})

	// Set code from nomenclature
	info := NomenclatureRegistry[m.Code]
	obs.Code = models.CodeableConcept{
		Coding: []models.Coding{
			{
				System:  "urn:iso:std:iso:11073:10101",
				Code:    fmt.Sprintf("%d", m.Code),
				Display: info.Description,
			},
		},
	}

	// Add LOINC mapping if available
	if loincCode := c.mapToLOINC(m.Code); loincCode != "" {
		obs.Code.Coding = append(obs.Code.Coding, models.Coding{
			System: "http://loinc.org",
			Code:   loincCode,
		})
	}

	// Set subject
	obs.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set device reference
	obs.Device = &models.Reference{
		Display: m.DeviceID,
	}

	// Set effective time
	obs.EffectiveDateTime = m.Timestamp.Format(time.RFC3339)

	// Set value
	unitSymbol := UnitRegistry[m.Unit]
	obs.ValueQuantity = &models.Quantity{
		Value:  m.Value,
		Unit:   unitSymbol,
		System: "urn:iso:std:iso:11073:10101",
		Code:   fmt.Sprintf("%d", m.Unit),
	}

	// Set reference range if available
	if m.LowerRange != nil || m.UpperRange != nil {
		rr := models.ObservationReferenceRange{}
		if m.LowerRange != nil {
			rr.Low = &models.Quantity{Value: *m.LowerRange, Unit: unitSymbol}
		}
		if m.UpperRange != nil {
			rr.High = &models.Quantity{Value: *m.UpperRange, Unit: unitSymbol}
		}
		obs.ReferenceRange = append(obs.ReferenceRange, rr)
	}

	// Set identifier
	obs.Identifier = append(obs.Identifier, models.Identifier{
		System: "urn:savegress:measurement",
		Value:  m.ID,
	})

	return obs
}

// DeviceToFHIRDevice converts IEEE 11073 device config to FHIR Device
func (c *FHIRConverter) DeviceToFHIRDevice(config *DeviceConfiguration) *models.Device {
	device := &models.Device{
		ResourceType: models.ResourceTypeDevice,
		Status:       "active",
	}

	// Set identifier
	device.Identifier = append(device.Identifier, models.Identifier{
		System: "urn:iso:std:iso:11073:10101",
		Value:  config.DeviceID.ID,
	})

	// Set manufacturer
	device.Manufacturer = config.DeviceID.Manufacturer

	// Set model number
	device.ModelNumber = config.DeviceID.Model

	// Set serial number
	device.SerialNumber = config.DeviceID.SerialNumber

	// Set type based on category
	device.Type = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				System:  "urn:iso:std:iso:11073:10101",
				Code:    c.getCategoryMDCCode(config.Category),
				Display: string(config.Category),
			},
		},
	}

	// Add specialization for IEEE 11073
	device.Specialization = append(device.Specialization, models.DeviceSpecialization{
		SystemType: models.CodeableConcept{
			Coding: []models.Coding{
				{
					System: "urn:iso:std:iso:11073:10101",
					Code:   c.getCategoryMDCCode(config.Category),
				},
			},
		},
	})

	// Set version (firmware)
	if config.DeviceID.FirmwareRev != "" {
		device.Version = append(device.Version, models.DeviceVersion{
			Type: &models.CodeableConcept{
				Coding: []models.Coding{
					{Code: "firmware"},
				},
			},
			Value: config.DeviceID.FirmwareRev,
		})
	}

	return device
}

// AlertToDetectedIssue converts IEEE 11073 alert to FHIR DetectedIssue
func (c *FHIRConverter) AlertToDetectedIssue(alert *Alert, patientRef string) *models.DetectedIssue {
	issue := &models.DetectedIssue{
		ResourceType: models.ResourceTypeDetectedIssue,
		Status:       "final",
	}

	// Set severity
	switch alert.Priority {
	case AlertPriorityHigh:
		issue.Severity = "high"
	case AlertPriorityMedium:
		issue.Severity = "moderate"
	case AlertPriorityLow:
		issue.Severity = "low"
	}

	// Set code
	issue.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			{
				System:  "urn:iso:std:iso:11073:10101",
				Code:    fmt.Sprintf("%d", alert.Code),
				Display: alert.Message,
			},
		},
		Text: alert.Message,
	}

	// Set patient
	issue.Patient = &models.Reference{
		Reference: patientRef,
	}

	// Set identified time
	issue.IdentifiedDateTime = alert.Timestamp.Format(time.RFC3339)

	// Set detail
	issue.Detail = alert.Message

	// Set identifier
	issue.Identifier = append(issue.Identifier, models.Identifier{
		System: "urn:savegress:alert",
		Value:  alert.ID,
	})

	return issue
}

// VitalSignsBundle creates a FHIR Bundle of vital signs observations
func (c *FHIRConverter) VitalSignsBundle(measurements []Measurement, patientRef string) *models.Bundle {
	bundle := &models.Bundle{
		ResourceType: "Bundle",
		Type:         "collection",
	}

	for _, m := range measurements {
		obs := c.MeasurementToObservation(&m, patientRef)
		bundle.Entry = append(bundle.Entry, models.BundleEntry{
			Resource: obs,
		})
	}

	return bundle
}

// Helper methods

func (c *FHIRConverter) getCategoryCode(category DeviceCategory) string {
	switch category {
	case CategoryPulseOximeter, CategoryBloodPressure, CategoryThermometer:
		return "vital-signs"
	case CategoryGlucoseMeter, CategoryContinuousGlucose, CategoryINR:
		return "laboratory"
	case CategoryActivityMonitor, CategoryStrengthFitness:
		return "activity"
	case CategoryCardioVascular:
		return "procedure"
	default:
		return "exam"
	}
}

func (c *FHIRConverter) getCategoryMDCCode(category DeviceCategory) string {
	codes := map[DeviceCategory]string{
		CategoryPulseOximeter:      "65573",  // MDC_DEV_SPEC_PROFILE_PULS_OXIM
		CategoryBloodPressure:      "65574",  // MDC_DEV_SPEC_PROFILE_BP
		CategoryThermometer:        "65575",  // MDC_DEV_SPEC_PROFILE_TEMP
		CategoryWeighingScale:      "65576",  // MDC_DEV_SPEC_PROFILE_SCALE
		CategoryGlucoseMeter:       "65577",  // MDC_DEV_SPEC_PROFILE_GLUCOSE
		CategoryINR:                "65578",  // MDC_DEV_SPEC_PROFILE_INR
		CategoryBodyComposition:    "65579",  // MDC_DEV_SPEC_PROFILE_BCA
		CategoryPeakFlow:           "65580",  // MDC_DEV_SPEC_PROFILE_PEAK_FLOW
		CategoryCardioVascular:     "65581",  // MDC_DEV_SPEC_PROFILE_CV
		CategoryStrengthFitness:    "65582",  // MDC_DEV_SPEC_PROFILE_STRENGTH
		CategoryActivityMonitor:    "65583",  // MDC_DEV_SPEC_PROFILE_HF_ACTIVITY
		CategoryMedication:         "65584",  // MDC_DEV_SPEC_PROFILE_MEDICATION
		CategoryContinuousGlucose:  "65585",  // MDC_DEV_SPEC_PROFILE_CGM
		CategoryInsulinPump:        "65586",  // MDC_DEV_SPEC_PROFILE_INSULIN_PUMP
		CategorySleepApnea:         "65587",  // MDC_DEV_SPEC_PROFILE_SLEEP
		CategoryRespiratoryMonitor: "65588",  // MDC_DEV_SPEC_PROFILE_RESP
		CategoryIndependentLiving:  "65589",  // MDC_DEV_SPEC_PROFILE_AI_LIVING
	}

	if code, ok := codes[category]; ok {
		return code
	}
	return "65536" // Default device
}

func (c *FHIRConverter) mapToLOINC(code NomenclatureCode) string {
	// Map IEEE 11073 codes to LOINC codes
	mapping := map[NomenclatureCode]string{
		MDC_PULS_OXIM_SAT_O2:         "2708-6",  // Oxygen saturation
		MDC_PULS_OXIM_PULS_RATE:      "8867-4",  // Heart rate
		MDC_PULS_RATE_NON_INV:        "8867-4",  // Heart rate
		MDC_PRESS_BLD_NONINV_SYS:     "8480-6",  // Systolic BP
		MDC_PRESS_BLD_NONINV_DIA:     "8462-4",  // Diastolic BP
		MDC_PRESS_BLD_NONINV_MEAN:    "8478-0",  // Mean arterial pressure
		MDC_TEMP_BODY:                "8310-5",  // Body temperature
		MDC_TEMP_ORAL:                "8331-1",  // Oral temperature
		MDC_TEMP_AXILLA:              "8328-7",  // Axillary temperature
		MDC_TEMP_TYMP:                "8333-7",  // Tympanic temperature
		MDC_TEMP_RECT:                "8332-9",  // Rectal temperature
		MDC_MASS_BODY_ACTUAL:         "29463-7", // Body weight
		MDC_LEN_BODY_ACTUAL:          "8302-2",  // Body height
		MDC_RATIO_MASS_BODY_LEN_SQ:   "39156-5", // BMI
		MDC_CONC_GLU_CAPILLARY_WHOLEBLOOD: "41653-7", // Glucose blood
		MDC_CONC_GLU_GEN:             "2339-0",  // Glucose
		MDC_RESP_RATE:                "9279-1",  // Respiratory rate
		MDC_ECG_HEART_RATE:           "8867-4",  // Heart rate
		MDC_HF_STEPS:                 "55423-8", // Steps
		MDC_BODY_FAT:                 "41982-0", // Body fat percentage
	}

	return mapping[code]
}

// CreateVitalSignsPanel creates a FHIR vital signs panel observation
func (c *FHIRConverter) CreateVitalSignsPanel(measurements []Measurement, patientRef string) *models.Observation {
	panel := &models.Observation{
		ResourceType: models.ResourceTypeObservation,
		Status:       "final",
	}

	// Set code for vital signs panel
	panel.Code = models.CodeableConcept{
		Coding: []models.Coding{
			{
				System:  "http://loinc.org",
				Code:    "85353-1",
				Display: "Vital signs, weight, height, head circumference, oximetry and BMI panel",
			},
		},
	}

	// Set category
	panel.Category = append(panel.Category, models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/observation-category",
				Code:   "vital-signs",
			},
		},
	})

	// Set subject
	panel.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set effective time (use earliest measurement time)
	var earliest time.Time
	for _, m := range measurements {
		if earliest.IsZero() || m.Timestamp.Before(earliest) {
			earliest = m.Timestamp
		}
	}
	if !earliest.IsZero() {
		panel.EffectiveDateTime = earliest.Format(time.RFC3339)
	}

	// Add has-member references for each measurement
	for i, m := range measurements {
		panel.HasMember = append(panel.HasMember, models.Reference{
			Reference: fmt.Sprintf("Observation/measurement-%d", i),
		})

		// Also add as component
		info := NomenclatureRegistry[m.Code]
		unitSymbol := UnitRegistry[m.Unit]

		panel.Component = append(panel.Component, models.ObservationComponent{
			Code: models.CodeableConcept{
				Coding: []models.Coding{
					{
						System:  "urn:iso:std:iso:11073:10101",
						Code:    fmt.Sprintf("%d", m.Code),
						Display: info.Description,
					},
				},
			},
			ValueQuantity: &models.Quantity{
				Value: m.Value,
				Unit:  unitSymbol,
			},
		})
	}

	return panel
}

// BloodPressureObservation creates a FHIR blood pressure observation from systolic and diastolic measurements
func (c *FHIRConverter) BloodPressureObservation(systolic, diastolic, mean *Measurement, patientRef string) *models.Observation {
	obs := &models.Observation{
		ResourceType: models.ResourceTypeObservation,
		Status:       "final",
	}

	// Set code
	obs.Code = models.CodeableConcept{
		Coding: []models.Coding{
			{
				System:  "http://loinc.org",
				Code:    "85354-9",
				Display: "Blood pressure panel with all children optional",
			},
			{
				System:  "urn:iso:std:iso:11073:10101",
				Code:    "18948",
				Display: "Blood Pressure",
			},
		},
	}

	// Set category
	obs.Category = append(obs.Category, models.CodeableConcept{
		Coding: []models.Coding{
			{
				System: "http://terminology.hl7.org/CodeSystem/observation-category",
				Code:   "vital-signs",
			},
		},
	})

	// Set subject
	obs.Subject = &models.Reference{
		Reference: patientRef,
	}

	// Set effective time
	if systolic != nil {
		obs.EffectiveDateTime = systolic.Timestamp.Format(time.RFC3339)
	}

	// Add systolic component
	if systolic != nil {
		obs.Component = append(obs.Component, models.ObservationComponent{
			Code: models.CodeableConcept{
				Coding: []models.Coding{
					{
						System:  "http://loinc.org",
						Code:    "8480-6",
						Display: "Systolic blood pressure",
					},
				},
			},
			ValueQuantity: &models.Quantity{
				Value:  systolic.Value,
				Unit:   "mmHg",
				System: "http://unitsofmeasure.org",
				Code:   "mm[Hg]",
			},
		})
	}

	// Add diastolic component
	if diastolic != nil {
		obs.Component = append(obs.Component, models.ObservationComponent{
			Code: models.CodeableConcept{
				Coding: []models.Coding{
					{
						System:  "http://loinc.org",
						Code:    "8462-4",
						Display: "Diastolic blood pressure",
					},
				},
			},
			ValueQuantity: &models.Quantity{
				Value:  diastolic.Value,
				Unit:   "mmHg",
				System: "http://unitsofmeasure.org",
				Code:   "mm[Hg]",
			},
		})
	}

	// Add mean component if present
	if mean != nil {
		obs.Component = append(obs.Component, models.ObservationComponent{
			Code: models.CodeableConcept{
				Coding: []models.Coding{
					{
						System:  "http://loinc.org",
						Code:    "8478-0",
						Display: "Mean blood pressure",
					},
				},
			},
			ValueQuantity: &models.Quantity{
				Value:  mean.Value,
				Unit:   "mmHg",
				System: "http://unitsofmeasure.org",
				Code:   "mm[Hg]",
			},
		})
	}

	return obs
}
