package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	// Create a temporary directory for logs
	tmpDir := t.TempDir()

	config := Config{
		Channel:       ChannelDaily,
		LogDir:        tmpDir,
		RetentionDays: 5,
		AppName:       "test-app",
	}
	l := New(config)

	title := "Test Event"
	details := map[string]string{
		"Key1": "Value1",
		"Key2": "Value2",
	}

	if err := l.Info(title, details); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Calculate expected filename
	dateStr := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(tmpDir, "log-"+dateStr+".log")

	// Verify file exists
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify content structure
	// Expected: [Timestamp] test-app.INFO: Test Event {"Key1":"Value1","Key2":"Value2"}
	// JSON order is not guaranteed, so we check parts

	if !strings.Contains(logContent, "test-app.INFO: Test Event") {
		t.Errorf("Log content missing formatted header. Got:\n%s", logContent)
	}
	if !strings.Contains(logContent, "\"Key1\":\"Value1\"") {
		t.Errorf("Log content missing Key1. Got:\n%s", logContent)
	}
	if !strings.Contains(logContent, "\"Key2\":\"Value2\"") {
		t.Errorf("Log content missing Key2. Got:\n%s", logContent)
	}
}

func TestRetention(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an old file
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	oldFile := filepath.Join(tmpDir, "log-"+oldDate+".log")
	if err := os.WriteFile(oldFile, []byte("old log"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}

	// Create a new file
	newDate := time.Now().Format("2006-01-02")
	newFile := filepath.Join(tmpDir, "log-"+newDate+".log")
	if err := os.WriteFile(newFile, []byte("new log"), 0644); err != nil {
		t.Fatalf("Failed to create new log file: %v", err)
	}

	config := Config{
		Channel:       ChannelDaily,
		LogDir:        tmpDir,
		RetentionDays: 5,
		AppName:       "test-app",
	}

	// Initializing logger should trigger cleanup
	_ = New(config)

	// Allow some time for goroutine
	time.Sleep(100 * time.Millisecond)

	// Check if old file is gone
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("Old log file was not deleted: %s", oldFile)
	}

	// Check if new file is still there
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("New log file was deleted incorrectly: %s", newFile)
	}
}
