package product

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if input.Name == "" || input.Price < 0 {
		http.Error(w, "name and valid price are required", http.StatusBadRequest)
		return
	}
	p, err := h.repo.Create(r.Context(), storeID, input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	products, err := h.repo.List(r.Context(), storeID, false)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if products == nil {
		products = []Product{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	id := chi.URLParam(r, "id")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	p, err := h.repo.GetByID(r.Context(), id, storeID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	id := chi.URLParam(r, "id")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	p, err := h.repo.Update(r.Context(), id, storeID, input)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	id := chi.URLParam(r, "id")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	if err := h.repo.Delete(r.Context(), id, storeID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	s, err := h.storeRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	products, err := h.repo.List(r.Context(), s.ID, true)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if products == nil {
		products = []Product{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	id := chi.URLParam(r, "id")
	s, err := h.storeRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	p, err := h.repo.GetByID(r.Context(), id, s.ID)
	if err != nil || !p.Published {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}
