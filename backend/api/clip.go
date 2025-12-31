package api

import (
	"auto-clip-backend/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

type ClipRequest struct {
	URL   string  `json:"url"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type ClipResponse struct {
	OutputPath     string `json:"output_path"`
	ThumbnailPath  string `json:"thumbnail_path,omitempty"`
	ThumbnailError string `json:"thumbnail_error,omitempty"`
	Duration       string `json:"duration"`
}

type Handler struct {
	Logger *logger.Logger
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers http.Header
}

func (rt headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, vals := range rt.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	return rt.base.RoundTrip(req)
}

func (h *Handler) Clip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse Request
	var req ClipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	duration := req.End - req.Start
	if duration <= 0 {
		http.Error(w, "End time must be greater than start time", http.StatusBadRequest)
		return
	}

	// 2. Log the incoming request
	h.Logger.Info("Clip Request Received", map[string]string{
		"URL":   req.URL,
		"Start": fmt.Sprintf("%.2f", req.Start),
		"End":   fmt.Sprintf("%.2f", req.End),
		"IP":    r.RemoteAddr,
	})

	// 3. Get Stream URL using go-youtube library
	userAgent := resolveUserAgent()
	cookie := resolveYouTubeCookie(h.Logger)

	defaultHeaders := http.Header{}
	defaultHeaders.Set("User-Agent", userAgent)
	if cookie != "" {
		defaultHeaders.Set("Cookie", cookie)
	}

	// Proxy handling: disable proxies by default to avoid corporate proxy 403s; allow override via YOUTUBE_PROXY or YOUTUBE_DISABLE_PROXY=false
	proxyCfg := resolveProxyConfig(h.Logger)
	logProxySettings(h.Logger, proxyCfg)

	client := youtube.Client{
		HTTPClient: &http.Client{
			Transport: headerRoundTripper{
				base:    proxyCfg.baseTransport,
				headers: defaultHeaders,
			},
		},
	}
	video, err := client.GetVideo(req.URL)
	if err != nil {
		h.Logger.Error("Failed to get video info", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to get video info", http.StatusInternalServerError)
		return
	}

	// Find best format with both audio and video
	var format *youtube.Format

	// Sort formats by quality (optional, but library formats are usually unsorted)
	// For simplicity, we just look for the first decent match: mp4 with audio
	for _, f := range video.Formats {
		if f.MimeType != "" && strings.Contains(f.MimeType, "video/mp4") && f.AudioChannels > 0 {
			format = &f
			break
		}
	}

	// Fallback: if no mp4 with audio, just take the first format that has audio and video
	if format == nil {
		for _, f := range video.Formats {
			if f.AudioChannels > 0 && f.Width > 0 {
				format = &f
				break
			}
		}
	}

	if format == nil {
		h.Logger.Error("No suitable format found", nil)
		http.Error(w, "No suitable video format found", http.StatusInternalServerError)
		return
	}

	streamURL, err := client.GetStreamURL(video, format)
	if err != nil {
		h.Logger.Error("Failed to get stream URL", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to get stream URL", http.StatusInternalServerError)
		return
	}

	outputDir := strings.TrimSpace(os.Getenv("CLIP_OUTPUT_DIR"))
	if outputDir == "" {
		outputDir = filepath.Join("output", "clips")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		h.Logger.Error("Failed to create output directory", map[string]string{"Error": err.Error(), "Path": outputDir})
		http.Error(w, "Failed to create output directory", http.StatusInternalServerError)
		return
	}

	clipID := strings.TrimSpace(video.ID)
	if clipID != "" {
		clipID = strings.Map(func(r rune) rune {
			switch {
			case r >= 'a' && r <= 'z':
				return r
			case r >= 'A' && r <= 'Z':
				return r
			case r >= '0' && r <= '9':
				return r
			case r == '-' || r == '_':
				return r
			default:
				return -1
			}
		}, clipID)
	}

	ts := time.Now().UnixNano()
	baseName := fmt.Sprintf("clip-%d.mp4", ts)
	if clipID != "" {
		baseName = fmt.Sprintf("clip-%s-%d.mp4", clipID, ts)
	}
	outputPath := filepath.Join(outputDir, baseName)

	// 4. Render via ffmpeg
	// -ss before -i is crucial for fast seeking.
	// -t is duration.
	// -movflags frag_keyframe+empty_moov is needed for fragmented MP4 streaming.
	// -preset ultrafast helps on low-resource VMs.
	// -user_agent mimics a browser to avoid 403s on some playback URLs.
	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}
	ffmpegArgs := []string{
		"-user_agent", userAgent,
		"-headers", ffmpegHeaders,
	}
	if proxyCfg.proxyURLStr != "" {
		ffmpegArgs = append(ffmpegArgs, "-http_proxy", proxyCfg.proxyURLStr)
	}
	ffmpegArgs = append(ffmpegArgs,
		"-ss", fmt.Sprintf("%f", req.Start),
		"-i", streamURL,
		"-t", fmt.Sprintf("%f", duration),
		"-c:v", "libx264", "-preset", "ultrafast", // Re-encode to ensure precise cuts
		"-c:a", "aac",
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		outputPath,
	)

	h.Logger.Info("FFmpeg Args", map[string]string{
		"Args": sanitizeFFmpegArgs(ffmpegArgs),
	})

	cmdFfmpeg := exec.Command("ffmpeg", ffmpegArgs...)

	// Disable proxy for ffmpeg to avoid 403 from Google Video unless explicitly set
	cmdFfmpeg.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)

	// Capture stderr for debugging (optional, but good for logs)
	var ffmpegErr bytes.Buffer
	cmdFfmpeg.Stderr = &ffmpegErr

	startRender := time.Now()
	if err := cmdFfmpeg.Run(); err != nil {
		_ = os.Remove(outputPath)
		h.Logger.Error("Clip Failed", map[string]string{
			"Error":  err.Error(),
			"Stderr": ffmpegErr.String(), // Only useful if we buffered it, which we did partially
			"Output": outputPath,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  "Failed to generate clip",
			"detail": "ffmpeg failed while rendering the clip",
		})
		return
	}

	renderDuration := time.Since(startRender)
	h.Logger.Info("Clip Success", map[string]string{
		"Duration":   renderDuration.String(),
		"OutputPath": outputPath,
	})

	response := ClipResponse{
		OutputPath: outputPath,
		Duration:   renderDuration.String(),
	}

	thumbPath, thumbErr := generateThumbnail(outputPath)
	if thumbErr != nil {
		h.Logger.Warning("Thumbnail Failed", map[string]string{
			"Error":      thumbErr.Error(),
			"OutputPath": outputPath,
		})
		response.ThumbnailError = thumbErr.Error()
	} else {
		h.Logger.Info("Thumbnail Success", map[string]string{
			"OutputPath":    outputPath,
			"ThumbnailPath": thumbPath,
		})
		response.ThumbnailPath = thumbPath
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Clip Response Failed", map[string]string{"Error": err.Error()})
	}
}

func buildFFmpegEnv(proxyURL string, disableProxy bool, systemProxy string) []string {
	var env []string
	systemNoProxy := os.Getenv("NO_PROXY")
	systemAllProxy := os.Getenv("ALL_PROXY")
	for _, e := range os.Environ() {
		key := strings.ToLower(strings.SplitN(e, "=", 2)[0])
		if key == "http_proxy" || key == "https_proxy" || key == "all_proxy" || key == "no_proxy" {
			continue
		}
		env = append(env, e)
	}

	switch {
	case proxyURL != "":
		env = append(env,
			"HTTP_PROXY="+proxyURL,
			"HTTPS_PROXY="+proxyURL,
			"NO_PROXY=",
		)
	case disableProxy:
		env = append(env,
			"HTTP_PROXY=",
			"HTTPS_PROXY=",
			"ALL_PROXY=",
			"NO_PROXY=*",
		)
	case systemProxy != "":
		env = append(env,
			"HTTP_PROXY="+systemProxy,
			"HTTPS_PROXY="+systemProxy,
		)
		if systemAllProxy != "" {
			env = append(env, "ALL_PROXY="+systemAllProxy)
		}
		if systemNoProxy != "" {
			env = append(env, "NO_PROXY="+systemNoProxy)
		}
	}

	return env
}

type proxyConfig struct {
	proxyURLStr   string
	httpProxyEnv  string
	proxyDisabled bool
	baseTransport http.RoundTripper
}

func resolveUserAgent() string {
	userAgent := strings.TrimSpace(os.Getenv("YOUTUBE_USER_AGENT"))
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	return userAgent
}

func resolveProxyConfig(l *logger.Logger) proxyConfig {
	proxyURLStr := strings.TrimSpace(os.Getenv("YOUTUBE_PROXY"))
	disableProxy := strings.ToLower(strings.TrimSpace(os.Getenv("YOUTUBE_DISABLE_PROXY")))
	httpProxyEnv := strings.TrimSpace(os.Getenv("HTTP_PROXY"))

	proxyDisabled := disableProxy == "true"
	if disableProxy == "" && proxyURLStr == "" && httpProxyEnv == "" {
		proxyDisabled = true // keep prior default: no proxy unless explicitly configured
	}

	baseTransport := http.DefaultTransport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		tCopy := t.Clone()
		switch {
		case proxyURLStr != "":
			parsed, err := url.Parse(proxyURLStr)
			if err != nil {
				if l != nil {
					l.Error("Invalid YOUTUBE_PROXY", map[string]string{"Error": err.Error(), "Value": proxyURLStr})
				}
			} else {
				tCopy.Proxy = http.ProxyURL(parsed)
				proxyDisabled = false
			}
		case proxyDisabled:
			tCopy.Proxy = nil
		case httpProxyEnv != "":
			tCopy.Proxy = http.ProxyFromEnvironment
		}
		baseTransport = tCopy
	}

	return proxyConfig{
		proxyURLStr:   proxyURLStr,
		httpProxyEnv:  httpProxyEnv,
		proxyDisabled: proxyDisabled,
		baseTransport: baseTransport,
	}
}

func logProxySettings(l *logger.Logger, cfg proxyConfig) {
	if l == nil {
		return
	}

	l.Info("Proxy Settings", map[string]string{
		"YOUTUBE_PROXY": redactProxyURL(cfg.proxyURLStr),
		"HTTP_PROXY":    redactProxyURL(cfg.httpProxyEnv),
		"ProxyDisabled": fmt.Sprintf("%t", cfg.proxyDisabled),
	})
}

func redactProxyURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "***"
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		if username == "" {
			parsed.User = url.UserPassword("***", "***")
		} else {
			parsed.User = url.UserPassword(username, "***")
		}
	}

	return parsed.String()
}

func sanitizeFFmpegArgs(args []string) string {
	sanitized := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-headers":
			sanitized = append(sanitized, arg)
			if i+1 < len(args) {
				sanitized = append(sanitized, redactFFmpegHeaders(args[i+1]))
				i++
			}
		case "-http_proxy":
			sanitized = append(sanitized, arg)
			if i+1 < len(args) {
				sanitized = append(sanitized, redactProxyURL(args[i+1]))
				i++
			}
		case "-i":
			sanitized = append(sanitized, arg)
			if i+1 < len(args) {
				sanitized = append(sanitized, redactStreamURL(args[i+1]))
				i++
			}
		default:
			sanitized = append(sanitized, arg)
		}
	}

	return strings.Join(sanitized, " ")
}

func redactFFmpegHeaders(raw string) string {
	if raw == "" {
		return ""
	}

	parts := strings.Split(raw, "\r\n")
	cleaned := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(part), "cookie:") {
			cleaned = append(cleaned, "Cookie: ***")
		} else {
			cleaned = append(cleaned, part)
		}
	}

	return strings.Join(cleaned, "; ")
}

func redactStreamURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "***"
	}

	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/...", scheme, parsed.Host)
}
