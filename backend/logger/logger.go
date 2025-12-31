package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of the log
type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Notice
	Warning
	Error
	Critical
	Alert
	Emergency
)

func (l LogLevel) String() string {
	return [...]string{"DEBUG", "INFO", "NOTICE", "WARNING", "ERROR", "CRITICAL", "ALERT", "EMERGENCY"}[l]
}

// LogChannel represents the output destination
type LogChannel string

const (
	ChannelSingle LogChannel = "single"
	ChannelDaily  LogChannel = "daily"
	ChannelStderr LogChannel = "stderr"
	// Stack and others can be implemented by combining these
)

// Config holds the logger configuration
type Config struct {
	Channel       LogChannel
	LogDir        string
	RetentionDays int    // Used for daily channel
	AppName       string // e.g., "local", "production"
}

type Logger struct {
	config Config
	mu     sync.Mutex
}

// New creates a new Logger instance
func New(config Config) *Logger {
	if config.LogDir == "" {
		config.LogDir = "logs"
	}
	if config.RetentionDays <= 0 {
		config.RetentionDays = 5 // Default retention
	}
	if config.AppName == "" {
		config.AppName = "local"
	}

	l := &Logger{
		config: config,
	}

	// Ensure log directory exists if we are writing to files
	if config.Channel == ChannelSingle || config.Channel == ChannelDaily {
		_ = os.MkdirAll(config.LogDir, 0755)
	}

	// Run cleanup immediately if daily
	if config.Channel == ChannelDaily {
		go l.cleanOldLogs()
	}

	return l
}

// Helper methods for each level
func (l *Logger) Debug(message string, context map[string]string) error {
	return l.log(Debug, message, context)
}

func (l *Logger) Info(message string, context map[string]string) error {
	return l.log(Info, message, context)
}

func (l *Logger) Notice(message string, context map[string]string) error {
	return l.log(Notice, message, context)
}

func (l *Logger) Warning(message string, context map[string]string) error {
	return l.log(Warning, message, context)
}

func (l *Logger) Error(message string, context map[string]string) error {
	return l.log(Error, message, context)
}

func (l *Logger) Critical(message string, context map[string]string) error {
	return l.log(Critical, message, context)
}

func (l *Logger) Alert(message string, context map[string]string) error {
	return l.log(Alert, message, context)
}

func (l *Logger) Emergency(message string, context map[string]string) error {
	return l.log(Emergency, message, context)
}

// log handles the core logging logic
func (l *Logger) log(level LogLevel, message string, context map[string]string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05")

	// Format: [Timestamp] AppName.LEVEL: Message {Context}
	// Example: [2025-12-29 12:00:00] local.INFO: Server Started {"Port": "8888"}

	contextStr := "{}"
	if len(context) > 0 {
		jsonBytes, err := json.Marshal(context)
		if err == nil {
			contextStr = string(jsonBytes)
		} else {
			// Fallback if JSON fails
			contextStr = fmt.Sprintf("%v", context)
		}
	}

	caller := ""
	if level >= Error {
		caller = callerLocation()
	}

	logLine := fmt.Sprintf("[%s] %s.%s: %s %s\n", timestamp, l.config.AppName, level.String(), message, contextStr)
	if caller != "" {
		logLine = fmt.Sprintf("[%s] %s.%s: (%s) %s %s\n", timestamp, l.config.AppName, level.String(), caller, message, contextStr)
	}

	// Output to Console (Stderr) - always do this or generic writer?
	// The user asked for "stderr" type, but also "log nya juga di serve nya" (console).
	// Let's print to console by default for visibility in CLI/Docker
	fmt.Print(logLine)

	// Output to File based on Channel
	if l.config.Channel == ChannelStderr {
		return nil // Already printed
	}

	var fileName string
	if l.config.Channel == ChannelDaily {
		dateStr := now.Format("2006-01-02")
		fileName = filepath.Join(l.config.LogDir, fmt.Sprintf("log-%s.log", dateStr))
	} else if l.config.Channel == ChannelSingle {
		fileName = filepath.Join(l.config.LogDir, "log.log")
	} else {
		return nil // Unknown channel
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(logLine); err != nil {
		return err
	}

	return nil
}

func callerLocation() string {
	pcs := make([]uintptr, 10)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if frame.File != "" && !strings.HasSuffix(frame.File, "logger.go") {
			return fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
		}
		if !more {
			break
		}
	}

	return ""
}

// cleanOldLogs removes logs older than RetentionDays
func (l *Logger) cleanOldLogs() {
	files, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "log-") && strings.HasSuffix(file.Name(), ".log") {
			logFiles = append(logFiles, filepath.Join(l.config.LogDir, file.Name()))
		}
	}

	// If we have fewer files than retention, nothing to do
	// Wait, retention is based on DATE, not count of files usually, but simpler is count or date parsing.
	// User said "5 days", usually implies age.
	// Let's check file modification time or parse filename. Filename is safer.

	cutoff := time.Now().AddDate(0, 0, -l.config.RetentionDays)

	for _, path := range logFiles {
		filename := filepath.Base(path)
		// Expected: log-2025-12-29.log
		// Remove prefix and suffix
		datePart := strings.TrimSuffix(strings.TrimPrefix(filename, "log-"), ".log")

		fileDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue // Skip malformed filenames
		}

		if fileDate.Before(cutoff) {
			_ = os.Remove(path)
		}
	}
}
