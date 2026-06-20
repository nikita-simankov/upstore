package analytics

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/internal/auth"
	storeRepo "github.com/nikita-simankov/upstore/internal/store"
)

type Handler struct {
	db        *pgxpool.Pool
	storeRepo *storeRepo.Repository
}

func NewHandler(db *pgxpool.Pool, sr *storeRepo.Repository) *Handler {
	return &Handler{db: db, storeRepo: sr}
}

type StoreStats struct {
	Revenue    int `json:"revenue"`
	OrderCount int `json:"order_count"`
	PaidCount  int `json:"paid_count"`
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	merchantID := auth.MerchantIDFromContext(r.Context())
	storeID := chi.URLParam(r, "storeId")
	if _, err := h.storeRepo.GetByID(r.Context(), storeID, merchantID); err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}
	var stats StoreStats
	_ = h.db.QueryRow(r.Context(),
		`SELECT
		   COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0),
		   COUNT(*),
		   COUNT(*) FILTER (WHERE status = 'paid')
		 FROM orders WHERE store_id = $1`,
		storeID,
	).Scan(&stats.Revenue, &stats.OrderCount, &stats.PaidCount)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
