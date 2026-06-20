package product

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

func (r *Repository) Create(ctx context.Context, storeID string, input CreateInput) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx,
		`INSERT INTO products (store_id, name, description, price, stock, image_url, published)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, store_id, name, description, price, stock, image_url, published, created_at`,
		storeID, input.Name, input.Description, input.Price, input.Stock, input.ImageURL, input.Published,
	).Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.Published, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) List(ctx context.Context, storeID string, publishedOnly bool) ([]Product, error) {
	query := `SELECT id, store_id, name, description, price, stock, image_url, published, created_at
	          FROM products WHERE store_id = $1`
	if publishedOnly {
		query += " AND published = TRUE"
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.Query(ctx, query, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.Published, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *Repository) GetByID(ctx context.Context, id, storeID string) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx,
		`SELECT id, store_id, name, description, price, stock, image_url, published, created_at
		 FROM products WHERE id = $1 AND store_id = $2`,
		id, storeID,
	).Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.Published, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}
	return &p, nil
}

func (r *Repository) Update(ctx context.Context, id, storeID string, input UpdateInput) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx,
		`UPDATE products SET
		   name        = COALESCE($3, name),
		   description = COALESCE($4, description),
		   price       = COALESCE($5, price),
		   stock       = COALESCE($6, stock),
		   image_url   = COALESCE($7, image_url),
		   published   = COALESCE($8, published),
		   updated_at  = NOW()
		 WHERE id = $1 AND store_id = $2
		 RETURNING id, store_id, name, description, price, stock, image_url, published, created_at`,
		id, storeID, input.Name, input.Description, input.Price, input.Stock, input.ImageURL, input.Published,
	).Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.Published, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}
	return &p, nil
}

func (r *Repository) Delete(ctx context.Context, id, storeID string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1 AND store_id = $2`, id, storeID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}
