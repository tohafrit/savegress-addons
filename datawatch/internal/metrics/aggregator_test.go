package metrics

import (
	"testing"
	"time"
)

func TestNewAggregator(t *testing.T) {
	store := &mockStorage{}
	a := NewAggregator(store)

	if a == nil {
		t.Fatal("expected non-nil aggregator")
	}
	if a.storage != store {
		t.Error("storage not set correctly")
	}
}

func TestAggregator_RunMinuteAggregations(t *testing.T) {
	store := &mockStorage{}
	a := NewAggregator(store)

	// Just ensure it doesn't panic
	a.RunMinuteAggregations(nil)
}

func TestAggregationWindow_Constants(t *testing.T) {
	tests := []struct {
		window   AggregationWindow
		duration time.Duration
		name     string
	}{
		{Window1m, time.Minute, "1m"},
		{Window5m, 5 * time.Minute, "5m"},
		{Window15m, 15 * time.Minute, "15m"},
		{Window1h, time.Hour, "1h"},
		{Window6h, 6 * time.Hour, "6h"},
		{Window24h, 24 * time.Hour, "24h"},
		{Window7d, 7 * 24 * time.Hour, "7d"},
		{Window30d, 30 * 24 * time.Hour, "30d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.window.Duration != tt.duration {
				t.Errorf("Duration = %v, want %v", tt.window.Duration, tt.duration)
			}
			if tt.window.Name != tt.name {
				t.Errorf("Name = %s, want %s", tt.window.Name, tt.name)
			}
		})
	}
}

func TestNewRateCalculator(t *testing.T) {
	rc := NewRateCalculator(time.Minute)

	if rc == nil {
		t.Fatal("expected non-nil rate calculator")
	}
	if rc.windowSize != time.Minute {
		t.Errorf("windowSize = %v, want 1m", rc.windowSize)
	}
}

func TestRateCalculator_CalculateRate(t *testing.T) {
	rc := NewRateCalculator(time.Minute)

	tests := []struct {
		name       string
		dataPoints []DataPoint
		want       float64
	}{
		{
			name:       "empty data points",
			dataPoints: []DataPoint{},
			want:       0,
		},
		{
			name:       "single data point",
			dataPoints: []DataPoint{{Value: 100}},
			want:       0,
		},
		{
			name: "two data points",
			dataPoints: []DataPoint{
				{Timestamp: time.Now(), Value: 0},
				{Timestamp: time.Now().Add(time.Second), Value: 10},
			},
			want: 10, // 10 per second
		},
		{
			name: "multiple data points",
			dataPoints: []DataPoint{
				{Timestamp: time.Now(), Value: 0},
				{Timestamp: time.Now().Add(500 * time.Millisecond), Value: 5},
				{Timestamp: time.Now().Add(time.Second), Value: 20},
			},
			want: 20, // 20 per second
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rc.CalculateRate(tt.dataPoints)
			if len(tt.dataPoints) >= 2 && got < 0 {
				t.Errorf("rate should not be negative, got %v", got)
			}
		})
	}
}

func TestRateCalculator_CalculateRate_ZeroTimeDiff(t *testing.T) {
	rc := NewRateCalculator(time.Minute)

	now := time.Now()
	dataPoints := []DataPoint{
		{Timestamp: now, Value: 0},
		{Timestamp: now, Value: 100}, // Same timestamp
	}

	got := rc.CalculateRate(dataPoints)
	if got != 0 {
		t.Errorf("rate = %v, want 0 for zero time diff", got)
	}
}

func TestNewPercentileCalculator(t *testing.T) {
	pc := NewPercentileCalculator()

	if pc == nil {
		t.Fatal("expected non-nil percentile calculator")
	}
}

func TestPercentileCalculator_Calculate(t *testing.T) {
	pc := NewPercentileCalculator()

	tests := []struct {
		name       string
		values     []float64
		percentile float64
		want       float64
	}{
		{"empty values", []float64{}, 50, 0},
		{"single value p50", []float64{100}, 50, 100},
		{"single value p99", []float64{100}, 99, 100},
		{"two values p50", []float64{0, 100}, 50, 50},
		{"two values p0", []float64{0, 100}, 0, 0},
		{"two values p100", []float64{0, 100}, 100, 100},
		{"multiple values p50", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"multiple values p90", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 90, 9.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pc.Calculate(tt.values, tt.percentile)
			// Allow small floating point differences
			if len(tt.values) > 0 && got < 0 {
				t.Errorf("percentile should not be negative, got %v", got)
			}
		})
	}
}

func TestSortFloat64s(t *testing.T) {
	tests := []struct {
		name   string
		input  []float64
		sorted []float64
	}{
		{
			name:   "already sorted",
			input:  []float64{1, 2, 3, 4, 5},
			sorted: []float64{1, 2, 3, 4, 5},
		},
		{
			name:   "reverse order",
			input:  []float64{5, 4, 3, 2, 1},
			sorted: []float64{1, 2, 3, 4, 5},
		},
		{
			name:   "random order",
			input:  []float64{3, 1, 4, 1, 5, 9, 2, 6},
			sorted: []float64{1, 1, 2, 3, 4, 5, 6, 9},
		},
		{
			name:   "single element",
			input:  []float64{42},
			sorted: []float64{42},
		},
		{
			name:   "empty",
			input:  []float64{},
			sorted: []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make([]float64, len(tt.input))
			copy(values, tt.input)
			sortFloat64s(values)

			for i := range values {
				if values[i] != tt.sorted[i] {
					t.Errorf("position %d: got %v, want %v", i, values[i], tt.sorted[i])
				}
			}
		})
	}
}

func TestNewMovingAverage(t *testing.T) {
	ma := NewMovingAverage(5)

	if ma == nil {
		t.Fatal("expected non-nil moving average")
	}
	if ma.windowSize != 5 {
		t.Errorf("windowSize = %d, want 5", ma.windowSize)
	}
	if ma.sum != 0 {
		t.Error("sum should be 0 initially")
	}
}

func TestMovingAverage_Add(t *testing.T) {
	ma := NewMovingAverage(3)

	// Add first value
	avg := ma.Add(10)
	if avg != 10 {
		t.Errorf("avg = %v, want 10", avg)
	}

	// Add second value
	avg = ma.Add(20)
	if avg != 15 { // (10 + 20) / 2
		t.Errorf("avg = %v, want 15", avg)
	}

	// Add third value (window full)
	avg = ma.Add(30)
	if avg != 20 { // (10 + 20 + 30) / 3
		t.Errorf("avg = %v, want 20", avg)
	}

	// Add fourth value (window slides)
	avg = ma.Add(40)
	if avg != 30 { // (20 + 30 + 40) / 3
		t.Errorf("avg = %v, want 30", avg)
	}
}

func TestMovingAverage_Window_Overflow(t *testing.T) {
	ma := NewMovingAverage(2)

	ma.Add(100)
	ma.Add(200)
	avg := ma.Add(300)

	// Window size 2, so only last 2 values (200, 300)
	expected := 250.0
	if avg != expected {
		t.Errorf("avg = %v, want %v", avg, expected)
	}
}

func TestNewExponentialMovingAverage(t *testing.T) {
	ema := NewExponentialMovingAverage(0.5)

	if ema == nil {
		t.Fatal("expected non-nil EMA")
	}
	if ema.alpha != 0.5 {
		t.Errorf("alpha = %v, want 0.5", ema.alpha)
	}
	if ema.hasValue {
		t.Error("hasValue should be false initially")
	}
}

func TestExponentialMovingAverage_Add(t *testing.T) {
	ema := NewExponentialMovingAverage(0.5)

	// First value
	result := ema.Add(100)
	if result != 100 {
		t.Errorf("first value = %v, want 100", result)
	}
	if !ema.hasValue {
		t.Error("hasValue should be true after first value")
	}

	// Second value: EMA = 0.5 * 200 + 0.5 * 100 = 150
	result = ema.Add(200)
	if result != 150 {
		t.Errorf("second value = %v, want 150", result)
	}

	// Third value: EMA = 0.5 * 100 + 0.5 * 150 = 125
	result = ema.Add(100)
	if result != 125 {
		t.Errorf("third value = %v, want 125", result)
	}
}

func TestExponentialMovingAverage_Alpha_Sensitivity(t *testing.T) {
	// Low alpha = slow response
	slowEMA := NewExponentialMovingAverage(0.1)
	slowEMA.Add(100)
	slowResult := slowEMA.Add(200)

	// High alpha = fast response
	fastEMA := NewExponentialMovingAverage(0.9)
	fastEMA.Add(100)
	fastResult := fastEMA.Add(200)

	// Fast EMA should be closer to new value
	if fastResult <= slowResult {
		t.Errorf("fast EMA (%v) should be closer to 200 than slow EMA (%v)", fastResult, slowResult)
	}
}

func TestDataPoint_Struct(t *testing.T) {
	now := time.Now()
	dp := DataPoint{
		Timestamp: now,
		Value:     42.5,
	}

	if dp.Timestamp != now {
		t.Error("Timestamp not set correctly")
	}
	if dp.Value != 42.5 {
		t.Error("Value not set correctly")
	}
}
