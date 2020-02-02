package mocks

import (
	"fmt"

	"github.com/mainflux/mainflux/logger"
)

// Logger mock implementation of logger.Logger interface
type Logger struct{}

var _ logger.Logger = Logger{}

// Debug logs any object in JSON format on debug level.
func (l Logger) Debug(msg string) {
	fmt.Printf("Debug: %s\n", msg)
}

// Info logs any object in JSON format on info level.
func (l Logger) Info(msg string) {
	fmt.Printf("Info: %s\n", msg)
}

// Warn logs any object in JSON format on warning level.
func (l Logger) Warn(msg string) {
	fmt.Printf("Warn: %s\n", msg)
}

// Error logs any object in JSON format on error level.
func (l Logger) Error(msg string) {
	fmt.Printf("Error: %s\n", msg)
}
