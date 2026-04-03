package configuration

import (
	"io"
	"log/slog"
	"os"
)

// Logger represents a logger
type Logger struct {
	logger *slog.Logger
}

// UnmarshalYAML implements the Unmarshaler interface
func (l *Logger) UnmarshalYAML(unmarshal func(any) error) error {
	params := struct {
		Level            string `yaml:"level"`
		File             string `yaml:"file"`
		DisableTimestamp bool   `yaml:"disable_timestamp"`
	}{}

	if err := unmarshal(&params); err != nil {
		return err
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(params.Level)); err != nil {
		return err
	}

	var logOut io.Writer
	if params.File == "" {
		logOut = os.Stderr
	} else {
		f, err := os.OpenFile(params.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return err
		}
		logOut = f
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	if params.DisableTimestamp {
		opts.ReplaceAttr = func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		}
	}

	l.logger = slog.New(slog.NewTextHandler(logOut, opts))
	return nil
}
