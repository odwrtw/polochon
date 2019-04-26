package configuration

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger represents a logger
type Logger struct {
	logger *logrus.Logger
}

// UnmarshalYAML implements the Unmarshaler interface
func (l *Logger) UnmarshalYAML(unmarshal func(interface{}) error) error {
	params := struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	}{}

	if err := unmarshal(&params); err != nil {
		return err
	}

	// Create a new logger
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}

	// Get the log level
	logLevel, err := logrus.ParseLevel(params.Level)
	if err != nil {
		return err
	}
	logger.Level = logLevel

	// Setup the output file
	var logOut io.Writer
	if params.File == "" {
		logOut = os.Stderr
	} else {
		var err error
		logOut, err = os.OpenFile(params.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return err
		}
	}
	logger.Out = logOut

	l.logger = logger

	return nil
}
