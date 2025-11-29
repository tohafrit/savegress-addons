package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"getchainlens.com/chainlens/backend/internal/nfts"
)

// NFTHandlers provides HTTP handlers for NFT endpoints
type NFTHandlers struct {
	service *nfts.Service
}

// NewNFTHandlers creates new NFT handlers
func NewNFTHandlers(service *nfts.Service) *NFTHandlers {
	return &NFTHandlers{service: service}
}

// RegisterRoutes registers NFT routes
func (h *NFTHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/nfts", func(r chi.Router) {
		// Collections
		r.Get("/{network}/collections", h.ListCollections)
		r.Get("/{network}/collections/search", h.SearchCollections)
		r.Get("/{network}/collections/{address}", h.GetCollection)
		r.Get("/{network}/collections/{address}/items", h.ListItems)
		r.Get("/{network}/collections/{address}/transfers", h.ListCollectionTransfers)
		r.Get("/{network}/collections/{address}/holders", h.GetCollectionHolders)

		// Items
		r.Get("/{network}/items/{address}/{tokenId}", h.GetItem)
		r.Get("/{network}/items/{address}/{tokenId}/transfers", h.ListItemTransfers)
		r.Get("/{network}/items/{address}/{tokenId}/holders", h.GetTokenHolders)
		r.Post("/{network}/items/{address}/{tokenId}/refresh", h.RefreshMetadata)

		// Address NFTs
		r.Get("/{network}/address/{address}", h.GetAddressNFTs)
		r.Get("/{network}/address/{address}/transfers", h.GetAddressTransfers)
	})
}

// ListCollections returns NFT collections for a network
// @Summary List NFT collections
// @Tags NFTs
// @Param network path string true "Network name"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTCollection
// @Router /nfts/{network}/collections [get]
func (h *NFTHandlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	collections, err := h.service.ListCollections(r.Context(), network, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, collections)
}

// SearchCollections searches NFT collections by name
// @Summary Search NFT collections
// @Tags NFTs
// @Param network path string true "Network name"
// @Param q query string true "Search query"
// @Param limit query int false "Limit" default(10)
// @Success 200 {array} nfts.NFTCollection
// @Router /nfts/{network}/collections/search [get]
func (h *NFTHandlers) SearchCollections(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	query := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if query == "" {
		http.Error(w, "query parameter 'q' required", http.StatusBadRequest)
		return
	}

	if limit <= 0 || limit > 50 {
		limit = 10
	}

	collections, err := h.service.SearchCollections(r.Context(), network, query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, collections)
}

// GetCollection returns a single NFT collection
// @Summary Get NFT collection
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Success 200 {object} nfts.NFTCollection
// @Router /nfts/{network}/collections/{address} [get]
func (h *NFTHandlers) GetCollection(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")

	collection, err := h.service.GetCollection(r.Context(), network, address)
	if err != nil {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, collection)
}

// ListItems returns NFT items in a collection
// @Summary List NFT items in collection
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTItem
// @Router /nfts/{network}/collections/{address}/items [get]
func (h *NFTHandlers) ListItems(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	items, err := h.service.ListItems(r.Context(), network, address, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// ListCollectionTransfers returns transfers for a collection
// @Summary List NFT collection transfers
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTTransfer
// @Router /nfts/{network}/collections/{address}/transfers [get]
func (h *NFTHandlers) ListCollectionTransfers(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	transfers, err := h.service.ListTransfers(r.Context(), network, address, nil, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, transfers)
}

// GetCollectionHolders returns holders of a collection
// @Summary Get NFT collection holders
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTBalance
// @Router /nfts/{network}/collections/{address}/holders [get]
func (h *NFTHandlers) GetCollectionHolders(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	holders, err := h.service.GetCollectionHolders(r.Context(), network, address, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, holders)
}

// GetItem returns a single NFT item
// @Summary Get NFT item
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param tokenId path string true "Token ID"
// @Success 200 {object} nfts.NFTItem
// @Router /nfts/{network}/items/{address}/{tokenId} [get]
func (h *NFTHandlers) GetItem(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	tokenID := chi.URLParam(r, "tokenId")

	item, err := h.service.GetItem(r.Context(), network, address, tokenID)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// ListItemTransfers returns transfers for a specific NFT item
// @Summary List NFT item transfers
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param tokenId path string true "Token ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTTransfer
// @Router /nfts/{network}/items/{address}/{tokenId}/transfers [get]
func (h *NFTHandlers) ListItemTransfers(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	tokenID := chi.URLParam(r, "tokenId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	transfers, err := h.service.ListTransfers(r.Context(), network, address, &tokenID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, transfers)
}

// GetTokenHolders returns holders of a specific ERC-1155 token
// @Summary Get NFT token holders
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param tokenId path string true "Token ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTBalance
// @Router /nfts/{network}/items/{address}/{tokenId}/holders [get]
func (h *NFTHandlers) GetTokenHolders(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	tokenID := chi.URLParam(r, "tokenId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	holders, err := h.service.GetTokenHolders(r.Context(), network, address, tokenID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, holders)
}

// RefreshMetadata refreshes metadata for an NFT item
// @Summary Refresh NFT metadata
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Contract address"
// @Param tokenId path string true "Token ID"
// @Success 200 {object} map[string]string
// @Router /nfts/{network}/items/{address}/{tokenId}/refresh [post]
func (h *NFTHandlers) RefreshMetadata(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	tokenID := chi.URLParam(r, "tokenId")

	if err := h.service.RefreshMetadata(r.Context(), network, address, tokenID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "queued"})
}

// GetAddressNFTs returns all NFTs owned by an address
// @Summary Get address NFTs
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Wallet address"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTItem
// @Router /nfts/{network}/address/{address} [get]
func (h *NFTHandlers) GetAddressNFTs(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	items, err := h.service.GetAddressNFTs(r.Context(), network, address, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// GetAddressTransfers returns NFT transfers for an address
// @Summary Get address NFT transfers
// @Tags NFTs
// @Param network path string true "Network name"
// @Param address path string true "Wallet address"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} nfts.NFTTransfer
// @Router /nfts/{network}/address/{address}/transfers [get]
func (h *NFTHandlers) GetAddressTransfers(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	address := chi.URLParam(r, "address")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	transfers, err := h.service.GetAddressTransfers(r.Context(), network, address, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, transfers)
}
