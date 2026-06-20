package store

import "time"

type Store struct {
	ID          string    `json:"id"`
	MerchantID  string    `json:"merchant_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	LogoURL     *string   `json:"logo_url"`
	Plan        string    `json:"plan"`
	TrialEndsAt time.Time `json:"trial_ends_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInput struct {
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	LogoURL *string `json:"logo_url"`
}

type UpdateInput struct {
	Name    *string `json:"name"`
	Slug    *string `json:"slug"`
	LogoURL *string `json:"logo_url"`
}
