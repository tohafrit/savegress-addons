package anomaly

import (
	"testing"
)

func TestNewStatisticalDetector(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
	}{
		{"low sensitivity", 4.0},
		{"medium sensitivity", 3.0},
		{"high sensitivity", 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewStatisticalDetector(tt.threshold)
			if d == nil {
				t.Fatal("expected non-nil detector")
			}
			if d.threshold != tt.threshold {
				t.Errorf("threshold = %v, want %v", d.threshold, tt.threshold)
			}
		})
	}
}

func TestStatisticalDetector_Detect(t *testing.T) {
	tests := []struct {
		name          string
		value         float64
		baseline      *MetricBaseline
		threshold     float64
		wantAnomaly   bool
		wantType      AnomalyType
		wantScoreGt   float64
	}{
		{
			name:  "normal value within range",
			value: 100,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: false,
		},
		{
			name:  "spike anomaly",
			value: 150,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalySpike,
			wantScoreGt: 0.5,
		},
		{
			name:  "drop anomaly",
			value: 50,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalyDrop,
			wantScoreGt: 0.5,
		},
		{
			name:  "zero stddev returns no anomaly",
			value: 150,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 0,
			},
			threshold:   3.0,
			wantAnomaly: false,
		},
		{
			name:  "value exactly at threshold boundary",
			value: 130,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: false, // exactly 3 stddev is not anomaly (> not >=)
		},
		{
			name:  "value just beyond threshold",
			value: 131,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalySpike,
		},
		{
			name:  "negative value drop",
			value: -50,
			baseline: &MetricBaseline{
				Mean:   100,
				StdDev: 10,
			},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalyDrop,
			wantScoreGt: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewStatisticalDetector(tt.threshold)
			score, isAnomaly, anomalyType := d.Detect(tt.value, tt.baseline)

			if isAnomaly != tt.wantAnomaly {
				t.Errorf("isAnomaly = %v, want %v", isAnomaly, tt.wantAnomaly)
			}

			if tt.wantAnomaly {
				if anomalyType != tt.wantType {
					t.Errorf("anomalyType = %v, want %v", anomalyType, tt.wantType)
				}
				if tt.wantScoreGt > 0 && score <= tt.wantScoreGt {
					t.Errorf("score = %v, want > %v", score, tt.wantScoreGt)
				}
			}

			// Score should always be between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("score = %v, should be between 0 and 1", score)
			}
		})
	}
}

func TestStatisticalDetector_DetectWithMAD(t *testing.T) {
	tests := []struct {
		name        string
		value       float64
		values      []float64
		threshold   float64
		wantAnomaly bool
		wantType    AnomalyType
	}{
		{
			name:        "insufficient data",
			value:       100,
			values:      []float64{1, 2},
			threshold:   3.0,
			wantAnomaly: false,
		},
		{
			name:        "normal value",
			value:       50,
			values:      []float64{45, 50, 55, 48, 52, 49, 51, 47, 53, 50},
			threshold:   3.0,
			wantAnomaly: false,
		},
		{
			name:        "outlier spike",
			value:       200,
			values:      []float64{45, 50, 55, 48, 52, 49, 51, 47, 53, 50},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalySpike,
		},
		{
			name:        "outlier drop",
			value:       -50,
			values:      []float64{45, 50, 55, 48, 52, 49, 51, 47, 53, 50},
			threshold:   3.0,
			wantAnomaly: true,
			wantType:    AnomalyDrop,
		},
		{
			name:        "zero MAD (all same values)",
			value:       100,
			values:      []float64{50, 50, 50, 50, 50},
			threshold:   3.0,
			wantAnomaly: false, // MAD is 0, can't detect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewStatisticalDetector(tt.threshold)
			score, isAnomaly, anomalyType := d.DetectWithMAD(tt.value, tt.values)

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

func TestGrubbsTest(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		values     []float64
		wantOutlier bool
	}{
		{
			name:       "insufficient data",
			value:      100,
			values:     []float64{1, 2, 3, 4, 5},
			wantOutlier: false,
		},
		{
			name:       "normal value",
			value:      50,
			values:     []float64{45, 50, 55, 48, 52, 49, 51, 47, 53, 50},
			wantOutlier: false,
		},
		{
			name:       "outlier value",
			value:      200,
			values:     []float64{45, 50, 55, 48, 52, 49, 51, 47, 53, 50},
			wantOutlier: true,
		},
		{
			name:       "zero stddev",
			value:      100,
			values:     []float64{50, 50, 50, 50, 50, 50, 50},
			wantOutlier: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGrubbsTest(0.05)
			isOutlier := g.Test(tt.value, tt.values)
			if isOutlier != tt.wantOutlier {
				t.Errorf("isOutlier = %v, want %v", isOutlier, tt.wantOutlier)
			}
		})
	}
}

func TestIQRDetector(t *testing.T) {
	tests := []struct {
		name        string
		value       float64
		values      []float64
		multiplier  float64
		wantOutlier bool
	}{
		{
			name:        "insufficient data",
			value:       100,
			values:      []float64{1, 2, 3},
			multiplier:  1.5,
			wantOutlier: false,
		},
		{
			name:        "normal value",
			value:       50,
			values:      []float64{40, 45, 50, 55, 60, 45, 50, 55},
			multiplier:  1.5,
			wantOutlier: false,
		},
		{
			name:        "outlier above",
			value:       150,
			values:      []float64{40, 45, 50, 55, 60, 45, 50, 55},
			multiplier:  1.5,
			wantOutlier: true,
		},
		{
			name:        "outlier below",
			value:       -20,
			values:      []float64{40, 45, 50, 55, 60, 45, 50, 55},
			multiplier:  1.5,
			wantOutlier: true,
		},
		{
			name:        "extreme outlier threshold",
			value:       200,
			values:      []float64{40, 45, 50, 55, 60, 45, 50, 55},
			multiplier:  3.0,
			wantOutlier: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewIQRDetector(tt.multiplier)
			isOutlier, lower, upper := d.Detect(tt.value, tt.values)

			if isOutlier != tt.wantOutlier {
				t.Errorf("isOutlier = %v, want %v (bounds: [%v, %v])", isOutlier, tt.wantOutlier, lower, upper)
			}

			// If we have enough data, bounds should be valid
			if len(tt.values) >= 4 {
				if lower > upper {
					t.Errorf("lower bound %v > upper bound %v", lower, upper)
				}
			}
		})
	}
}

func TestNewGrubbsTest(t *testing.T) {
	g := NewGrubbsTest(0.05)
	if g == nil {
		t.Fatal("expected non-nil GrubbsTest")
	}
	if g.significance != 0.05 {
		t.Errorf("significance = %v, want 0.05", g.significance)
	}
}

func TestNewIQRDetector(t *testing.T) {
	d := NewIQRDetector(1.5)
	if d == nil {
		t.Fatal("expected non-nil IQRDetector")
	}
	if d.multiplier != 1.5 {
		t.Errorf("multiplier = %v, want 1.5", d.multiplier)
	}
}
