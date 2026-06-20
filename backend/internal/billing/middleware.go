package billing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	storeRepo "github.com/nikita-simankov/upstore/internal/store"
)

var planLimits = map[string]int{
	"start":  100,
	"growth": 10000,
	"pro":    -1,
}

type Middleware struct {
	db        *pgxpool.Pool
	storeRepo *storeRepo.Repository
}

func NewMiddleware(db *pgxpool.Pool, sr *storeRepo.Repository) *Middleware {
	return &Middleware{db: db, storeRepo: sr}
}

func (m *Middleware) EnforceProductLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storeID := chi.URLParam(r, "storeId")
		var plan string
		if err := m.db.QueryRow(r.Context(),
			`SELECT plan FROM stores WHERE id = $1`, storeID,
		).Scan(&plan); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		limit, ok := planLimits[plan]
		if !ok || limit == -1 {
			next.ServeHTTP(w, r)
			return
		}
		var count int
		_ = m.db.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM products WHERE store_id = $1`, storeID,
		).Scan(&count)
		if count >= limit {
			http.Error(w, "product limit reached for current plan", http.StatusPaymentRequired)
			return
		}
		next.ServeHTTP(w, r)
	})
}
