// Package httpapi exposes registration and sign-in over HTTP.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/time/rate"

	"github.com/nikita-simankov/upstore/services/accounts/internal/auth"
	"github.com/nikita-simankov/upstore/shared/events"
)

// maxBodyBytes caps request bodies. Registration and sign-in bodies are a few hundred bytes,
// so the cap is generous and still blocks large uploads.
const maxBodyBytes = 4096

// maxEmailLength is the RFC 5321 limit for an email address.
const maxEmailLength = 254

// Config sets the rate limits for the sign-in and registration endpoints.
type Config struct {
	// LoginPerIP is the number of sign-in attempts allowed per client IP per minute, with a burst.
	LoginPerIP rate.Limit
	LoginBurst int
	// LoginPerEmail limits sign-in attempts for one email per minute, with a burst. It is looser
	// than the per-IP limit, because a tight per-email limit lets an attacker lock out a victim.
	LoginPerEmail rate.Limit
	EmailBurst    int
	// RegisterPerIP limits registrations per client IP per minute, with a burst.
	RegisterPerIP rate.Limit
	RegisterBurst int
}

// DefaultConfig returns the limits used in production.
func DefaultConfig() Config {
	return Config{
		LoginPerIP:    rate.Every(6 * time.Second), // 10 per minute
		LoginBurst:    10,
		LoginPerEmail: rate.Every(15 * time.Second), // 4 per minute
		EmailBurst:    20,
		RegisterPerIP: rate.Every(12 * time.Second), // 5 per minute
		RegisterBurst: 5,
	}
}

// API serves the auth endpoints.
type API struct {
	auth       *auth.Service
	sessions   *auth.Sessions
	loginIP    *keyedLimiter
	loginEmail *keyedLimiter
	registerIP *keyedLimiter
}

// New returns an API that uses the given services and limits.
func New(service *auth.Service, sessions *auth.Sessions, cfg Config) *API {
	now := time.Now
	return &API{
		auth:       service,
		sessions:   sessions,
		loginIP:    newKeyedLimiter(cfg.LoginPerIP, cfg.LoginBurst, now),
		loginEmail: newKeyedLimiter(cfg.LoginPerEmail, cfg.EmailBurst, now),
		registerIP: newKeyedLimiter(cfg.RegisterPerIP, cfg.RegisterBurst, now),
	}
}

// Routes returns the handler for the auth endpoints.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/register", a.register)
	mux.HandleFunc("POST /v1/auth/login", a.login)
	mux.HandleFunc("POST /v1/auth/refresh", a.refresh)
	mux.HandleFunc("POST /v1/auth/logout", a.logout)
	return mux
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	AccountType string `json:"account_type"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type accountResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// sessionResponse is returned by sign-in: the account and a new session's tokens.
type sessionResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// tokenResponse is returned by refresh.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	if !a.registerIP.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many registration attempts, try again later")
		return
	}

	var req registerRequest
	if !decodeBody(w, r, &req) {
		return
	}

	email := strings.TrimSpace(req.Email)
	if !validEmail(email) {
		writeError(w, http.StatusBadRequest, "email is not valid")
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	accountType := events.AccountType(req.AccountType)
	if !accountType.Valid() {
		writeError(w, http.StatusBadRequest, "account_type must be seller or shopper")
		return
	}

	account, err := a.auth.Register(r.Context(), email, req.Password, accountType)
	switch {
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email is already registered")
	case err != nil:
		slog.ErrorContext(r.Context(), "register failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		writeJSON(w, http.StatusCreated, accountResponse{ID: formatUUID(account.ID), Status: string(account.Status)})
	}
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeBody(w, r, &req) {
		return
	}
	email := strings.TrimSpace(req.Email)

	if !a.loginIP.Allow(clientIP(r)) || !a.loginEmail.Allow(strings.ToLower(email)) {
		writeError(w, http.StatusTooManyRequests, "too many sign-in attempts, try again later")
		return
	}

	account, err := a.auth.Login(r.Context(), email, req.Password)
	if err != nil {
		a.writeAuthError(w, r, err)
		return
	}

	tok, err := a.sessions.Issue(r.Context(), account)
	if err != nil {
		slog.ErrorContext(r.Context(), "issue session failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{
		ID:           formatUUID(account.ID),
		Status:       string(account.Status),
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
	})
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	if !a.loginIP.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	var req refreshRequest
	if !decodeBody(w, r, &req) {
		return
	}

	tok, err := a.sessions.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, auth.ErrInvalidRefreshToken.Error())
			return
		}
		a.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
	})
}

// logout always answers 204, whether or not the refresh token was valid, so it reveals nothing.
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if err := a.sessions.Logout(r.Context(), req.RefreshToken); err != nil {
		slog.ErrorContext(r.Context(), "logout failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeAuthError maps the sign-in and refresh errors that have a client-facing meaning.
// Anything else is logged and returned as a generic 500.
func (a *API) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, auth.ErrInvalidCredentials.Error())
	case errors.Is(err, auth.ErrAccountBanned):
		writeError(w, http.StatusForbidden, auth.ErrAccountBanned.Error())
	case errors.Is(err, auth.ErrAccountNotActive):
		writeError(w, http.StatusForbidden, auth.ErrAccountNotActive.Error())
	default:
		slog.ErrorContext(r.Context(), "auth request failed with internal error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// decodeBody reads one JSON object from a size-capped body. Unknown fields are rejected.
// It writes the error response and returns false if the body is not acceptable.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "request body must be a JSON object with known fields")
		return false
	}
	return true
}

// validEmail is a basic shape check. The database and the sign-in path are the real checks.
func validEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	return len(email) > 0 && len(email) <= maxEmailLength && at > 0 && at < len(email)-1
}

// clientIP returns the address of the TCP peer. X-Forwarded-For is not used, because a client
// can set it. Behind a trusted proxy, the proxy's address must be taken from its own config.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// formatUUID returns the canonical 8-4-4-4-12 form of a database UUID.
func formatUUID(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
