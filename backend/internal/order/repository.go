package order

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

func (r *Repository) Create(ctx context.Context, input CreateOrderInput) (*Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var o Order
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (store_id, customer_email, customer_name, customer_phone, shipping_address, total, payment_provider)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, store_id, customer_email, customer_name, customer_phone, shipping_address, total, status, payment_provider, payment_id, created_at`,
		input.StoreID, input.CustomerEmail, input.CustomerName, input.CustomerPhone,
		input.ShippingAddress, input.Total, input.PaymentProvider,
	).Scan(&o.ID, &o.StoreID, &o.CustomerEmail, &o.CustomerName, &o.CustomerPhone,
		&o.ShippingAddress, &o.Total, &o.Status, &o.PaymentProvider, &o.PaymentID, &o.CreatedAt)
	if err != nil {
		return nil, err
	}

	for _, item := range input.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, name, price, quantity) VALUES ($1, $2, $3, $4, $5)`,
			o.ID, item.ProductID, item.Name, item.Price, item.Quantity,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o.Items = input.Items
	return &o, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Order, error) {
	var o Order
	err := r.db.QueryRow(ctx,
		`SELECT id, store_id, customer_email, customer_name, customer_phone, shipping_address, total, status, payment_provider, payment_id, created_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.StoreID, &o.CustomerEmail, &o.CustomerName, &o.CustomerPhone,
		&o.ShippingAddress, &o.Total, &o.Status, &o.PaymentProvider, &o.PaymentID, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}
	return &o, nil
}

func (r *Repository) ListByStore(ctx context.Context, storeID string) ([]Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, store_id, customer_email, customer_name, customer_phone, shipping_address, total, status, payment_provider, payment_id, created_at
		 FROM orders WHERE store_id = $1 ORDER BY created_at DESC`,
		storeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.StoreID, &o.CustomerEmail, &o.CustomerName, &o.CustomerPhone,
			&o.ShippingAddress, &o.Total, &o.Status, &o.PaymentProvider, &o.PaymentID, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *Repository) UpdatePaymentID(ctx context.Context, orderID, paymentID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET payment_id = $2, updated_at = NOW() WHERE id = $1`,
		orderID, paymentID,
	)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, paymentID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $2, updated_at = NOW() WHERE payment_id = $1`,
		paymentID, status,
	)
	return err
}

func (r *Repository) UpdateStatusByID(ctx context.Context, id, status string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}
