package order

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nikita-simankov/upstore/internal/auth"
	storeRepo "github.com/nikita-simankov/upstore/internal/store"
)

type Handler struct {
	repo      *Repository
	storeRepo *storeRepo.Repository
}

func NewHandler(repo *Repository, sr *storeRepo.Repository) *Handler {
	return &Handler{repo: repo, storeRepo: sr}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	orders, err := h.repo.ListByStore(r.Context(), storeID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if orders == nil {
		orders = []Order{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	orderID := chi.URLParam(r, "id")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateStatusByID(r.Context(), orderID, body.Status); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
