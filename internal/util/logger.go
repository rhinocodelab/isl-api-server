package util

import (
	"log"
	"os"
	"path/filepath"
)

// Logger wraps the standard log package with structured logging
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	logFile     *os.File
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.LstdFlags|log.Lshortfile),
	}
}

// NewLoggerWithFile creates a new logger instance with file logging
func NewLoggerWithFile(logPath string) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Create multi-writer to write to both file and console
	multiWriter := logFile

	return &Logger{
		infoLogger:  log.New(multiWriter, "INFO: ", log.LstdFlags|log.Lshortfile),
		errorLogger: log.New(multiWriter, "ERROR: ", log.LstdFlags|log.Lshortfile),
		fatalLogger: log.New(multiWriter, "FATAL: ", log.LstdFlags|log.Lshortfile),
		logFile:     logFile,
	}, nil
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.infoLogger.Printf(msg+" %v", args...)
	} else {
		l.infoLogger.Println(msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.errorLogger.Printf(msg+" %v", args...)
	} else {
		l.errorLogger.Println(msg)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.fatalLogger.Printf(msg+" %v", args...)
	} else {
		l.fatalLogger.Println(msg)
	}
	os.Exit(1)
}

// Close closes the log file if it was opened
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// GetLogFile returns the log file for external use
func (l *Logger) GetLogFile() *os.File {
	return l.logFile
}
