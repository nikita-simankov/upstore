package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// resendBaseURL is the Resend API root. It can be overridden in tests.
const resendBaseURL = "https://api.resend.com"

// ResendMailer sends verification emails through the Resend API (https://resend.com).
type ResendMailer struct {
	apiKey  string
	from    string
	baseURL string
	client  *http.Client
}

// NewResendMailer returns a mailer that sends from the given address with the given API key.
// The key is never logged or included in errors.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		apiKey:  apiKey,
		from:    from,
		baseURL: resendBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type resendEmail struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

// SendVerification emails the link to the recipient. The link is a working credential, so it is
// included in the request body only and never written to a log or an error.
func (m *ResendMailer) SendVerification(ctx context.Context, to, link string) error {
	body, err := json.Marshal(resendEmail{
		From:    m.from,
		To:      []string{to},
		Subject: "Confirm your Upstore email",
		Text: "Welcome to Upstore.\n\n" +
			"Confirm your email address by opening this link within 24 hours:\n\n" +
			link + "\n\n" +
			"If you did not create an account, you can ignore this message.",
	})
	if err != nil {
		return fmt.Errorf("encode email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Read a little of the body for diagnosis. It is provider text and does not contain the link.
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend returned status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}
	return nil
}
