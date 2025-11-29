package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"getchainlens.com/chainlens/backend/internal/internaltx"
)

// InternalTxHandlers provides HTTP handlers for internal transaction endpoints
type InternalTxHandlers struct {
	service *internaltx.Service
}

// NewInternalTxHandlers creates new internal transaction handlers
func NewInternalTxHandlers(service *internaltx.Service) *InternalTxHandlers {
	return &InternalTxHandlers{service: service}
}

// RegisterRoutes registers internal transaction routes
func (h *InternalTxHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/internal-txs", func(r chi.Router) {
		// Transaction internal calls
		r.Get("/{network}/tx/{hash}", h.GetTransactionInternalTxs)
		r.Get("/{network}/tx/{hash}/tree", h.GetTransactionTraceTree)
		r.Get("/{network}/tx/{hash}/stats", h.GetTransactionCallStats)
		r.Post("/{network}/tx/{hash}/trace", h.TraceTransactionOnDemand)

		// Address internal transactions
		r.Get("/{network}/address/{address}", h.GetAddressInternalTxs)

		// Created contracts
		r.Get("/{network}/contracts-created", h.GetCreatedContracts)

		// Processing status
		r.Get("/{network}/tx/{hash}/status", h.GetProcessingStatus)
		r.Get("/{network}/stats", h.GetProcessingStats)
	})
}

// GetTransactionInternalTxs returns internal transactions for a transaction
// @Summary Get transaction internal transactions
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param hash path string true "Transaction hash"
// @Success 200 {array} internaltx.InternalTransaction
// @Router /internal-txs/{network}/tx/{hash} [get]
func (h *InternalTxHandlers) GetTransactionInternalTxs(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hash := chi.URLParam(r, "hash")

	txs, err := h.service.GetInternalTransactions(r.Context(), network, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if txs == nil {
		txs = []*internaltx.InternalTransaction{}
	}

	respondJSON(w, http.StatusOK, txs)
}

// GetTransactionTraceTree returns internal transactions as a tree
// @Summary Get transaction trace tree
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param hash path string true "Transaction hash"
// @Success 200 {object} internaltx.TraceTree
// @Router /internal-txs/{network}/tx/{hash}/tree [get]
func (h *InternalTxHandlers) GetTransactionTraceTree(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hash := chi.URLParam(r, "hash")

	tree, err := h.service.GetInternalTransactionTree(r.Context(), network, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tree == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"tree": nil})
		return
	}

	respondJSON(w, http.StatusOK, tree)
}

// GetTransactionCallStats returns call statistics for a transaction
// @Summary Get transaction call statistics
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param hash path string true "Transaction hash"
// @Success 200 {object} internaltx.CallStats
// @Router /internal-txs/{network}/tx/{hash}/stats [get]
func (h *InternalTxHandlers) GetTransactionCallStats(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hash := chi.URLParam(r, "hash")

	stats, err := h.service.GetCallStats(r.Context(), network, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// TraceTransactionOnDemand traces a transaction immediately
// @Summary Trace transaction on demand
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param hash path string true "Transaction hash"
// @Success 200 {array} internaltx.InternalTransaction
// @Router /internal-txs/{network}/tx/{hash}/trace [post]
func (h *InternalTxHandlers) TraceTransactionOnDemand(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hash := chi.URLParam(r, "hash")

	txs, err := h.service.TraceTransactionOnDemand(r.Context(), network, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if txs == nil {
		txs = []*internaltx.InternalTransaction{}
	}

	respondJSON(w, http.StatusOK, txs)
}

// GetAddressInternalTxs returns internal transactions for an address
// @Summary Get address internal transactions
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param address path string true "Address"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {array} internaltx.InternalTransaction
// @Router /internal-txs/{network}/address/{address} [get]
func (h *InternalTxHandlers) GetAddressInternalTxs(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	txs, err := h.service.GetAddressInternalTransactions(r.Context(), network, address, page, pageSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if txs == nil {
		txs = []*internaltx.InternalTransaction{}
	}

	respondJSON(w, http.StatusOK, txs)
}

// GetCreatedContracts returns contracts created via internal transactions
// @Summary Get contracts created via internal transactions
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} internaltx.InternalTransaction
// @Router /internal-txs/{network}/contracts-created [get]
func (h *InternalTxHandlers) GetCreatedContracts(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	txs, err := h.service.GetCreatedContracts(r.Context(), network, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if txs == nil {
		txs = []*internaltx.InternalTransaction{}
	}

	respondJSON(w, http.StatusOK, txs)
}

// GetProcessingStatus returns processing status for a transaction
// @Summary Get trace processing status
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Param hash path string true "Transaction hash"
// @Success 200 {object} internaltx.TraceProcessingStatus
// @Router /internal-txs/{network}/tx/{hash}/status [get]
func (h *InternalTxHandlers) GetProcessingStatus(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	hash := chi.URLParam(r, "hash")

	status, err := h.service.GetProcessingStatus(r.Context(), network, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if status == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// GetProcessingStats returns processing statistics for a network
// @Summary Get trace processing statistics
// @Tags Internal Transactions
// @Param network path string true "Network name"
// @Success 200 {object} map[string]int64
// @Router /internal-txs/{network}/stats [get]
func (h *InternalTxHandlers) GetProcessingStats(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")

	stats, err := h.service.GetProcessingStats(r.Context(), network)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
