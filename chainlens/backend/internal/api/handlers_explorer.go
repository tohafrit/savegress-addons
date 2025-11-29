package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"getchainlens.com/chainlens/backend/internal/explorer"
)

// ExplorerHandlers contains all explorer-related HTTP handlers
type ExplorerHandlers struct {
	explorer *explorer.Explorer
}

// NewExplorerHandlers creates a new ExplorerHandlers instance
func NewExplorerHandlers(exp *explorer.Explorer) *ExplorerHandlers {
	return &ExplorerHandlers{explorer: exp}
}

// ============================================================================
// BLOCKS
// ============================================================================

// HandleListBlocks handles GET /explorer/{network}/blocks
func (h *ExplorerHandlers) HandleListBlocks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)
		miner := parseOptionalParam(r, "miner")

		result, err := h.explorer.ListBlocks(r.Context(), network, page, pageSize, miner)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetBlock handles GET /explorer/{network}/blocks/{identifier}
func (h *ExplorerHandlers) HandleGetBlock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		identifier := chi.URLParam(r, "identifier")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		block, err := h.explorer.GetBlock(r.Context(), network, identifier)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if block == nil {
			respondError(w, http.StatusNotFound, "block not found")
			return
		}

		respondJSON(w, http.StatusOK, block)
	}
}

// HandleGetBlockTransactions handles GET /explorer/{network}/blocks/{number}/txs
func (h *ExplorerHandlers) HandleGetBlockTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		numberStr := chi.URLParam(r, "number")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		number, err := strconv.ParseInt(numberStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid block number")
			return
		}

		txs, err := h.explorer.GetBlockTransactions(r.Context(), network, number)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"transactions": txs,
			"count":        len(txs),
		})
	}
}

// HandleGetLatestBlock handles GET /explorer/{network}/blocks/latest
func (h *ExplorerHandlers) HandleGetLatestBlock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		block, err := h.explorer.GetLatestBlock(r.Context(), network)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if block == nil {
			respondError(w, http.StatusNotFound, "no blocks indexed")
			return
		}

		respondJSON(w, http.StatusOK, block)
	}
}

// ============================================================================
// TRANSACTIONS
// ============================================================================

// HandleListTransactions handles GET /explorer/{network}/transactions
func (h *ExplorerHandlers) HandleListTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		filter := explorer.TransactionFilter{
			Network:           network,
			PaginationOptions: explorer.NewPaginationOptions(parseIntParam(r, "page", 1), parseIntParam(r, "pageSize", 20)),
		}

		if blockNum := r.URL.Query().Get("block"); blockNum != "" {
			num, err := strconv.ParseInt(blockNum, 10, 64)
			if err == nil {
				filter.BlockNumber = &num
			}
		}

		if from := r.URL.Query().Get("from"); from != "" {
			filter.FromAddress = &from
		}

		if to := r.URL.Query().Get("to"); to != "" {
			filter.ToAddress = &to
		}

		if status := r.URL.Query().Get("status"); status != "" {
			s, err := strconv.Atoi(status)
			if err == nil {
				filter.Status = &s
			}
		}

		result, err := h.explorer.ListTransactions(r.Context(), filter)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetTransaction handles GET /explorer/{network}/transactions/{hash}
func (h *ExplorerHandlers) HandleGetTransaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		hash := chi.URLParam(r, "hash")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		tx, err := h.explorer.GetTransaction(r.Context(), network, hash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if tx == nil {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}

		respondJSON(w, http.StatusOK, tx)
	}
}

// HandleGetTransactionLogs handles GET /explorer/{network}/transactions/{hash}/logs
func (h *ExplorerHandlers) HandleGetTransactionLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		hash := chi.URLParam(r, "hash")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		logs, err := h.explorer.GetTransactionLogs(r.Context(), network, hash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"logs":  logs,
			"count": len(logs),
		})
	}
}

// ============================================================================
// ADDRESSES
// ============================================================================

// HandleGetAddress handles GET /explorer/{network}/addresses/{address}
func (h *ExplorerHandlers) HandleGetAddress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		addr, err := h.explorer.GetAddress(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, addr)
	}
}

// HandleGetAddressTransactions handles GET /explorer/{network}/addresses/{address}/txs
func (h *ExplorerHandlers) HandleGetAddressTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.explorer.GetAddressTransactions(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetAddressLogs handles GET /explorer/{network}/addresses/{address}/logs
func (h *ExplorerHandlers) HandleGetAddressLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.explorer.GetAddressLogs(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// ============================================================================
// SEARCH & STATS
// ============================================================================

// HandleSearch handles GET /explorer/{network}/search
func (h *ExplorerHandlers) HandleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		query := r.URL.Query().Get("q")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		if query == "" {
			respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		results, err := h.explorer.Search(r.Context(), network, query)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, results)
	}
}

// HandleGetNetworkStats handles GET /explorer/{network}/stats
func (h *ExplorerHandlers) HandleGetNetworkStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		stats, err := h.explorer.GetNetworkStats(r.Context(), network)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, stats)
	}
}

// HandleGetSyncState handles GET /explorer/{network}/sync
func (h *ExplorerHandlers) HandleGetSyncState() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		if !explorer.IsValidNetwork(network) {
			respondError(w, http.StatusBadRequest, "invalid network")
			return
		}

		state, err := h.explorer.GetSyncState(r.Context(), network)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if state == nil {
			respondError(w, http.StatusNotFound, "sync state not found")
			return
		}

		respondJSON(w, http.StatusOK, state)
	}
}

// HandleGetNetworks handles GET /explorer/networks
func (h *ExplorerHandlers) HandleGetNetworks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		networks := explorer.SupportedNetworks()

		type NetworkInfo struct {
			Name    string `json:"name"`
			ChainID int64  `json:"chainId"`
		}

		chainIDs := map[string]int64{
			"ethereum":  1,
			"polygon":   137,
			"arbitrum":  42161,
			"optimism":  10,
			"base":      8453,
			"bsc":       56,
			"avalanche": 43114,
		}

		result := make([]NetworkInfo, len(networks))
		for i, name := range networks {
			result[i] = NetworkInfo{
				Name:    name,
				ChainID: chainIDs[name],
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"networks": result,
		})
	}
}

// ============================================================================
// HELPERS
// ============================================================================

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return n
}

func parseOptionalParam(r *http.Request, name string) *string {
	val := r.URL.Query().Get(name)
	if val == "" {
		return nil
	}
	return &val
}

// RegisterExplorerRoutes registers explorer routes on the router
func RegisterExplorerRoutes(r chi.Router, exp *explorer.Explorer) {
	h := NewExplorerHandlers(exp)

	r.Route("/explorer", func(r chi.Router) {
		// Networks list
		r.Get("/networks", h.HandleGetNetworks())

		// Network-specific routes
		r.Route("/{network}", func(r chi.Router) {
			// Blocks
			r.Get("/blocks", h.HandleListBlocks())
			r.Get("/blocks/latest", h.HandleGetLatestBlock())
			r.Get("/blocks/{identifier}", h.HandleGetBlock())
			r.Get("/blocks/{number}/txs", h.HandleGetBlockTransactions())

			// Transactions
			r.Get("/transactions", h.HandleListTransactions())
			r.Get("/transactions/{hash}", h.HandleGetTransaction())
			r.Get("/transactions/{hash}/logs", h.HandleGetTransactionLogs())

			// Addresses
			r.Get("/addresses/{address}", h.HandleGetAddress())
			r.Get("/addresses/{address}/txs", h.HandleGetAddressTransactions())
			r.Get("/addresses/{address}/logs", h.HandleGetAddressLogs())

			// Search & Stats
			r.Get("/search", h.HandleSearch())
			r.Get("/stats", h.HandleGetNetworkStats())
			r.Get("/sync", h.HandleGetSyncState())
		})
	})
}
