package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"getchainlens.com/chainlens/backend/internal/tokens"
)

// TokenHandlers contains all token-related HTTP handlers
type TokenHandlers struct {
	service *tokens.Service
}

// NewTokenHandlers creates a new TokenHandlers instance
func NewTokenHandlers(service *tokens.Service) *TokenHandlers {
	return &TokenHandlers{service: service}
}

// ============================================================================
// TOKENS
// ============================================================================

// HandleListTokens handles GET /tokens/{network}
func (h *TokenHandlers) HandleListTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		filter := tokens.TokenFilter{
			Network:  network,
			Page:     parseIntParam(r, "page", 1),
			PageSize: parseIntParam(r, "pageSize", 20),
			SortBy:   r.URL.Query().Get("sortBy"),
			SortOrder: r.URL.Query().Get("sortOrder"),
		}

		if q := r.URL.Query().Get("q"); q != "" {
			filter.Query = &q
		}
		if tokenType := r.URL.Query().Get("type"); tokenType != "" {
			filter.TokenType = &tokenType
		}

		result, err := h.service.ListTokens(r.Context(), filter)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetToken handles GET /tokens/{network}/{address}
func (h *TokenHandlers) HandleGetToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		token, err := h.service.GetToken(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if token == nil {
			respondError(w, http.StatusNotFound, "token not found")
			return
		}

		respondJSON(w, http.StatusOK, token)
	}
}

// HandleSearchTokens handles GET /tokens/{network}/search
func (h *TokenHandlers) HandleSearchTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		query := r.URL.Query().Get("q")

		if query == "" {
			respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		result, err := h.service.SearchTokens(r.Context(), network, query)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"tokens": result,
			"count":  len(result),
		})
	}
}

// ============================================================================
// TOKEN TRANSFERS
// ============================================================================

// HandleListTokenTransfers handles GET /tokens/{network}/{address}/transfers
func (h *TokenHandlers) HandleListTokenTransfers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.service.GetTokenTransfers(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetTxTokenTransfers handles GET /tokens/{network}/tx/{hash}
func (h *TokenHandlers) HandleGetTxTokenTransfers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		txHash := chi.URLParam(r, "hash")

		transfers, err := h.service.GetTransfersByTxHash(r.Context(), network, txHash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"transfers": transfers,
			"count":     len(transfers),
		})
	}
}

// ============================================================================
// TOKEN HOLDERS
// ============================================================================

// HandleListTokenHolders handles GET /tokens/{network}/{address}/holders
func (h *TokenHandlers) HandleListTokenHolders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.service.ListHolders(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetTokenBalance handles GET /tokens/{network}/{address}/balance/{holder}
func (h *TokenHandlers) HandleGetTokenBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		tokenAddress := chi.URLParam(r, "address")
		holderAddress := chi.URLParam(r, "holder")

		// Try to get from cache first
		balance, err := h.service.GetBalance(r.Context(), network, tokenAddress, holderAddress)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// If refresh=true or balance not found, fetch from chain
		if balance == nil || r.URL.Query().Get("refresh") == "true" {
			balance, err = h.service.SyncBalance(r.Context(), network, tokenAddress, holderAddress)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		if balance == nil {
			respondJSON(w, http.StatusOK, map[string]string{
				"balance": "0",
			})
			return
		}

		// Get token for formatting
		token, _ := h.service.GetToken(r.Context(), network, tokenAddress)
		decimals := 18
		if token != nil {
			decimals = token.Decimals
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"balance":          balance.Balance,
			"formattedBalance": tokens.FormatBalance(balance.Balance, decimals),
			"lastTransferAt":   balance.LastTransferAt,
		})
	}
}

// ============================================================================
// ADDRESS TOKEN BALANCES
// ============================================================================

// HandleGetAddressTokens handles GET /addresses/{network}/{address}/tokens
func (h *TokenHandlers) HandleGetAddressTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.service.GetHolderTokens(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleGetAddressTokenTransfers handles GET /addresses/{network}/{address}/token-transfers
func (h *TokenHandlers) HandleGetAddressTokenTransfers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		result, err := h.service.GetAddressTransfers(r.Context(), network, address, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// ============================================================================
// ROUTES
// ============================================================================

// RegisterTokenRoutes registers token routes on the router
func RegisterTokenRoutes(r chi.Router, service *tokens.Service) {
	h := NewTokenHandlers(service)

	r.Route("/tokens", func(r chi.Router) {
		r.Route("/{network}", func(r chi.Router) {
			r.Get("/", h.HandleListTokens())
			r.Get("/search", h.HandleSearchTokens())
			r.Get("/tx/{hash}", h.HandleGetTxTokenTransfers())

			r.Route("/{address}", func(r chi.Router) {
				r.Get("/", h.HandleGetToken())
				r.Get("/transfers", h.HandleListTokenTransfers())
				r.Get("/holders", h.HandleListTokenHolders())
				r.Get("/balance/{holder}", h.HandleGetTokenBalance())
			})
		})
	})

	// Address token routes (alternative access pattern)
	r.Route("/addresses/{network}/{address}", func(r chi.Router) {
		r.Get("/tokens", h.HandleGetAddressTokens())
		r.Get("/token-transfers", h.HandleGetAddressTokenTransfers())
	})
}

// parseIntParamWithMax parses an integer query parameter with a maximum value
func parseIntParamWithMax(r *http.Request, name string, defaultValue, maxValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	if n > maxValue {
		return maxValue
	}
	if n < 1 {
		return 1
	}
	return n
}
