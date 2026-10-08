package mail

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testLink = "https://app.test/verify-email?token=secret-token-value"

// newResendTestMailer points a ResendMailer at a test server. The handler decides the response.
func newResendTestMailer(t *testing.T, handler http.HandlerFunc) *ResendMailer {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	m := NewResendMailer("re_test_key", "Upstore <no-reply@upstore.test>")
	m.baseURL = server.URL
	return m
}

// TestResendSendsExpectedRequest tests the method, path, auth header, and body of the request.
func TestResendSendsExpectedRequest(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotType string
	var gotBody map[string]any

	m := newResendTestMailer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth, gotType = r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	})

	if err := m.SendVerification(context.Background(), "ivan@mail.by", testLink); err != nil {
		t.Fatalf("SendVerification: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/emails" {
		t.Errorf("request = %s %s, want POST /emails", gotMethod, gotPath)
	}
	if gotAuth != "Bearer re_test_key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotType != "application/json" {
		t.Errorf("Content-Type = %q", gotType)
	}
	if gotBody["from"] != "Upstore <no-reply@upstore.test>" {
		t.Errorf("from = %v", gotBody["from"])
	}
	to, _ := gotBody["to"].([]any)
	if len(to) != 1 || to[0] != "ivan@mail.by" {
		t.Errorf("to = %v", gotBody["to"])
	}
	if text, _ := gotBody["text"].(string); !strings.Contains(text, testLink) {
		t.Errorf("text does not contain the link: %q", text)
	}
}

// TestResendReportsProviderFailure tests that a non-2xx response is an error, and that the
// error does not contain the link.
func TestResendReportsProviderFailure(t *testing.T) {
	m := newResendTestMailer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"API key is invalid"}`))
	})

	err := m.SendVerification(context.Background(), "ivan@mail.by", testLink)
	if err == nil {
		t.Fatal("SendVerification returned nil on a 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error does not mention the status: %v", err)
	}
	if strings.Contains(err.Error(), "secret-token-value") || strings.Contains(err.Error(), "re_test_key") {
		t.Errorf("error leaks a secret: %v", err)
	}
}
