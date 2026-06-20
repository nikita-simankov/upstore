package order

import "time"

type Order struct {
	ID              string    `json:"id"`
	StoreID         string    `json:"store_id"`
	CustomerEmail   string    `json:"customer_email"`
	CustomerName    string    `json:"customer_name"`
	CustomerPhone   *string   `json:"customer_phone"`
	ShippingAddress string    `json:"shipping_address"`
	Total           int       `json:"total"`
	Status          string    `json:"status"`
	PaymentProvider *string   `json:"payment_provider"`
	PaymentID       *string   `json:"payment_id"`
	CreatedAt       time.Time `json:"created_at"`
	Items           []Item    `json:"items,omitempty"`
}

type Item struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderInput struct {
	StoreID         string
	CustomerEmail   string
	CustomerName    string
	CustomerPhone   *string
	ShippingAddress string
	Total           int
	PaymentProvider string
	Items           []Item
}
