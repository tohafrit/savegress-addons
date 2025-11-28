package transactions

import (
	"sync"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// Aggregator aggregates transaction statistics
type Aggregator struct {
	mu sync.RWMutex

	totalCount  int
	totalVolume decimal.Decimal

	byType     map[string]*TypeStats
	byCategory map[string]*CategoryStats
	byStatus   map[string]int
	byHour     map[int]int
	dailyVolume map[string]decimal.Decimal

	// Rolling windows
	hourlyVolume  []decimal.Decimal // Last 24 hours
	hourlyCount   []int
	currentHour   int
}

// NewAggregator creates a new aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		byType:       make(map[string]*TypeStats),
		byCategory:   make(map[string]*CategoryStats),
		byStatus:     make(map[string]int),
		byHour:       make(map[int]int),
		dailyVolume:  make(map[string]decimal.Decimal),
		hourlyVolume: make([]decimal.Decimal, 24),
		hourlyCount:  make([]int, 24),
	}
}

// Add adds a transaction to the aggregates
func (a *Aggregator) Add(txn *models.Transaction) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update totals
	a.totalCount++
	a.totalVolume = a.totalVolume.Add(txn.Amount)

	// By type
	typeName := string(txn.Type)
	if a.byType[typeName] == nil {
		a.byType[typeName] = &TypeStats{}
	}
	a.byType[typeName].Count++
	a.byType[typeName].Volume = a.byType[typeName].Volume.Add(txn.Amount)
	if a.byType[typeName].Count > 0 {
		a.byType[typeName].Average = a.byType[typeName].Volume.Div(decimal.NewFromInt(int64(a.byType[typeName].Count)))
	}

	// By category
	if txn.Category != "" {
		if a.byCategory[txn.Category] == nil {
			a.byCategory[txn.Category] = &CategoryStats{}
		}
		a.byCategory[txn.Category].Count++
		a.byCategory[txn.Category].Volume = a.byCategory[txn.Category].Volume.Add(txn.Amount)
	}

	// By status
	a.byStatus[string(txn.Status)]++

	// By hour
	hour := txn.CreatedAt.Hour()
	a.byHour[hour]++

	// Daily volume
	day := txn.CreatedAt.Format("2006-01-02")
	if _, ok := a.dailyVolume[day]; !ok {
		a.dailyVolume[day] = decimal.Zero
	}
	a.dailyVolume[day] = a.dailyVolume[day].Add(txn.Amount)

	// Hourly rolling window
	currentHour := time.Now().Hour()
	if currentHour != a.currentHour {
		// Reset if we've moved to a new hour
		a.currentHour = currentHour
		a.hourlyVolume[currentHour] = decimal.Zero
		a.hourlyCount[currentHour] = 0
	}
	a.hourlyVolume[currentHour] = a.hourlyVolume[currentHour].Add(txn.Amount)
	a.hourlyCount[currentHour]++
}

// Remove removes a transaction from the aggregates (for reversals)
func (a *Aggregator) Remove(txn *models.Transaction) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update totals
	a.totalCount--
	a.totalVolume = a.totalVolume.Sub(txn.Amount)

	// By type
	typeName := string(txn.Type)
	if stats, ok := a.byType[typeName]; ok {
		stats.Count--
		stats.Volume = stats.Volume.Sub(txn.Amount)
		if stats.Count > 0 {
			stats.Average = stats.Volume.Div(decimal.NewFromInt(int64(stats.Count)))
		} else {
			stats.Average = decimal.Zero
		}
	}

	// By category
	if txn.Category != "" {
		if stats, ok := a.byCategory[txn.Category]; ok {
			stats.Count--
			stats.Volume = stats.Volume.Sub(txn.Amount)
		}
	}

	// By status
	if count, ok := a.byStatus[string(txn.Status)]; ok && count > 0 {
		a.byStatus[string(txn.Status)]--
	}

	// By hour
	hour := txn.CreatedAt.Hour()
	if count, ok := a.byHour[hour]; ok && count > 0 {
		a.byHour[hour]--
	}

	// Daily volume
	day := txn.CreatedAt.Format("2006-01-02")
	if vol, ok := a.dailyVolume[day]; ok {
		a.dailyVolume[day] = vol.Sub(txn.Amount)
	}
}

// GetStats returns current statistics
func (a *Aggregator) GetStats() *TransactionStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Copy stats to avoid concurrent access issues
	byType := make(map[string]*TypeStats)
	for k, v := range a.byType {
		byType[k] = &TypeStats{
			Count:   v.Count,
			Volume:  v.Volume,
			Average: v.Average,
		}
	}

	byCategory := make(map[string]*CategoryStats)
	for k, v := range a.byCategory {
		byCategory[k] = &CategoryStats{
			Count:  v.Count,
			Volume: v.Volume,
		}
	}

	byStatus := make(map[string]int)
	for k, v := range a.byStatus {
		byStatus[k] = v
	}

	dailyVolume := make(map[string]decimal.Decimal)
	for k, v := range a.dailyVolume {
		dailyVolume[k] = v
	}

	// Calculate average
	var averageAmount decimal.Decimal
	if a.totalCount > 0 {
		averageAmount = a.totalVolume.Div(decimal.NewFromInt(int64(a.totalCount)))
	}

	// Find peak hour
	peakHour := 0
	peakCount := 0
	for hour, count := range a.byHour {
		if count > peakCount {
			peakHour = hour
			peakCount = count
		}
	}

	return &TransactionStats{
		TotalCount:    a.totalCount,
		TotalVolume:   a.totalVolume,
		ByType:        byType,
		ByCategory:    byCategory,
		ByStatus:      byStatus,
		DailyVolume:   dailyVolume,
		AverageAmount: averageAmount,
		PeakHour:      peakHour,
	}
}

// GetHourlyVolume returns hourly volume for the last 24 hours
func (a *Aggregator) GetHourlyVolume() []HourlyStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := make([]HourlyStats, 24)
	now := time.Now()

	for i := 0; i < 24; i++ {
		hour := (now.Hour() - 23 + i + 24) % 24
		stats[i] = HourlyStats{
			Hour:   hour,
			Volume: a.hourlyVolume[hour],
			Count:  a.hourlyCount[hour],
		}
	}

	return stats
}

// HourlyStats contains hourly statistics
type HourlyStats struct {
	Hour   int             `json:"hour"`
	Volume decimal.Decimal `json:"volume"`
	Count  int             `json:"count"`
}

// GetDailyVolume returns daily volumes for a date range
func (a *Aggregator) GetDailyVolume(startDate, endDate time.Time) []DailyStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var stats []DailyStats
	current := startDate

	for !current.After(endDate) {
		day := current.Format("2006-01-02")
		volume := decimal.Zero
		if v, ok := a.dailyVolume[day]; ok {
			volume = v
		}
		stats = append(stats, DailyStats{
			Date:   current,
			Volume: volume,
		})
		current = current.AddDate(0, 0, 1)
	}

	return stats
}

// DailyStats contains daily statistics
type DailyStats struct {
	Date   time.Time       `json:"date"`
	Volume decimal.Decimal `json:"volume"`
}

// Reset resets all aggregates
func (a *Aggregator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalCount = 0
	a.totalVolume = decimal.Zero
	a.byType = make(map[string]*TypeStats)
	a.byCategory = make(map[string]*CategoryStats)
	a.byStatus = make(map[string]int)
	a.byHour = make(map[int]int)
	a.dailyVolume = make(map[string]decimal.Decimal)
	a.hourlyVolume = make([]decimal.Decimal, 24)
	a.hourlyCount = make([]int, 24)
}
