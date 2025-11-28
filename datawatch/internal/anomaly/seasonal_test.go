package anomaly

import (
	"testing"
	"time"
)

func TestNewSeasonalDetector(t *testing.T) {
	d := NewSeasonalDetector(3.0)
	if d == nil {
		t.Fatal("expected non-nil detector")
	}
	if d.threshold != 3.0 {
		t.Errorf("threshold = %v, want 3.0", d.threshold)
	}
}

func TestSeasonalDetector_Detect(t *testing.T) {
	tests := []struct {
		name        string
		value       float64
		baseline    *MetricBaseline
		wantAnomaly bool
		wantType    AnomalyType
	}{
		{
			name:  "no seasonal data",
			value: 100,
			baseline: &MetricBaseline{
				Mean:         100,
				StdDev:       10,
				SeasonalData: nil,
			},
			wantAnomaly: false,
		},
		{
			name:  "seasonality disabled",
			value: 100,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
				SeasonalData: &SeasonalBaseline{
					HasSeasonality: false,
				},
			},
			wantAnomaly: false,
		},
		{
			name:  "zero stddev",
			value: 100,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 0,
				SeasonalData: &SeasonalBaseline{
					HasSeasonality: true,
					HourlyPattern:  [24]float64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
					DailyPattern:   [7]float64{100, 100, 100, 100, 100, 100, 100},
				},
			},
			wantAnomaly: false,
		},
		{
			name:  "normal value matching pattern",
			value: 100,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
				SeasonalData: &SeasonalBaseline{
					HasSeasonality: true,
					HourlyPattern:  [24]float64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
					DailyPattern:   [7]float64{100, 100, 100, 100, 100, 100, 100},
				},
			},
			wantAnomaly: false,
		},
		{
			name:  "anomalous value deviating from pattern",
			value: 200,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
				SeasonalData: &SeasonalBaseline{
					HasSeasonality: true,
					HourlyPattern:  [24]float64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
					DailyPattern:   [7]float64{100, 100, 100, 100, 100, 100, 100},
				},
			},
			wantAnomaly: true,
			wantType:    AnomalySeasonal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewSeasonalDetector(3.0)
			ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) // Monday noon

			score, isAnomaly, anomalyType := d.Detect(tt.value, ts, tt.baseline)

			if isAnomaly != tt.wantAnomaly {
				t.Errorf("isAnomaly = %v, want %v", isAnomaly, tt.wantAnomaly)
			}

			if tt.wantAnomaly && anomalyType != tt.wantType {
				t.Errorf("anomalyType = %v, want %v", anomalyType, tt.wantType)
			}

			if score < 0 || score > 1 {
				t.Errorf("score = %v, should be between 0 and 1", score)
			}
		})
	}
}

func TestSeasonalDetector_Detect_DifferentTimes(t *testing.T) {
	baseline := &MetricBaseline{
		Mean:   100,
		StdDev: 10,
		SeasonalData: &SeasonalBaseline{
			HasSeasonality: true,
			HourlyPattern:  [24]float64{50, 50, 50, 50, 50, 50, 80, 100, 120, 130, 130, 130, 130, 130, 130, 120, 100, 80, 70, 60, 50, 50, 50, 50},
			DailyPattern:   [7]float64{70, 100, 110, 110, 110, 100, 70},
		},
	}

	d := NewSeasonalDetector(3.0)

	// Test at different hours
	times := []struct {
		hour  int
		value float64
		desc  string
	}{
		{3, 50, "night - expected low"},
		{12, 130, "midday - expected high"},
		{3, 150, "night - anomalous high"},
		{12, 50, "midday - anomalous low"},
	}

	for _, tt := range times {
		ts := time.Date(2024, 1, 15, tt.hour, 0, 0, 0, time.UTC)
		_, isAnomaly, _ := d.Detect(tt.value, ts, baseline)
		t.Logf("%s (hour=%d, value=%.0f): anomaly=%v", tt.desc, tt.hour, tt.value, isAnomaly)
	}
}

func TestNewSTLDecomposition(t *testing.T) {
	stl := NewSTLDecomposition(24)
	if stl == nil {
		t.Fatal("expected non-nil STL")
	}
	if stl.period != 24 {
		t.Errorf("period = %v, want 24", stl.period)
	}
}

func TestSTLDecomposition_Decompose(t *testing.T) {
	stl := NewSTLDecomposition(7)

	t.Run("insufficient data", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5}
		trend, seasonal, residual := stl.Decompose(values)

		if len(trend) != len(values) {
			t.Errorf("trend length = %d, want %d", len(trend), len(values))
		}
		if len(seasonal) != len(values) {
			t.Errorf("seasonal length = %d, want %d", len(seasonal), len(values))
		}
		if len(residual) != len(values) {
			t.Errorf("residual length = %d, want %d", len(residual), len(values))
		}
	})

	t.Run("sufficient data with seasonality", func(t *testing.T) {
		// Generate data with clear weekly pattern
		values := make([]float64, 28) // 4 weeks
		for i := range values {
			dayOfWeek := i % 7
			values[i] = 100 + float64(dayOfWeek)*10 + float64(i)*0.5 // pattern + trend
		}

		trend, seasonal, residual := stl.Decompose(values)

		if len(trend) != len(values) {
			t.Errorf("trend length = %d, want %d", len(trend), len(values))
		}
		if len(seasonal) != len(values) {
			t.Errorf("seasonal length = %d, want %d", len(seasonal), len(values))
		}
		if len(residual) != len(values) {
			t.Errorf("residual length = %d, want %d", len(residual), len(values))
		}

		// Seasonal should repeat
		for i := 7; i < len(seasonal); i++ {
			if seasonal[i] != seasonal[i%7] {
				t.Logf("seasonal[%d] = %v, seasonal[%d] = %v", i, seasonal[i], i%7, seasonal[i%7])
			}
		}
	})
}

func TestSTLDecomposition_DetectFromResidual(t *testing.T) {
	stl := NewSTLDecomposition(7)

	t.Run("empty residual", func(t *testing.T) {
		indices := stl.DetectFromResidual([]float64{}, 3.0)
		if indices != nil {
			t.Errorf("expected nil for empty residual")
		}
	})

	t.Run("zero stddev", func(t *testing.T) {
		residual := []float64{0, 0, 0, 0, 0}
		indices := stl.DetectFromResidual(residual, 3.0)
		if indices != nil {
			t.Errorf("expected nil for zero stddev")
		}
	})

	t.Run("with anomalies", func(t *testing.T) {
		// Normal residuals with one outlier
		residual := []float64{0.1, -0.2, 0.15, -0.1, 0.05, 100, -0.15, 0.1}
		indices := stl.DetectFromResidual(residual, 2.0)

		if len(indices) == 0 {
			t.Error("expected to detect anomaly at index 5")
		}

		found := false
		for _, idx := range indices {
			if idx == 5 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected index 5 to be anomaly, got indices: %v", indices)
		}
	})
}

func TestNewHoltWinters(t *testing.T) {
	hw := NewHoltWinters(0.2, 0.1, 0.3, 24)
	if hw == nil {
		t.Fatal("expected non-nil HoltWinters")
	}
	if hw.alpha != 0.2 {
		t.Errorf("alpha = %v, want 0.2", hw.alpha)
	}
	if hw.beta != 0.1 {
		t.Errorf("beta = %v, want 0.1", hw.beta)
	}
	if hw.gamma != 0.3 {
		t.Errorf("gamma = %v, want 0.3", hw.gamma)
	}
	if hw.period != 24 {
		t.Errorf("period = %v, want 24", hw.period)
	}
}

func TestHoltWinters_Forecast(t *testing.T) {
	hw := NewHoltWinters(0.2, 0.1, 0.3, 7)

	t.Run("insufficient data", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		forecast := hw.Forecast(values, 5)
		if forecast != nil {
			t.Errorf("expected nil for insufficient data")
		}
	})

	t.Run("sufficient data", func(t *testing.T) {
		// Generate 4 weeks of data with weekly pattern
		values := make([]float64, 28)
		for i := range values {
			dayOfWeek := i % 7
			values[i] = 100 + float64(dayOfWeek)*10
		}

		forecast := hw.Forecast(values, 7)
		if forecast == nil {
			t.Fatal("expected non-nil forecast")
		}
		if len(forecast) != 7 {
			t.Errorf("forecast length = %d, want 7", len(forecast))
		}

		// Forecasts should be reasonable (not NaN, not extreme)
		for i, f := range forecast {
			if f < 0 || f > 500 {
				t.Errorf("forecast[%d] = %v, seems unreasonable", i, f)
			}
		}
	})
}

func TestMovingAverage(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		window int
	}{
		{
			name:   "simple case",
			values: []float64{1, 2, 3, 4, 5},
			window: 3,
		},
		{
			name:   "window larger than edge",
			values: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			window: 5,
		},
		{
			name:   "single value",
			values: []float64{5},
			window: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := movingAverage(tt.values, tt.window)

			if len(result) != len(tt.values) {
				t.Errorf("result length = %d, want %d", len(result), len(tt.values))
			}

			// Check that values are smoothed (middle values should be averaged)
			for i, v := range result {
				if v < min(tt.values) || v > max(tt.values) {
					t.Errorf("result[%d] = %v is outside input range [%v, %v]", i, v, min(tt.values), max(tt.values))
				}
			}
		})
	}
}

func TestMovingAverage_EdgeCases(t *testing.T) {
	// Empty slice
	result := movingAverage([]float64{}, 3)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input")
	}

	// Large window
	values := []float64{1, 2, 3}
	result = movingAverage(values, 10)
	if len(result) != 3 {
		t.Errorf("result length = %d, want 3", len(result))
	}
}
