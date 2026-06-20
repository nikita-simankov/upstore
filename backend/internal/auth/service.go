package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/internal/merchant"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db        *pgxpool.Pool
	jwtSecret string
}

func NewService(db *pgxpool.Pool, jwtSecret string) *Service {
	return &Service{db: db, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, email, password, name string) (*merchant.Merchant, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	var m merchant.Merchant
	err = s.db.QueryRow(ctx,
		`INSERT INTO merchants (email, password_hash, name) VALUES ($1, $2, $3)
		 RETURNING id, email, name, created_at`,
		email, string(hash), name,
	).Scan(&m.ID, &m.Email, &m.Name, &m.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("email already registered")
	}

	token, err := s.issueToken(m.ID)
	if err != nil {
		return nil, "", err
	}
	return &m, token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*merchant.Merchant, string, error) {
	var m merchant.Merchant
	var hash string
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at FROM merchants WHERE email = $1`,
		email,
	).Scan(&m.ID, &m.Email, &m.Name, &hash, &m.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	token, err := s.issueToken(m.ID)
	if err != nil {
		return nil, "", err
	}
	return &m, token, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*merchant.Merchant, error) {
	var m merchant.Merchant
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, created_at FROM merchants WHERE id = $1`, id,
	).Scan(&m.ID, &m.Email, &m.Name, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("merchant not found")
	}
	return &m, nil
}

func (s *Service) issueToken(merchantID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": merchantID,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
