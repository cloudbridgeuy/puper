package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

func init() {
	Logger = log.New(os.Stderr)
}

// Verbose enables verbose mode for the logger
func Verbose() {
	Logger.SetLevel(log.DebugLevel)
}

// Debug logs a debug message.
func Debug(msg interface{}, keyvals ...interface{}) {
	Logger.Debug(msg, keyvals)
}

// Info logs an info message.
func Info(msg interface{}, keyvals ...interface{}) {
	Logger.Info(msg, keyvals)
}

// Error logs an error message.
func Error(msg interface{}, keyvals ...interface{}) {
	Logger.Error(msg, keyvals)
}
