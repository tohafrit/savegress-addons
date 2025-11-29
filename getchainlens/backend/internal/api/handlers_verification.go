package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"getchainlens.com/chainlens/backend/internal/verification"
)

// VerificationHandlers contains all verification-related HTTP handlers
type VerificationHandlers struct {
	service *verification.Service
	caller  *verification.ContractCaller
}

// NewVerificationHandlers creates a new VerificationHandlers instance
func NewVerificationHandlers(service *verification.Service, caller *verification.ContractCaller) *VerificationHandlers {
	return &VerificationHandlers{
		service: service,
		caller:  caller,
	}
}

// ============================================================================
// VERIFIED CONTRACTS
// ============================================================================

// HandleGetVerifiedContract handles GET /contracts/{network}/{address}
func (h *VerificationHandlers) HandleGetVerifiedContract() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		if !verification.IsNetworkSupported(network) {
			respondError(w, http.StatusBadRequest, "unsupported network")
			return
		}

		contract, err := h.service.GetVerifiedContract(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if contract == nil {
			respondError(w, http.StatusNotFound, "contract not verified")
			return
		}

		respondJSON(w, http.StatusOK, contract)
	}
}

// HandleListVerifiedContracts handles GET /contracts/{network}
func (h *VerificationHandlers) HandleListVerifiedContracts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")

		if !verification.IsNetworkSupported(network) {
			respondError(w, http.StatusBadRequest, "unsupported network")
			return
		}

		page := parseIntParam(r, "page", 1)
		pageSize := parseIntParam(r, "pageSize", 20)

		contracts, total, err := h.service.ListVerifiedContracts(r.Context(), network, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contracts":  contracts,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": (total + int64(pageSize) - 1) / int64(pageSize),
		})
	}
}

// HandleSearchContracts handles GET /contracts/{network}/search
func (h *VerificationHandlers) HandleSearchContracts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		query := r.URL.Query().Get("q")

		if !verification.IsNetworkSupported(network) {
			respondError(w, http.StatusBadRequest, "unsupported network")
			return
		}

		if query == "" {
			respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		contracts, err := h.service.SearchContracts(r.Context(), network, query)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contracts": contracts,
			"count":     len(contracts),
		})
	}
}

// HandleGetContractABI handles GET /contracts/{network}/{address}/abi
func (h *VerificationHandlers) HandleGetContractABI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		abi, err := h.service.GetContractABI(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if abi == nil {
			respondError(w, http.StatusNotFound, "ABI not available")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"abi": abi,
		})
	}
}

// HandleGetContractSourceCode handles GET /contracts/{network}/{address}/source
func (h *VerificationHandlers) HandleGetContractSourceCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		contract, err := h.service.GetVerifiedContract(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if contract == nil {
			respondError(w, http.StatusNotFound, "contract not verified")
			return
		}

		// Parse source files if available
		var sourceFiles map[string]string
		if len(contract.SourceFiles) > 0 {
			json.Unmarshal(contract.SourceFiles, &sourceFiles)
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contractName":    contract.ContractName,
			"compilerVersion": contract.CompilerVersion,
			"sourceCode":      contract.SourceCode,
			"sourceFiles":     sourceFiles,
		})
	}
}

// ============================================================================
// CONTRACT INTERACTION
// ============================================================================

// HandleReadContract handles POST /contracts/{network}/{address}/read
func (h *VerificationHandlers) HandleReadContract() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		var req struct {
			Function string        `json:"function"`
			Args     []interface{} `json:"args"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Function == "" {
			respondError(w, http.StatusBadRequest, "function name is required")
			return
		}

		readReq := &verification.ReadContractRequest{
			Network:  network,
			Address:  address,
			Function: req.Function,
			Args:     req.Args,
		}

		result, err := h.caller.ReadContract(r.Context(), readReq)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleEncodeWrite handles POST /contracts/{network}/{address}/encode
func (h *VerificationHandlers) HandleEncodeWrite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		var req struct {
			Function string        `json:"function"`
			Args     []interface{} `json:"args"`
			Value    string        `json:"value,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Function == "" {
			respondError(w, http.StatusBadRequest, "function name is required")
			return
		}

		writeReq := &verification.WriteContractRequest{
			Network:  network,
			Address:  address,
			Function: req.Function,
			Args:     req.Args,
			Value:    req.Value,
		}

		result, err := h.caller.EncodeWriteTransaction(r.Context(), writeReq)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// ============================================================================
// VERIFICATION REQUESTS
// ============================================================================

// HandleSubmitVerification handles POST /contracts/verify
func (h *VerificationHandlers) HandleSubmitVerification() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verification.VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Network == "" || req.Address == "" {
			respondError(w, http.StatusBadRequest, "network and address are required")
			return
		}

		result, err := h.service.SubmitVerificationRequest(r.Context(), &req)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		respondJSON(w, http.StatusAccepted, result)
	}
}

// HandleGetVerificationStatus handles GET /contracts/verify/{id}
func (h *VerificationHandlers) HandleGetVerificationStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		var id int64
		if _, err := json.Number(idStr).Int64(); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request ID")
			return
		}
		id, _ = json.Number(idStr).Int64()

		request, err := h.service.GetVerificationRequest(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if request == nil {
			respondError(w, http.StatusNotFound, "verification request not found")
			return
		}

		respondJSON(w, http.StatusOK, request)
	}
}

// ============================================================================
// SIGNATURE DECODING
// ============================================================================

// HandleDecodeFunction handles POST /decode/function
func (h *VerificationHandlers) HandleDecodeFunction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Data string `json:"data"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Data == "" {
			respondError(w, http.StatusBadRequest, "data is required")
			return
		}

		sig, err := h.service.DecodeFunction(r.Context(), req.Data)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		if sig == nil {
			respondError(w, http.StatusNotFound, "unknown function selector")
			return
		}

		respondJSON(w, http.StatusOK, sig)
	}
}

// HandleDecodeEvent handles POST /decode/event
func (h *VerificationHandlers) HandleDecodeEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Topic0 string `json:"topic0"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Topic0 == "" {
			respondError(w, http.StatusBadRequest, "topic0 is required")
			return
		}

		sig, err := h.service.DecodeEvent(r.Context(), req.Topic0)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if sig == nil {
			respondError(w, http.StatusNotFound, "unknown event topic")
			return
		}

		respondJSON(w, http.StatusOK, sig)
	}
}

// ============================================================================
// INTERFACES
// ============================================================================

// HandleListInterfaces handles GET /interfaces
func (h *VerificationHandlers) HandleListInterfaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interfaces, err := h.service.ListContractInterfaces(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"interfaces": interfaces,
			"count":      len(interfaces),
		})
	}
}

// HandleGetInterface handles GET /interfaces/{name}
func (h *VerificationHandlers) HandleGetInterface() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		iface, err := h.service.GetContractInterface(r.Context(), name)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if iface == nil {
			respondError(w, http.StatusNotFound, "interface not found")
			return
		}

		respondJSON(w, http.StatusOK, iface)
	}
}

// HandleDetectInterfaces handles POST /contracts/{network}/{address}/detect
func (h *VerificationHandlers) HandleDetectInterfaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		network := chi.URLParam(r, "network")
		address := chi.URLParam(r, "address")

		analysis, err := h.service.AnalyzeContract(r.Context(), network, address)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, analysis)
	}
}

// ============================================================================
// ROUTES
// ============================================================================

// RegisterVerificationRoutes registers verification routes on the router
func RegisterVerificationRoutes(r chi.Router, service *verification.Service, caller *verification.ContractCaller) {
	h := NewVerificationHandlers(service, caller)

	r.Route("/contracts", func(r chi.Router) {
		// Verification submission
		r.Post("/verify", h.HandleSubmitVerification())
		r.Get("/verify/{id}", h.HandleGetVerificationStatus())

		// Network-specific contract routes
		r.Route("/{network}", func(r chi.Router) {
			r.Get("/", h.HandleListVerifiedContracts())
			r.Get("/search", h.HandleSearchContracts())

			r.Route("/{address}", func(r chi.Router) {
				r.Get("/", h.HandleGetVerifiedContract())
				r.Get("/abi", h.HandleGetContractABI())
				r.Get("/source", h.HandleGetContractSourceCode())
				r.Post("/read", h.HandleReadContract())
				r.Post("/encode", h.HandleEncodeWrite())
				r.Get("/detect", h.HandleDetectInterfaces())
			})
		})
	})

	// Decode endpoints
	r.Route("/decode", func(r chi.Router) {
		r.Post("/function", h.HandleDecodeFunction())
		r.Post("/event", h.HandleDecodeEvent())
	})

	// Interfaces
	r.Route("/interfaces", func(r chi.Router) {
		r.Get("/", h.HandleListInterfaces())
		r.Get("/{name}", h.HandleGetInterface())
	})
}
