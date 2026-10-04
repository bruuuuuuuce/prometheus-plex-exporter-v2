package plex

import (
	"io"
	"log/slog"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// SetLogger configures the structured logger used by the client. Passing nil
// disables logging. Logging is disabled by default.
func (p *Plex) SetLogger(logger *slog.Logger) {
	p.Logger = logger
}

func (p *Plex) logger() *slog.Logger {
	if p.Logger != nil {
		return p.Logger
	}

	return discardLogger
}

// SetLogger configures the structured logger used by the webhook handler.
// Passing nil disables logging. Logging is disabled by default.
func (wh *WebhookEvents) SetLogger(logger *slog.Logger) {
	wh.Logger = logger
}

func (wh *WebhookEvents) logger() *slog.Logger {
	if wh.Logger != nil {
		return wh.Logger
	}

	return discardLogger
}
