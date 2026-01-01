package api

import (
	"auto-clip-backend/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
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
	URL            string          `json:"url"`
	Start          float64         `json:"start"`
	End            float64         `json:"end"`
	OutputSettings *OutputSettings `json:"output_settings,omitempty"`
}

type ClipSegmentRequest struct {
	Start       float64  `json:"start"`
	End         float64  `json:"end"`
	ThumbnailAt *float64 `json:"thumbnail_at,omitempty"`
}

type BatchClipRequest struct {
	URL            string               `json:"url"`
	Segments       []ClipSegmentRequest `json:"segments"`
	OutputSettings *OutputSettings      `json:"output_settings,omitempty"`
}

type OutputSettings struct {
	Preset      string `json:"preset,omitempty"`
	Vertical    bool   `json:"vertical,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	AspectMode  string `json:"aspect_mode,omitempty"`
	CropMethod  string `json:"crop_method,omitempty"`
	Captions    bool   `json:"captions,omitempty"`
	CaptionStyle string `json:"caption_style,omitempty"`
	CaptionSource string `json:"caption_source,omitempty"`
	CaptionLanguage string `json:"caption_language,omitempty"`
	CaptionFormat string `json:"caption_format,omitempty"`
	CaptionText string `json:"caption_text,omitempty"`
	FontSize int `json:"font_size,omitempty"`
	HighlightColor string `json:"highlight_color,omitempty"`
}

func buildVideoFilter(settings *OutputSettings) string {
	if settings == nil || !settings.Vertical {
		return ""
	}

	targetWidth := 1080
	targetHeight := 1920
	switch strings.TrimSpace(settings.AspectRatio) {
	case "1:1":
		targetHeight = 1080
	case "4:5":
		targetHeight = 1350
	case "9:16", "":
		// default
	default:
		// Unknown ratios fall back to 9:16.
	}

	cropX := "(in_w-out_w)/2"
	cropY := "(in_h-out_h)/2"
	switch strings.ToLower(strings.TrimSpace(settings.CropMethod)) {
	case "top":
		cropY = "0"
	case "bottom":
		cropY = "in_h-out_h"
	case "center", "":
		// keep default
	default:
		// "smart" or unknown values fall back to center
	}

	aspectMode := strings.ToLower(strings.TrimSpace(settings.AspectMode))
	if aspectMode == "fit" && targetHeight < 1920 {
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:%s:%s,setsar=1,pad=%d:%d:0:(%d-ih)/2:color=black",
			targetWidth,
			targetHeight,
			targetWidth,
			targetHeight,
			cropX,
			cropY,
			targetWidth,
			1920,
			1920,
		)
	}

	return fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:%s:%s,setsar=1",
		targetWidth,
		targetHeight,
		targetWidth,
		targetHeight,
		cropX,
		cropY,
	)
}

func joinFilters(filters ...string) string {
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		if strings.TrimSpace(filter) != "" {
			parts = append(parts, filter)
		}
	}
	return strings.Join(parts, ",")
}

type ClipResponse struct {
	OutputPath     string `json:"output_path"`
	ThumbnailPath  string `json:"thumbnail_path,omitempty"`
	ThumbnailError string `json:"thumbnail_error,omitempty"`
	Duration       string `json:"duration"`
}

type ClipSegmentResponse struct {
	Index          int     `json:"index"`
	Start          float64 `json:"start"`
	End            float64 `json:"end"`
	OutputPath     string  `json:"output_path,omitempty"`
	ThumbnailPath  string  `json:"thumbnail_path,omitempty"`
	ThumbnailError string  `json:"thumbnail_error,omitempty"`
	RenderDuration string  `json:"render_duration,omitempty"`
	ThumbnailAt    float64 `json:"thumbnail_at,omitempty"`
	Error          string  `json:"error,omitempty"`
}

type BatchClipResponse struct {
	Results []ClipSegmentResponse `json:"results"`
	Total   int                  `json:"total"`
	Success int                  `json:"success"`
	Failed  int                  `json:"failed"`
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

func pickClipFormat(formats youtube.FormatList) *youtube.Format {
	for _, f := range formats {
		if f.MimeType != "" && strings.Contains(f.MimeType, "video/mp4") && f.AudioChannels > 0 {
			return &f
		}
	}

	for _, f := range formats {
		if f.AudioChannels > 0 && f.Width > 0 {
			return &f
		}
	}

	return nil
}

func sanitizeVideoID(raw string) string {
	clipID := strings.TrimSpace(raw)
	if clipID == "" {
		return ""
	}

	return strings.Map(func(r rune) rune {
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

func buildClipBaseName(clipID string, ts int64, index int) string {
	base := fmt.Sprintf("clip-%d", ts)
	if clipID != "" {
		base = fmt.Sprintf("clip-%s-%d", clipID, ts)
	}
	if index > 0 {
		base = fmt.Sprintf("%s-%02d", base, index)
	}
	return base + ".mp4"
}

func buildThumbnailBaseName(clipID string, ts int64, index int) string {
	name := strings.TrimSuffix(buildClipBaseName(clipID, ts, index), ".mp4")
	return name + ".jpg"
}

func resolveThumbnailAt(duration float64, override *float64) float64 {
	thumbAt := defaultThumbnailTimestamp
	if override != nil {
		thumbAt = *override
	}
	if thumbAt < 0 {
		thumbAt = defaultThumbnailTimestamp
	}
	maxThumbAt := math.Max(duration-0.05, 0)
	if thumbAt > maxThumbAt {
		thumbAt = maxThumbAt
	}
	return thumbAt
}

func (h *Handler) getVideoStream(videoURL string) (*youtube.Video, string, string, string, proxyConfig, error) {
	userAgent := resolveUserAgent()
	cookie := resolveYouTubeCookie(h.Logger)

	defaultHeaders := http.Header{}
	defaultHeaders.Set("User-Agent", userAgent)
	if cookie != "" {
		defaultHeaders.Set("Cookie", cookie)
	}

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

	video, err := client.GetVideo(videoURL)
	if err != nil {
		return nil, "", userAgent, cookie, proxyCfg, err
	}

	format := pickClipFormat(video.Formats)
	if format == nil {
		return video, "", userAgent, cookie, proxyCfg, fmt.Errorf("no suitable video format found")
	}

	streamURL, err := client.GetStreamURL(video, format)
	if err != nil {
		return video, "", userAgent, cookie, proxyCfg, err
	}

	return video, streamURL, userAgent, cookie, proxyCfg, nil
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
	video, streamURL, userAgent, cookie, proxyCfg, err := h.getVideoStream(req.URL)
	if err != nil {
		h.Logger.Error("Failed to get stream URL", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to get stream URL", http.StatusInternalServerError)
		return
	}
	if err := validateSpeechConfig(req.OutputSettings); err != nil {
		h.Logger.Error("Caption speech config failed", map[string]string{"Error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateSpeechConfig(req.OutputSettings); err != nil {
		h.Logger.Error("Caption speech config failed", map[string]string{"Error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var captions *captionAsset
	if req.OutputSettings != nil && req.OutputSettings.Captions {
		captions, err = resolveCaptionAsset(req.OutputSettings, video.ID, userAgent, cookie, proxyCfg)
		if err != nil {
			h.Logger.Error("Caption source failed", map[string]string{"Error": err.Error()})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

	clipID := sanitizeVideoID(video.ID)

	ts := time.Now().UnixNano()
	baseName := buildClipBaseName(clipID, ts, 0)
	outputPath := filepath.Join(outputDir, baseName)
	videoFilter := buildVideoFilter(req.OutputSettings)
	subtitlePath, cleanupSubtitle, err := buildSegmentSubtitleFileWithSpeech(captions, req.OutputSettings, req.Start, req.End, streamURL, userAgent, cookie, proxyCfg)
	if err != nil {
		h.Logger.Error("Subtitle build failed", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to build subtitles", http.StatusInternalServerError)
		return
	}
	if cleanupSubtitle != nil {
		defer cleanupSubtitle()
	}
	subtitleFilter := subtitleFilterForPath(subtitlePath, req.OutputSettings)
	combinedFilter := joinFilters(videoFilter, subtitleFilter)

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
	)
	if combinedFilter != "" {
		ffmpegArgs = append(ffmpegArgs, "-vf", combinedFilter)
	}
	ffmpegArgs = append(ffmpegArgs,
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

func (h *Handler) ClipBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BatchClipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "Missing url field", http.StatusBadRequest)
		return
	}
	if len(req.Segments) == 0 {
		http.Error(w, "Missing segments field", http.StatusBadRequest)
		return
	}
	if len(req.Segments) > 500 {
		http.Error(w, "Too many segments (max 500)", http.StatusBadRequest)
		return
	}

	h.Logger.Info("Batch Clip Request Received", map[string]string{
		"URL":       req.URL,
		"Segments":  fmt.Sprintf("%d", len(req.Segments)),
		"Requester": r.RemoteAddr,
	})

	video, streamURL, userAgent, cookie, proxyCfg, err := h.getVideoStream(req.URL)
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

	clipID := sanitizeVideoID(video.ID)
	batchTimestamp := time.Now().UnixNano()
	videoFilter := buildVideoFilter(req.OutputSettings)
	var captions *captionAsset
	if req.OutputSettings != nil && req.OutputSettings.Captions {
		captions, err = resolveCaptionAsset(req.OutputSettings, video.ID, userAgent, cookie, proxyCfg)
		if err != nil {
			h.Logger.Error("Caption source failed", map[string]string{"Error": err.Error()})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}

	results := make([]ClipSegmentResponse, 0, len(req.Segments))
	successCount := 0
	failureCount := 0

	for idx, segment := range req.Segments {
		segmentIndex := idx + 1
		result := ClipSegmentResponse{
			Index: segmentIndex,
			Start: segment.Start,
			End:   segment.End,
		}

		if segment.Start < 0 {
			result.Error = "Start time must be >= 0"
			failureCount++
			results = append(results, result)
			continue
		}

		duration := segment.End - segment.Start
		if duration <= 0 {
			result.Error = "End time must be greater than start time"
			failureCount++
			results = append(results, result)
			continue
		}

		thumbAt := resolveThumbnailAt(duration, segment.ThumbnailAt)
		result.ThumbnailAt = thumbAt

		baseName := buildClipBaseName(clipID, batchTimestamp, segmentIndex)
		outputPath := filepath.Join(outputDir, baseName)

		subtitlePath, cleanupSubtitle, err := buildSegmentSubtitleFileWithSpeech(captions, req.OutputSettings, segment.Start, segment.End, streamURL, userAgent, cookie, proxyCfg)
		if err != nil {
			h.Logger.Error("Subtitle build failed", map[string]string{
				"Error":   err.Error(),
				"Segment": fmt.Sprintf("%d", segmentIndex),
			})
			result.Error = "Failed to build subtitles"
			failureCount++
			results = append(results, result)
			continue
		}
		combinedFilter := joinFilters(videoFilter, subtitleFilterForPath(subtitlePath, req.OutputSettings))

		ffmpegArgs := []string{
			"-user_agent", userAgent,
			"-headers", ffmpegHeaders,
		}
		if proxyCfg.proxyURLStr != "" {
			ffmpegArgs = append(ffmpegArgs, "-http_proxy", proxyCfg.proxyURLStr)
		}
		ffmpegArgs = append(ffmpegArgs,
			"-ss", fmt.Sprintf("%f", segment.Start),
			"-i", streamURL,
			"-t", fmt.Sprintf("%f", duration),
		)
		if combinedFilter != "" {
			ffmpegArgs = append(ffmpegArgs, "-vf", combinedFilter)
		}
		ffmpegArgs = append(ffmpegArgs,
			"-c:v", "libx264", "-preset", "ultrafast",
			"-c:a", "aac",
			"-movflags", "frag_keyframe+empty_moov",
			"-f", "mp4",
			outputPath,
		)

		h.Logger.Info("FFmpeg Args (Batch)", map[string]string{
			"Args":    sanitizeFFmpegArgs(ffmpegArgs),
			"Segment": fmt.Sprintf("%d", segmentIndex),
		})

		cmdFfmpeg := exec.Command("ffmpeg", ffmpegArgs...)
		cmdFfmpeg.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)

		var ffmpegErr bytes.Buffer
		cmdFfmpeg.Stderr = &ffmpegErr

		startRender := time.Now()
		if err := cmdFfmpeg.Run(); err != nil {
			if cleanupSubtitle != nil {
				cleanupSubtitle()
			}
			_ = os.Remove(outputPath)
			h.Logger.Error("Batch Clip Failed", map[string]string{
				"Error":   err.Error(),
				"Stderr":  ffmpegErr.String(),
				"Output":  outputPath,
				"Segment": fmt.Sprintf("%d", segmentIndex),
			})
			result.Error = "Failed to generate clip"
			failureCount++
			results = append(results, result)
			continue
		}
		if cleanupSubtitle != nil {
			cleanupSubtitle()
		}

		renderDuration := time.Since(startRender)
		result.OutputPath = outputPath
		result.RenderDuration = renderDuration.String()

		thumbPath, thumbErr := generateThumbnailAt(outputPath, thumbAt)
		if thumbErr != nil {
			h.Logger.Warning("Batch Thumbnail Failed", map[string]string{
				"Error":   thumbErr.Error(),
				"Output":  outputPath,
				"Segment": fmt.Sprintf("%d", segmentIndex),
			})
			result.ThumbnailError = thumbErr.Error()
		} else {
			h.Logger.Info("Batch Thumbnail Success", map[string]string{
				"OutputPath":    outputPath,
				"ThumbnailPath": thumbPath,
				"Segment":       fmt.Sprintf("%d", segmentIndex),
			})
			result.ThumbnailPath = thumbPath
		}

		successCount++
		results = append(results, result)
	}

	response := BatchClipResponse{
		Results: results,
		Total:   len(req.Segments),
		Success: successCount,
		Failed:  failureCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Batch Clip Response Failed", map[string]string{"Error": err.Error()})
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
