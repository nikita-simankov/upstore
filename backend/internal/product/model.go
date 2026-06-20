package product

import "time"

type Product struct {
	ID          string    `json:"id"`
	StoreID     string    `json:"store_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	ImageURL    *string   `json:"image_url"`
	Published   bool      `json:"published"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       int     `json:"price"`
	Stock       int     `json:"stock"`
	ImageURL    *string `json:"image_url"`
	Published   bool    `json:"published"`
}

type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *int    `json:"price"`
	Stock       *int    `json:"stock"`
	ImageURL    *string `json:"image_url"`
	Published   *bool   `json:"published"`
}
