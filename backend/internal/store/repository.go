package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, merchantID string, input CreateInput) (*Store, error) {
	var s Store
	err := r.db.QueryRow(ctx,
		`INSERT INTO stores (merchant_id, name, slug, logo_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, merchant_id, name, slug, logo_url, plan, trial_ends_at, created_at`,
		merchantID, input.Name, input.Slug, input.LogoURL,
	).Scan(&s.ID, &s.MerchantID, &s.Name, &s.Slug, &s.LogoURL, &s.Plan, &s.TrialEndsAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("slug already taken")
	}
	return &s, nil
}

func (r *Repository) ListByMerchant(ctx context.Context, merchantID string) ([]Store, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, merchant_id, name, slug, logo_url, plan, trial_ends_at, created_at
		 FROM stores WHERE merchant_id = $1 ORDER BY created_at DESC`,
		merchantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stores []Store
	for rows.Next() {
		var s Store
		if err := rows.Scan(&s.ID, &s.MerchantID, &s.Name, &s.Slug, &s.LogoURL, &s.Plan, &s.TrialEndsAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	return stores, nil
}

func (r *Repository) GetByID(ctx context.Context, id, merchantID string) (*Store, error) {
	var s Store
	err := r.db.QueryRow(ctx,
		`SELECT id, merchant_id, name, slug, logo_url, plan, trial_ends_at, created_at
		 FROM stores WHERE id = $1 AND merchant_id = $2`,
		id, merchantID,
	).Scan(&s.ID, &s.MerchantID, &s.Name, &s.Slug, &s.LogoURL, &s.Plan, &s.TrialEndsAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store not found")
	}
	return &s, nil
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Store, error) {
	var s Store
	err := r.db.QueryRow(ctx,
		`SELECT id, merchant_id, name, slug, logo_url, plan, trial_ends_at, created_at
		 FROM stores WHERE slug = $1`,
		slug,
	).Scan(&s.ID, &s.MerchantID, &s.Name, &s.Slug, &s.LogoURL, &s.Plan, &s.TrialEndsAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store not found")
	}
	return &s, nil
}

func (r *Repository) Update(ctx context.Context, id, merchantID string, input UpdateInput) (*Store, error) {
	var s Store
	err := r.db.QueryRow(ctx,
		`UPDATE stores SET
		   name       = COALESCE($3, name),
		   slug       = COALESCE($4, slug),
		   logo_url   = COALESCE($5, logo_url),
		   updated_at = NOW()
		 WHERE id = $1 AND merchant_id = $2
		 RETURNING id, merchant_id, name, slug, logo_url, plan, trial_ends_at, created_at`,
		id, merchantID, input.Name, input.Slug, input.LogoURL,
	).Scan(&s.ID, &s.MerchantID, &s.Name, &s.Slug, &s.LogoURL, &s.Plan, &s.TrialEndsAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store not found")
	}
	return &s, nil
}
