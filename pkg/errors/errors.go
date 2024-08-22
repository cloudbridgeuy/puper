package errors

import (
	"errors"
	"io"
	"os"

	"github.com/cloudbridgeuy/puper/pkg/logger"
	"github.com/cloudbridgeuy/puper/pkg/term"
)

// HandleAsPuperError logs an error message and returns an error.
func HandleAsPuperError(err error, reason string) {
	HandleError(NewPuperError(err, reason))
}

// HandleError logs an error message and returns an error.
func HandleError(err error) {
	// exhaust stdin
	if !term.IsInputTTY() {
		_, _ = io.ReadAll(os.Stdin)
	}

	format := "\n%s\n\n%s\n\n"

	var args []interface{}
	var perr PuperError

	if errors.As(err, &perr) {
		args = []interface{}{
			term.StderrStyles().ErrPadding.Render(term.StderrStyles().ErrorHeader.String(), perr.Reason()),
			term.StderrStyles().ErrPadding.Render(term.StderrStyles().ErrorDetails.Render(perr.Error())),
		}
	} else {
		args = []interface{}{
			term.StderrStyles().ErrPadding.Render(term.StderrStyles().ErrorDetails.Render(err.Error())),
		}
	}

	logger.Logger.Printf(format, args...)
}

// PuperError is a wrapper around an error that adds additional context.
type PuperError struct {
	err    error
	reason string
}

// NewPuperError creates a new PuperError.
func NewPuperError(err error, reason string) PuperError {
	return PuperError{err, reason}
}

// Error returns the error message.
func (m PuperError) Error() string {
	return m.err.Error()
}

// Reason returns the reason for the error.
func (m PuperError) Reason() string {
	return m.reason
}
