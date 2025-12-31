package api

import (
	"auto-clip-backend/logger"
	"os"
	"strings"
)

func resolveYouTubeCookie(l *logger.Logger) string {
	cookie := strings.TrimSpace(os.Getenv("YOUTUBE_COOKIE"))
	if cookie != "" {
		return cookie
	}

	cookieFile := strings.TrimSpace(os.Getenv("YOUTUBE_COOKIE_FILE"))
	if cookieFile == "" {
		return ""
	}

	content, err := os.ReadFile(cookieFile)
	if err != nil {
		if l != nil {
			l.Error("Failed to read cookie file", map[string]string{
				"Error": err.Error(),
				"Path":  cookieFile,
			})
		}
		return ""
	}

	rawCookie := strings.TrimSpace(string(content))
	if converted, ok := convertNetscapeCookieToHeader(rawCookie); ok {
		return converted
	}
	return rawCookie
}
