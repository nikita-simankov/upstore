package store

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nikita-simankov/upstore/internal/auth"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if input.Name == "" || input.Slug == "" {
		http.Error(w, "name and slug are required", http.StatusBadRequest)
		return
	}
	s, err := h.repo.Create(r.Context(), merchantID, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	stores, err := h.repo.ListByMerchant(r.Context(), merchantID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if stores == nil {
		stores = []Store{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stores)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	s, err := h.repo.GetByID(r.Context(), id, merchantID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	s, err := h.repo.Update(r.Context(), id, merchantID, input)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	s, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}
