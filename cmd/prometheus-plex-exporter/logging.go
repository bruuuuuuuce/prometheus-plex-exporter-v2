package main

import (
	"fmt"
	"io"
	"strings"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func newLogger(minimumLevel string, output io.Writer) (kitlog.Logger, error) {
	if strings.TrimSpace(minimumLevel) == "" {
		minimumLevel = "info"
	}

	parsedLevel, err := level.Parse(minimumLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: expected debug, info, warn, or error", minimumLevel)
	}

	baseLogger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(output))
	filteredLogger := level.NewFilter(baseLogger, level.Allow(parsedLevel))
	return level.NewInjector(filteredLogger, level.InfoValue()), nil
}
