// Package mail sends account emails. Only the interface and a development implementation live
// here. A production provider must implement Mailer before the service can run outside development.
package mail

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Mailer sends the email that carries a verification link.
// Implementations must not log the link in production, because it is a credential.
type Mailer interface {
	SendVerification(ctx context.Context, to, link string) error
}

// DevLogMailer writes verification links to a writer, which is stdout by default.
// It exists so that verification can be tested locally. It must never be used in production:
// the link is a working credential, and this writes it to the log.
type DevLogMailer struct {
	Out io.Writer
}

// NewDevLogMailer returns a DevLogMailer that writes to standard output.
func NewDevLogMailer() DevLogMailer {
	return DevLogMailer{Out: os.Stdout}
}

// SendVerification prints the link. The recipient is shown so the output is easy to match up.
func (m DevLogMailer) SendVerification(_ context.Context, to, link string) error {
	_, err := fmt.Fprintf(m.Out, "[DEV ONLY] verification link for %s: %s\n", to, link)
	return err
}
