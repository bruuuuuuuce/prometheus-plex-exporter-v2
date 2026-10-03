package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-kit/log/level"
)

func TestNewLoggerFiltersAtConfiguredLevel(t *testing.T) {
	tests := []struct {
		name           string
		minimumLevel   string
		wantVisible    []string
		wantSuppressed []string
	}{
		{
			name:           "defaults to info",
			minimumLevel:   "",
			wantVisible:    []string{"level=info", "level=warn", "level=error"},
			wantSuppressed: []string{"debug-entry"},
		},
		{
			name:           "info",
			minimumLevel:   "INFO",
			wantVisible:    []string{"level=info", "level=warn", "level=error"},
			wantSuppressed: []string{"debug-entry"},
		},
		{
			name:         "debug",
			minimumLevel: " DEBUG ",
			wantVisible:  []string{"level=debug", "level=info", "level=warn", "level=error"},
		},
		{
			name:           "warn",
			minimumLevel:   "warn",
			wantVisible:    []string{"level=warn", "level=error"},
			wantSuppressed: []string{"debug-entry", "info-entry"},
		},
		{
			name:           "error",
			minimumLevel:   "error",
			wantVisible:    []string{"level=error"},
			wantSuppressed: []string{"debug-entry", "info-entry", "warn-entry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger, err := newLogger(tt.minimumLevel, &output)
			if err != nil {
				t.Fatalf("newLogger() error = %v", err)
			}

			_ = level.Debug(logger).Log("msg", "debug-entry")
			_ = level.Info(logger).Log("msg", "info-entry")
			_ = level.Warn(logger).Log("msg", "warn-entry")
			_ = level.Error(logger).Log("msg", "error-entry")

			for _, expected := range tt.wantVisible {
				if !strings.Contains(output.String(), expected) {
					t.Errorf("output %q does not contain %q", output.String(), expected)
				}
			}
			for _, suppressed := range tt.wantSuppressed {
				if strings.Contains(output.String(), suppressed) {
					t.Errorf("output %q unexpectedly contains %q", output.String(), suppressed)
				}
			}
		})
	}
}

func TestNewLoggerDefaultsUnleveledEntriesToInfo(t *testing.T) {
	var output bytes.Buffer
	logger, err := newLogger("info", &output)
	if err != nil {
		t.Fatalf("newLogger() error = %v", err)
	}
	if err := logger.Log("msg", "plain-entry"); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "level=info") || !strings.Contains(got, "plain-entry") {
		t.Errorf("output = %q, want info level and message", got)
	}
}

func TestNewLoggerRejectsInvalidLevel(t *testing.T) {
	_, err := newLogger("trace", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "invalid LOG_LEVEL") {
		t.Fatalf("newLogger() error = %v, want invalid LOG_LEVEL error", err)
	}
}
