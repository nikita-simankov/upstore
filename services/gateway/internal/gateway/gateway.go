// Package gateway is the public entry point. It verifies access tokens, passes auth requests to
// the accounts service, and forwards other requests only when their token is valid.
package gateway

import (
	"crypto/ed25519"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/nikita-simankov/upstore/shared/authtoken"
)

// accountIDHeader carries the verified account to upstream services. A client can send this
// header too, so the gateway always removes the client's copy before setting its own.
const accountIDHeader = "X-Account-ID"

// Gateway routes requests to upstream services.
type Gateway struct {
	publicKey ed25519.PublicKey
	now       func() time.Time
	mux       *http.ServeMux
}

// New returns a Gateway that verifies tokens with publicKey and forwards auth requests to accounts.
func New(publicKey ed25519.PublicKey, accounts *url.URL) *Gateway {
	g := &Gateway{publicKey: publicKey, now: time.Now, mux: http.NewServeMux()}

	g.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Sign-in, registration, refresh, logout, and verification do not need an access token.
	// The accounts service checks what each request needs.
	g.mux.Handle("/v1/auth/", proxy(accounts, false))
	return g
}

// Protect forwards requests under prefix to target, but only for a request whose access token is
// valid. The verified account ID is sent to the upstream in X-Account-ID.
func (g *Gateway) Protect(prefix string, target *url.URL) {
	g.mux.Handle(prefix, g.requireToken(proxy(target, true)))
}

// Handler returns the gateway's HTTP handler.
func (g *Gateway) Handler() http.Handler {
	return g.mux
}

// requireToken rejects requests without a valid bearer token and passes the others on.
func (g *Gateway) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			unauthorized(w)
			return
		}
		claims, err := authtoken.Verify(g.publicKey, token, g.now())
		if err != nil {
			unauthorized(w)
			return
		}
		r.Header.Set(accountIDHeader, claims.AccountID)
		next.ServeHTTP(w, r)
	})
}

// proxy returns a reverse proxy to target.
//
// For a protected route, requireToken has already replaced the client's account header with the
// verified one, so the upstream receives it. The Authorization header is removed, because the
// upstream trusts the gateway's header and does not check the token again.
//
// For an auth route, nothing is verified, so the account header is removed. A client cannot set it.
func proxy(target *url.URL, protected bool) http.Handler {
	p := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			if protected {
				pr.Out.Header.Del("Authorization")
			} else {
				pr.Out.Header.Del(accountIDHeader)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "upstream request failed", "path", r.URL.Path, "error", err)
			writeError(w, http.StatusBadGateway, "service unavailable")
		},
	}
	return p
}

// bearerToken returns the token from an "Authorization: Bearer ..." header.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	writeError(w, http.StatusUnauthorized, "missing or invalid access token")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
