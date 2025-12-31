package api

import "strings"

// convertNetscapeCookieToHeader converts a Netscape cookie file content into a Cookie header value.
// It returns the converted header and true when at least one cookie is parsed.
func convertNetscapeCookieToHeader(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	lines := strings.Split(raw, "\n")
	cookies := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#HttpOnly_") {
			line = strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}

		name := strings.TrimSpace(parts[5])
		value := strings.TrimSpace(strings.Join(parts[6:], "\t"))
		if name == "" {
			continue
		}
		cookies = append(cookies, name+"="+value)
	}

	if len(cookies) == 0 {
		return "", false
	}

	return strings.Join(cookies, "; "), true
}
