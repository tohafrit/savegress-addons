package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"getchainlens.com/chainlens/backend/internal/analytics"
)

// AnalyticsHandlers provides HTTP handlers for analytics endpoints
type AnalyticsHandlers struct {
	service *analytics.Service
}

// NewAnalyticsHandlers creates new analytics handlers
func NewAnalyticsHandlers(service *analytics.Service) *AnalyticsHandlers {
	return &AnalyticsHandlers{service: service}
}

// RegisterRoutes registers analytics routes
func (h *AnalyticsHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/analytics", func(r chi.Router) {
		// Network overview
		r.Get("/{network}/overview", h.GetNetworkOverview)

		// Charts
		r.Get("/{network}/charts/transactions", h.GetTransactionChart)
		r.Get("/{network}/charts/gas", h.GetGasChart)
		r.Get("/{network}/charts/addresses", h.GetActiveAddressesChart)

		// Stats
		r.Get("/{network}/stats/daily", h.GetDailyStats)
		r.Get("/{network}/stats/hourly", h.GetHourlyStats)

		// Rankings
		r.Get("/{network}/top-tokens", h.GetTopTokens)
		r.Get("/{network}/top-contracts", h.GetTopContracts)
	})

	r.Route("/gas", func(r chi.Router) {
		r.Get("/{network}", h.GetCurrentGasPrice)
		r.Get("/{network}/history", h.GetGasPriceHistory)
	})
}

// GetNetworkOverview returns network overview statistics
// @Summary Get network overview
// @Tags Analytics
// @Param network path string true "Network name"
// @Success 200 {object} analytics.NetworkOverview
// @Router /analytics/{network}/overview [get]
func (h *AnalyticsHandlers) GetNetworkOverview(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	overview, err := h.service.GetNetworkOverview(r.Context(), network)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, overview)
}

// GetTransactionChart returns transaction count chart data
// @Summary Get transaction chart
// @Tags Analytics
// @Param network path string true "Network name"
// @Param days query int false "Number of days" default(30)
// @Success 200 {object} analytics.ChartData
// @Router /analytics/{network}/charts/transactions [get]
func (h *AnalyticsHandlers) GetTransactionChart(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if days <= 0 || days > 365 {
		days = 30
	}

	chart, err := h.service.GetTransactionChart(r.Context(), network, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, chart)
}

// GetGasChart returns gas price chart data
// @Summary Get gas price chart
// @Tags Analytics
// @Param network path string true "Network name"
// @Param hours query int false "Number of hours" default(24)
// @Success 200 {object} analytics.ChartData
// @Router /analytics/{network}/charts/gas [get]
func (h *AnalyticsHandlers) GetGasChart(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if hours <= 0 || hours > 168 { // max 1 week
		hours = 24
	}

	chart, err := h.service.GetGasChart(r.Context(), network, hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, chart)
}

// GetActiveAddressesChart returns active addresses chart data
// @Summary Get active addresses chart
// @Tags Analytics
// @Param network path string true "Network name"
// @Param days query int false "Number of days" default(30)
// @Success 200 {object} analytics.ChartData
// @Router /analytics/{network}/charts/addresses [get]
func (h *AnalyticsHandlers) GetActiveAddressesChart(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if days <= 0 || days > 365 {
		days = 30
	}

	chart, err := h.service.GetActiveAddressesChart(r.Context(), network, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, chart)
}

// GetDailyStats returns daily statistics
// @Summary Get daily statistics
// @Tags Analytics
// @Param network path string true "Network name"
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Success 200 {array} analytics.DailyStats
// @Router /analytics/{network}/stats/daily [get]
func (h *AnalyticsHandlers) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	// Default to last 30 days
	endDate := time.Now().UTC().Truncate(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -30)

	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = t
		}
	}

	stats, err := h.service.GetDailyStats(r.Context(), network, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetHourlyStats returns hourly statistics
// @Summary Get hourly statistics
// @Tags Analytics
// @Param network path string true "Network name"
// @Param hours query int false "Number of hours" default(24)
// @Success 200 {array} analytics.HourlyStats
// @Router /analytics/{network}/stats/hourly [get]
func (h *AnalyticsHandlers) GetHourlyStats(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if hours <= 0 || hours > 168 { // max 1 week
		hours = 24
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	stats, err := h.service.GetHourlyStats(r.Context(), network, startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetTopTokens returns top tokens by activity
// @Summary Get top tokens
// @Tags Analytics
// @Param network path string true "Network name"
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} analytics.TopToken
// @Router /analytics/{network}/top-tokens [get]
func (h *AnalyticsHandlers) GetTopTokens(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	tokens, err := h.service.GetTopTokens(r.Context(), network, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

// GetTopContracts returns top contracts by activity
// @Summary Get top contracts
// @Tags Analytics
// @Param network path string true "Network name"
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} analytics.TopContract
// @Router /analytics/{network}/top-contracts [get]
func (h *AnalyticsHandlers) GetTopContracts(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	contracts, err := h.service.GetTopContracts(r.Context(), network, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, contracts)
}

// GetCurrentGasPrice returns current gas price estimates
// @Summary Get current gas price
// @Tags Gas
// @Param network path string true "Network name"
// @Success 200 {object} analytics.GasPriceEstimate
// @Router /gas/{network} [get]
func (h *AnalyticsHandlers) GetCurrentGasPrice(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	gasPrice, err := h.service.GetCurrentGasPrice(r.Context(), network)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, gasPrice)
}

// GetGasPriceHistory returns gas price history
// @Summary Get gas price history
// @Tags Gas
// @Param network path string true "Network name"
// @Param hours query int false "Number of hours" default(24)
// @Success 200 {array} analytics.GasPrice
// @Router /gas/{network}/history [get]
func (h *AnalyticsHandlers) GetGasPriceHistory(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))

	if !isValidNetwork(network) {
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}

	if hours <= 0 || hours > 168 { // max 1 week
		hours = 24
	}

	history, err := h.service.GetGasPriceHistory(r.Context(), network, hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// isValidNetwork checks if network is supported
func isValidNetwork(network string) bool {
	for _, n := range analytics.SupportedNetworks {
		if n == network {
			return true
		}
	}
	return false
}
