package main

import (
	"auto-clip-backend/api"
	"auto-clip-backend/logger"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	loadedEnv := loadEnvFiles()

	// Initialize Logger
	// Ensure the logs directory exists relative to execution or hardcoded
	// For this CLI usage, we assume running from 'backend/' or root.
	// Let's ensure it maps to the correct place.
	logDir := "logs"
	auditLogger := logger.New(logger.Config{
		Channel:       logger.ChannelDaily,
		LogDir:        logDir,
		RetentionDays: 5,
		AppName:       "local",
	})

	if len(loadedEnv) > 0 {
		auditLogger.Info("Loaded env file(s)", map[string]string{
			"Files": strings.Join(loadedEnv, ", "),
		})
	}

	// Initialize Handlers
	clipHandler := &api.Handler{Logger: auditLogger}

	// Setup Router
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Clip Endpoint
	mux.HandleFunc("/api/clip", clipHandler.Clip)
	mux.HandleFunc("/proxy-check", clipHandler.ProxyCheck)

	// Middleware Chain
	handler := enableCORS(mux)
	handler = loggingMiddleware(handler, auditLogger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	auditLogger.Info("Server Started", map[string]string{"Port": port})

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// enableCORS handles Cross-Origin Resource Sharing
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production security
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs basic request details
func loggingMiddleware(next http.Handler, l *logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We can't easily capture status code without a custom ResponseWriter wrapper,
		// but we can log the attempt.
		// Detailed logging is done inside the handlers for specific actions.
		next.ServeHTTP(w, r)
	})
}

// loadEnvFiles pulls key=value pairs from common .env locations into the process environment.
// It intentionally runs before server startup so PORT/YOUTUBE_* overrides apply without shell exporting.
func loadEnvFiles() []string {
	paths := []string{
		".env.local",
		".env",
		"../.env.local",
		"../.env",
	}

	var loaded []string

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("env: could not open %s: %v", path, err)
			}
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(strings.TrimPrefix(parts[0], "export "))
			val := strings.TrimSpace(parts[1])

			// Strip surrounding quotes if present.
			val = strings.Trim(val, `"'`)

			if key != "" {
				_ = os.Setenv(key, val)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("env: error reading %s: %v", path, err)
		}

		if err := f.Close(); err != nil {
			log.Printf("env: error closing %s: %v", path, err)
		}

		loaded = append(loaded, path)
	}

	return loaded
}
