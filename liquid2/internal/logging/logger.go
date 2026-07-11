package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

const LevelTrace = slog.LevelDebug - 4

type Config struct {
	Level     string
	Format    string
	AddSource bool
}

func New(w io.Writer, config Config) (*slog.Logger, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	format, err := ParseFormat(config.Format)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{
		AddSource:   config.AddSource,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}
	if format == FormatJSON {
		return slog.New(slog.NewJSONHandler(w, options)), nil
	}
	return slog.New(slog.NewTextHandler(w, options)), nil
}

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level %q", value)
	}
}

func ParseFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", FormatJSON:
		return FormatJSON, nil
	case FormatText:
		return FormatText, nil
	default:
		return "", fmt.Errorf("invalid log format %q", value)
	}
}

func replaceAttr(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.LevelKey {
		return attr
	}
	level, ok := attr.Value.Any().(slog.Level)
	if !ok {
		return attr
	}
	return slog.String(slog.LevelKey, LevelName(level))
}

func LevelName(level slog.Level) string {
	switch {
	case level <= LevelTrace:
		return "trace"
	case level < slog.LevelInfo:
		return "debug"
	case level < slog.LevelWarn:
		return "info"
	case level < slog.LevelError:
		return "warn"
	default:
		return "error"
	}
}
