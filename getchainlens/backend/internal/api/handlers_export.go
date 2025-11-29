package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// ExportHandlers provides HTTP handlers for data export
type ExportHandlers struct {
	// Add service dependencies as needed
}

// NewExportHandlers creates new export handlers
func NewExportHandlers() *ExportHandlers {
	return &ExportHandlers{}
}

// RegisterRoutes registers export routes
func (h *ExportHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/export", func(r chi.Router) {
		r.Get("/{network}/transactions", h.ExportTransactions)
		r.Get("/{network}/token-transfers", h.ExportTokenTransfers)
		r.Get("/{network}/internal-txs", h.ExportInternalTransactions)
		r.Get("/{network}/address/{address}/transactions", h.ExportAddressTransactions)
		r.Get("/{network}/address/{address}/token-transfers", h.ExportAddressTokenTransfers)
	})
}

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportParams holds common export parameters
type ExportParams struct {
	Network   string
	Address   string
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Format    ExportFormat
}

// parseExportParams extracts export parameters from request
func parseExportParams(r *http.Request) *ExportParams {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")

	format := ExportFormat(r.URL.Query().Get("format"))
	if format != FormatJSON {
		format = FormatCSV
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	// Parse dates
	startDate := time.Now().AddDate(0, -1, 0) // Default: 1 month ago
	endDate := time.Now()

	if s := r.URL.Query().Get("start"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = t
		}
	}
	if e := r.URL.Query().Get("end"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	return &ExportParams{
		Network:   network,
		Address:   address,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Format:    format,
	}
}

// ExportTransactions exports transactions
// @Summary Export transactions
// @Tags Export
// @Param network path string true "Network name"
// @Param format query string false "Format (csv or json)" default(csv)
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(1000)
// @Produce text/csv,application/json
// @Router /export/{network}/transactions [get]
func (h *ExportHandlers) ExportTransactions(w http.ResponseWriter, r *http.Request) {
	params := parseExportParams(r)

	// Sample data - in real implementation, fetch from database
	transactions := []map[string]interface{}{
		{
			"hash":         "0xabc123...",
			"block_number": 18000000,
			"from":         "0x1111...",
			"to":           "0x2222...",
			"value":        "1000000000000000000",
			"gas_used":     21000,
			"gas_price":    "30000000000",
			"timestamp":    time.Now().Format(time.RFC3339),
			"status":       "success",
		},
	}

	filename := fmt.Sprintf("%s_transactions_%s_%s",
		params.Network,
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"))

	if params.Format == FormatJSON {
		h.writeJSON(w, filename, transactions)
	} else {
		headers := []string{"hash", "block_number", "from", "to", "value", "gas_used", "gas_price", "timestamp", "status"}
		h.writeCSV(w, filename, headers, transactions)
	}
}

// ExportTokenTransfers exports token transfers
// @Summary Export token transfers
// @Tags Export
// @Param network path string true "Network name"
// @Param token query string false "Token address filter"
// @Param format query string false "Format (csv or json)" default(csv)
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(1000)
// @Produce text/csv,application/json
// @Router /export/{network}/token-transfers [get]
func (h *ExportHandlers) ExportTokenTransfers(w http.ResponseWriter, r *http.Request) {
	params := parseExportParams(r)

	transfers := []map[string]interface{}{
		{
			"tx_hash":       "0xabc123...",
			"block_number":  18000000,
			"token_address": "0xtoken...",
			"token_symbol":  "USDC",
			"from":          "0x1111...",
			"to":            "0x2222...",
			"value":         "1000000",
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	filename := fmt.Sprintf("%s_token_transfers_%s_%s",
		params.Network,
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"))

	if params.Format == FormatJSON {
		h.writeJSON(w, filename, transfers)
	} else {
		headers := []string{"tx_hash", "block_number", "token_address", "token_symbol", "from", "to", "value", "timestamp"}
		h.writeCSV(w, filename, headers, transfers)
	}
}

// ExportInternalTransactions exports internal transactions
// @Summary Export internal transactions
// @Tags Export
// @Param network path string true "Network name"
// @Param format query string false "Format (csv or json)" default(csv)
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(1000)
// @Produce text/csv,application/json
// @Router /export/{network}/internal-txs [get]
func (h *ExportHandlers) ExportInternalTransactions(w http.ResponseWriter, r *http.Request) {
	params := parseExportParams(r)

	internalTxs := []map[string]interface{}{
		{
			"tx_hash":      "0xabc123...",
			"trace_index":  0,
			"block_number": 18000000,
			"trace_type":   "CALL",
			"from":         "0x1111...",
			"to":           "0x2222...",
			"value":        "1000000000000000000",
			"gas_used":     50000,
			"error":        "",
			"timestamp":    time.Now().Format(time.RFC3339),
		},
	}

	filename := fmt.Sprintf("%s_internal_txs_%s_%s",
		params.Network,
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"))

	if params.Format == FormatJSON {
		h.writeJSON(w, filename, internalTxs)
	} else {
		headers := []string{"tx_hash", "trace_index", "block_number", "trace_type", "from", "to", "value", "gas_used", "error", "timestamp"}
		h.writeCSV(w, filename, headers, internalTxs)
	}
}

// ExportAddressTransactions exports transactions for an address
// @Summary Export address transactions
// @Tags Export
// @Param network path string true "Network name"
// @Param address path string true "Address"
// @Param format query string false "Format (csv or json)" default(csv)
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(1000)
// @Produce text/csv,application/json
// @Router /export/{network}/address/{address}/transactions [get]
func (h *ExportHandlers) ExportAddressTransactions(w http.ResponseWriter, r *http.Request) {
	params := parseExportParams(r)

	transactions := []map[string]interface{}{
		{
			"hash":         "0xabc123...",
			"block_number": 18000000,
			"from":         params.Address,
			"to":           "0x2222...",
			"value":        "1000000000000000000",
			"gas_used":     21000,
			"gas_price":    "30000000000",
			"timestamp":    time.Now().Format(time.RFC3339),
			"status":       "success",
			"direction":    "out",
		},
	}

	filename := fmt.Sprintf("%s_%s_transactions_%s_%s",
		params.Network,
		params.Address[:10],
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"))

	if params.Format == FormatJSON {
		h.writeJSON(w, filename, transactions)
	} else {
		headers := []string{"hash", "block_number", "from", "to", "value", "gas_used", "gas_price", "timestamp", "status", "direction"}
		h.writeCSV(w, filename, headers, transactions)
	}
}

// ExportAddressTokenTransfers exports token transfers for an address
// @Summary Export address token transfers
// @Tags Export
// @Param network path string true "Network name"
// @Param address path string true "Address"
// @Param format query string false "Format (csv or json)" default(csv)
// @Param start query string false "Start date (YYYY-MM-DD)"
// @Param end query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(1000)
// @Produce text/csv,application/json
// @Router /export/{network}/address/{address}/token-transfers [get]
func (h *ExportHandlers) ExportAddressTokenTransfers(w http.ResponseWriter, r *http.Request) {
	params := parseExportParams(r)

	transfers := []map[string]interface{}{
		{
			"tx_hash":       "0xabc123...",
			"block_number":  18000000,
			"token_address": "0xtoken...",
			"token_symbol":  "USDC",
			"from":          params.Address,
			"to":            "0x2222...",
			"value":         "1000000",
			"timestamp":     time.Now().Format(time.RFC3339),
			"direction":     "out",
		},
	}

	filename := fmt.Sprintf("%s_%s_token_transfers_%s_%s",
		params.Network,
		params.Address[:10],
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"))

	if params.Format == FormatJSON {
		h.writeJSON(w, filename, transfers)
	} else {
		headers := []string{"tx_hash", "block_number", "token_address", "token_symbol", "from", "to", "value", "timestamp", "direction"}
		h.writeCSV(w, filename, headers, transfers)
	}
}

// writeCSV writes data as CSV
func (h *ExportHandlers) writeCSV(w http.ResponseWriter, filename string, headers []string, data []map[string]interface{}) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write headers
	writer.Write(headers)

	// Write data
	for _, row := range data {
		record := make([]string, len(headers))
		for i, header := range headers {
			if val, ok := row[header]; ok {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		writer.Write(record)
	}
}

// writeJSON writes data as JSON
func (h *ExportHandlers) writeJSON(w http.ResponseWriter, filename string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}
