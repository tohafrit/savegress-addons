package ieee11073

import (
	"time"
)

// IEEE 11073 Personal Health Device (PHD) Communication types
// Based on ISO/IEEE 11073-10101 (Nomenclature) and 11073-20601 (Protocol)

// DeviceCategory represents IEEE 11073 device categories
type DeviceCategory string

const (
	// Vital Signs Monitors
	CategoryPulseOximeter       DeviceCategory = "pulse_oximeter"        // IEEE 11073-10404
	CategoryBloodPressure       DeviceCategory = "blood_pressure"        // IEEE 11073-10407
	CategoryThermometer         DeviceCategory = "thermometer"           // IEEE 11073-10408
	CategoryWeighingScale       DeviceCategory = "weighing_scale"        // IEEE 11073-10415
	CategoryGlucoseMeter        DeviceCategory = "glucose_meter"         // IEEE 11073-10417
	CategoryINR                 DeviceCategory = "inr_monitor"           // IEEE 11073-10418
	CategoryBodyComposition     DeviceCategory = "body_composition"      // IEEE 11073-10420
	CategoryPeakFlow            DeviceCategory = "peak_flow"             // IEEE 11073-10421
	CategoryCardioVascular      DeviceCategory = "cardiovascular"        // IEEE 11073-10441
	CategoryStrengthFitness     DeviceCategory = "strength_fitness"      // IEEE 11073-10442
	CategoryActivityMonitor     DeviceCategory = "activity_monitor"      // IEEE 11073-10471
	CategoryMedication          DeviceCategory = "medication_monitor"    // IEEE 11073-10472
	CategoryContinuousGlucose   DeviceCategory = "continuous_glucose"    // IEEE 11073-10425
	CategoryInsulinPump         DeviceCategory = "insulin_pump"          // IEEE 11073-10419
	CategorySleepApnea          DeviceCategory = "sleep_apnea"           // IEEE 11073-10424
	CategoryRespiratoryMonitor  DeviceCategory = "respiratory_monitor"   // IEEE 11073-10422
	CategoryIndependentLiving   DeviceCategory = "independent_living"    // IEEE 11073-10473
)

// NomenclatureCode represents IEEE 11073-10101 nomenclature codes
type NomenclatureCode uint32

// Vital Signs Nomenclature Codes
const (
	// Physiological measurements
	MDC_PULS_OXIM_SAT_O2         NomenclatureCode = 19384 // SpO2
	MDC_PULS_OXIM_PULS_RATE      NomenclatureCode = 18458 // Pulse rate from SpO2
	MDC_PULS_RATE_NON_INV        NomenclatureCode = 18474 // Non-invasive pulse rate
	MDC_PRESS_BLD_NONINV_SYS     NomenclatureCode = 18949 // Systolic BP
	MDC_PRESS_BLD_NONINV_DIA     NomenclatureCode = 18950 // Diastolic BP
	MDC_PRESS_BLD_NONINV_MEAN    NomenclatureCode = 18951 // Mean BP
	MDC_TEMP_BODY                NomenclatureCode = 19292 // Body temperature
	MDC_TEMP_TYMP                NomenclatureCode = 19320 // Tympanic temperature
	MDC_TEMP_ORAL                NomenclatureCode = 19308 // Oral temperature
	MDC_TEMP_AXILLA              NomenclatureCode = 19312 // Axillary temperature
	MDC_TEMP_RECT                NomenclatureCode = 19316 // Rectal temperature
	MDC_MASS_BODY_ACTUAL         NomenclatureCode = 57664 // Body weight
	MDC_LEN_BODY_ACTUAL          NomenclatureCode = 57668 // Body height
	MDC_RATIO_MASS_BODY_LEN_SQ   NomenclatureCode = 57680 // BMI
	MDC_CONC_GLU_CAPILLARY_WHOLEBLOOD NomenclatureCode = 29112 // Capillary glucose
	MDC_CONC_GLU_GEN             NomenclatureCode = 28948 // General glucose
	MDC_CONC_GLU_INTERSTITIAL    NomenclatureCode = 29116 // Interstitial glucose (CGM)
	MDC_INR                      NomenclatureCode = 28761 // INR
	MDC_COAG_TIME_PT             NomenclatureCode = 28745 // Prothrombin time

	// Cardiovascular
	MDC_ECG_HEART_RATE           NomenclatureCode = 16770 // ECG heart rate
	MDC_ECG_HEART_RATE_INSTANT   NomenclatureCode = 16778 // Instantaneous heart rate
	MDC_ECG_AMPL_ST              NomenclatureCode = 16810 // ST amplitude
	MDC_ECG_TIME_PD_QT           NomenclatureCode = 16824 // QT interval
	MDC_ECG_TIME_PD_QTc          NomenclatureCode = 16828 // Corrected QT interval

	// Respiratory
	MDC_RESP_RATE                NomenclatureCode = 20490 // Respiratory rate
	MDC_AWAY_RESP_RATE           NomenclatureCode = 20498 // Airway respiratory rate
	MDC_FLOW_AWAY_EXP_FORCED_PEAK NomenclatureCode = 20636 // Peak expiratory flow
	MDC_VOL_AWAY_EXP_FORCED      NomenclatureCode = 20584 // Forced expiratory volume

	// Activity
	MDC_HF_ACT_WALK              NomenclatureCode = 65600 // Walking activity
	MDC_HF_ACT_RUN               NomenclatureCode = 65604 // Running activity
	MDC_HF_DISTANCE              NomenclatureCode = 65616 // Distance traveled
	MDC_HF_CAL_ENERGY            NomenclatureCode = 65620 // Calories burned
	MDC_HF_STEPS                 NomenclatureCode = 65624 // Step count
	MDC_HF_SLEEP                 NomenclatureCode = 65632 // Sleep duration

	// Body Composition
	MDC_BODY_FAT                 NomenclatureCode = 57696 // Body fat percentage
	MDC_MASS_BODY_FAT            NomenclatureCode = 57700 // Body fat mass
	MDC_MASS_BODY_LEAN           NomenclatureCode = 57704 // Lean body mass
	MDC_BODY_WATER               NomenclatureCode = 57708 // Body water percentage
	MDC_MASS_BODY_MUSCLE         NomenclatureCode = 57712 // Muscle mass
	MDC_MASS_BODY_BONE           NomenclatureCode = 57716 // Bone mass
)

// Unit codes from IEEE 11073-10101
type UnitCode uint32

const (
	MDC_DIM_PERCENT              UnitCode = 544  // Percent (%)
	MDC_DIM_BEAT_PER_MIN         UnitCode = 2720 // Beats per minute
	MDC_DIM_RESP_PER_MIN         UnitCode = 2784 // Breaths per minute
	MDC_DIM_MMHG                 UnitCode = 3872 // mmHg
	MDC_DIM_KILO_G               UnitCode = 1731 // Kilogram
	MDC_DIM_MILLI_G              UnitCode = 1729 // Milligram
	MDC_DIM_CENTI_M              UnitCode = 1297 // Centimeter
	MDC_DIM_MILLI_M              UnitCode = 1298 // Millimeter
	MDC_DIM_DEGC                 UnitCode = 6048 // Degrees Celsius
	MDC_DIM_FAHR                 UnitCode = 4416 // Degrees Fahrenheit
	MDC_DIM_MILLI_G_PER_DL       UnitCode = 2130 // mg/dL
	MDC_DIM_MILLI_MOLE_PER_L     UnitCode = 4722 // mmol/L
	MDC_DIM_INTL_UNIT            UnitCode = 5472 // International Unit
	MDC_DIM_SEC                  UnitCode = 2176 // Seconds
	MDC_DIM_MILLI_SEC            UnitCode = 2177 // Milliseconds
	MDC_DIM_MIN                  UnitCode = 2208 // Minutes
	MDC_DIM_HR                   UnitCode = 2240 // Hours
	MDC_DIM_STEP                 UnitCode = 6976 // Steps
	MDC_DIM_KILO_CAL             UnitCode = 6496 // Kilocalories
	MDC_DIM_KILO_M               UnitCode = 1283 // Kilometers
	MDC_DIM_M                    UnitCode = 1280 // Meters
	MDC_DIM_L_PER_MIN            UnitCode = 3170 // Liters per minute
	MDC_DIM_MILLI_L              UnitCode = 3122 // Milliliters
)

// DeviceSystemID uniquely identifies a device
type DeviceSystemID struct {
	ID           string
	Manufacturer string
	Model        string
	SerialNumber string
	FirmwareRev  string
}

// DeviceStatus represents the operational status of a device
type DeviceStatus string

const (
	StatusOperating     DeviceStatus = "operating"
	StatusNotOperating  DeviceStatus = "not_operating"
	StatusLowBattery    DeviceStatus = "low_battery"
	StatusError         DeviceStatus = "error"
	StatusCalibrating   DeviceStatus = "calibrating"
	StatusInitializing  DeviceStatus = "initializing"
)

// Measurement represents a device measurement
type Measurement struct {
	ID              string
	DeviceID        string
	Category        DeviceCategory
	Code            NomenclatureCode
	Value           float64
	Unit            UnitCode
	Timestamp       time.Time
	Status          MeasurementStatus
	Accuracy        *float64
	LowerRange      *float64
	UpperRange      *float64
	Supplemental    map[string]interface{}
}

// MeasurementStatus indicates the reliability of the measurement
type MeasurementStatus string

const (
	MeasStatusValid              MeasurementStatus = "valid"
	MeasStatusQuestionable       MeasurementStatus = "questionable"
	MeasStatusNotAvailable       MeasurementStatus = "not_available"
	MeasStatusOverflow           MeasurementStatus = "overflow"
	MeasStatusUnderflow          MeasurementStatus = "underflow"
	MeasStatusCalibrating        MeasurementStatus = "calibrating"
	MeasStatusNoData             MeasurementStatus = "no_data"
	MeasStatusMeasurementOngoing MeasurementStatus = "measurement_ongoing"
)

// APDU types for IEEE 11073-20601 protocol
type APDUType uint16

const (
	APDUAssociationRequest      APDUType = 0xE200
	APDUAssociationResponse     APDUType = 0xE300
	APDUAssociationRelease      APDUType = 0xE400
	APDUAssociationAbort        APDUType = 0xE500
	APDUPresentationData        APDUType = 0xE700
)

// Association states
type AssociationState string

const (
	StateDisconnected     AssociationState = "disconnected"
	StateConnecting       AssociationState = "connecting"
	StateConnected        AssociationState = "connected"
	StateAssociating      AssociationState = "associating"
	StateAssociated       AssociationState = "associated"
	StateConfiguring      AssociationState = "configuring"
	StateOperating        AssociationState = "operating"
	StateDisassociating   AssociationState = "disassociating"
)

// DeviceConfiguration holds device configuration data
type DeviceConfiguration struct {
	DeviceID        DeviceSystemID
	Category        DeviceCategory
	TransportType   TransportType
	Capabilities    []Capability
	SupportedMeasurements []NomenclatureCode
	Configuration   map[string]interface{}
}

// TransportType represents the communication transport
type TransportType string

const (
	TransportBluetooth    TransportType = "bluetooth"
	TransportBluetoothLE  TransportType = "bluetooth_le"
	TransportUSB          TransportType = "usb"
	TransportZigbee       TransportType = "zigbee"
	TransportWiFi         TransportType = "wifi"
	TransportNFC          TransportType = "nfc"
)

// Capability represents device capabilities
type Capability string

const (
	CapContinuousMonitoring Capability = "continuous_monitoring"
	CapEpisodicMeasurement  Capability = "episodic_measurement"
	CapStoredData           Capability = "stored_data"
	CapRealTimeData         Capability = "real_time_data"
	CapAlerts               Capability = "alerts"
	CapRemoteControl        Capability = "remote_control"
)

// Alert represents a device alert
type Alert struct {
	ID          string
	DeviceID    string
	Type        AlertType
	Priority    AlertPriority
	Code        NomenclatureCode
	Message     string
	Timestamp   time.Time
	Acknowledged bool
}

// AlertType categorizes alerts
type AlertType string

const (
	AlertTypeTechnical    AlertType = "technical"
	AlertTypePhysiological AlertType = "physiological"
	AlertTypeBattery      AlertType = "battery"
	AlertTypeCalibration  AlertType = "calibration"
)

// AlertPriority indicates alert urgency
type AlertPriority string

const (
	AlertPriorityHigh   AlertPriority = "high"
	AlertPriorityMedium AlertPriority = "medium"
	AlertPriorityLow    AlertPriority = "low"
)

// NomenclatureInfo provides human-readable information about a nomenclature code
type NomenclatureInfo struct {
	Code        NomenclatureCode
	RefID       string
	Description string
	Unit        UnitCode
	Category    DeviceCategory
}

// NomenclatureRegistry maps codes to their descriptions
var NomenclatureRegistry = map[NomenclatureCode]NomenclatureInfo{
	MDC_PULS_OXIM_SAT_O2: {
		Code:        MDC_PULS_OXIM_SAT_O2,
		RefID:       "MDC_PULS_OXIM_SAT_O2",
		Description: "Oxygen Saturation (SpO2)",
		Unit:        MDC_DIM_PERCENT,
		Category:    CategoryPulseOximeter,
	},
	MDC_PULS_OXIM_PULS_RATE: {
		Code:        MDC_PULS_OXIM_PULS_RATE,
		RefID:       "MDC_PULS_OXIM_PULS_RATE",
		Description: "Pulse Rate from Pulse Oximeter",
		Unit:        MDC_DIM_BEAT_PER_MIN,
		Category:    CategoryPulseOximeter,
	},
	MDC_PRESS_BLD_NONINV_SYS: {
		Code:        MDC_PRESS_BLD_NONINV_SYS,
		RefID:       "MDC_PRESS_BLD_NONINV_SYS",
		Description: "Systolic Blood Pressure (Non-invasive)",
		Unit:        MDC_DIM_MMHG,
		Category:    CategoryBloodPressure,
	},
	MDC_PRESS_BLD_NONINV_DIA: {
		Code:        MDC_PRESS_BLD_NONINV_DIA,
		RefID:       "MDC_PRESS_BLD_NONINV_DIA",
		Description: "Diastolic Blood Pressure (Non-invasive)",
		Unit:        MDC_DIM_MMHG,
		Category:    CategoryBloodPressure,
	},
	MDC_PRESS_BLD_NONINV_MEAN: {
		Code:        MDC_PRESS_BLD_NONINV_MEAN,
		RefID:       "MDC_PRESS_BLD_NONINV_MEAN",
		Description: "Mean Arterial Pressure (Non-invasive)",
		Unit:        MDC_DIM_MMHG,
		Category:    CategoryBloodPressure,
	},
	MDC_TEMP_BODY: {
		Code:        MDC_TEMP_BODY,
		RefID:       "MDC_TEMP_BODY",
		Description: "Body Temperature",
		Unit:        MDC_DIM_DEGC,
		Category:    CategoryThermometer,
	},
	MDC_MASS_BODY_ACTUAL: {
		Code:        MDC_MASS_BODY_ACTUAL,
		RefID:       "MDC_MASS_BODY_ACTUAL",
		Description: "Body Weight",
		Unit:        MDC_DIM_KILO_G,
		Category:    CategoryWeighingScale,
	},
	MDC_CONC_GLU_CAPILLARY_WHOLEBLOOD: {
		Code:        MDC_CONC_GLU_CAPILLARY_WHOLEBLOOD,
		RefID:       "MDC_CONC_GLU_CAPILLARY_WHOLEBLOOD",
		Description: "Blood Glucose (Capillary)",
		Unit:        MDC_DIM_MILLI_G_PER_DL,
		Category:    CategoryGlucoseMeter,
	},
	MDC_ECG_HEART_RATE: {
		Code:        MDC_ECG_HEART_RATE,
		RefID:       "MDC_ECG_HEART_RATE",
		Description: "ECG Heart Rate",
		Unit:        MDC_DIM_BEAT_PER_MIN,
		Category:    CategoryCardioVascular,
	},
	MDC_RESP_RATE: {
		Code:        MDC_RESP_RATE,
		RefID:       "MDC_RESP_RATE",
		Description: "Respiratory Rate",
		Unit:        MDC_DIM_RESP_PER_MIN,
		Category:    CategoryRespiratoryMonitor,
	},
	MDC_HF_STEPS: {
		Code:        MDC_HF_STEPS,
		RefID:       "MDC_HF_STEPS",
		Description: "Step Count",
		Unit:        MDC_DIM_STEP,
		Category:    CategoryActivityMonitor,
	},
}

// UnitRegistry maps unit codes to their symbols
var UnitRegistry = map[UnitCode]string{
	MDC_DIM_PERCENT:          "%",
	MDC_DIM_BEAT_PER_MIN:     "bpm",
	MDC_DIM_RESP_PER_MIN:     "rpm",
	MDC_DIM_MMHG:             "mmHg",
	MDC_DIM_KILO_G:           "kg",
	MDC_DIM_MILLI_G:          "mg",
	MDC_DIM_CENTI_M:          "cm",
	MDC_DIM_MILLI_M:          "mm",
	MDC_DIM_DEGC:             "°C",
	MDC_DIM_FAHR:             "°F",
	MDC_DIM_MILLI_G_PER_DL:   "mg/dL",
	MDC_DIM_MILLI_MOLE_PER_L: "mmol/L",
	MDC_DIM_INTL_UNIT:        "IU",
	MDC_DIM_SEC:              "s",
	MDC_DIM_MILLI_SEC:        "ms",
	MDC_DIM_MIN:              "min",
	MDC_DIM_HR:               "hr",
	MDC_DIM_STEP:             "steps",
	MDC_DIM_KILO_CAL:         "kcal",
	MDC_DIM_KILO_M:           "km",
	MDC_DIM_M:                "m",
	MDC_DIM_L_PER_MIN:        "L/min",
	MDC_DIM_MILLI_L:          "mL",
}
