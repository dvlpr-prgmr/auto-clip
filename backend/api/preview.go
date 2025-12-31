package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

type PreviewResponse struct {
	Title          string `json:"title"`
	DurationSec    int64  `json:"duration_seconds"`
	DurationLabel  string `json:"duration_label"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	FPS            int    `json:"fps,omitempty"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	SizeLabel      string `json:"size_label,omitempty"`
	ThumbnailURL   string `json:"thumbnail_url,omitempty"`
	PreviewQuality string `json:"preview_quality,omitempty"`
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, "Missing url query parameter", http.StatusBadRequest)
		return
	}

	userAgent := resolveUserAgent()
	cookie := resolveYouTubeCookie(h.Logger)

	defaultHeaders := http.Header{}
	defaultHeaders.Set("User-Agent", userAgent)
	if cookie != "" {
		defaultHeaders.Set("Cookie", cookie)
	}

	proxyCfg := resolveProxyConfig(h.Logger)

	client := youtube.Client{
		HTTPClient: &http.Client{
			Transport: headerRoundTripper{
				base:    proxyCfg.baseTransport,
				headers: defaultHeaders,
			},
		},
	}

	video, err := client.GetVideo(rawURL)
	if err != nil {
		h.Logger.Error("Preview Failed", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to fetch video metadata", http.StatusBadRequest)
		return
	}

	response := PreviewResponse{
		Title:         video.Title,
		DurationSec:   int64(video.Duration.Seconds()),
		DurationLabel: formatDuration(video.Duration),
		ThumbnailURL:  pickThumbnail(video.Thumbnails),
	}

	format, ok := pickPreviewFormat(video.Formats)
	if ok {
		response.Width = format.Width
		response.Height = format.Height
		response.FPS = format.FPS
		if format.ContentLength > 0 {
			response.SizeBytes = format.ContentLength
			response.SizeLabel = formatBytes(format.ContentLength)
		} else if format.Bitrate > 0 && response.DurationSec > 0 {
			sizeEstimate := int64(format.Bitrate) / 8 * response.DurationSec
			if sizeEstimate > 0 {
				response.SizeBytes = sizeEstimate
				response.SizeLabel = formatBytes(sizeEstimate)
			}
		}
		response.PreviewQuality = format.QualityLabel
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func pickPreviewFormat(formats youtube.FormatList) (youtube.Format, bool) {
	if len(formats) == 0 {
		return youtube.Format{}, false
	}

	candidates := formats.
		Type("video/mp4").
		WithAudioChannels()
	candidates.Sort()
	for _, format := range candidates {
		if format.Width > 0 && format.Height > 0 {
			return format, true
		}
	}

	formats.Sort()
	for _, format := range formats {
		if format.Width > 0 && format.Height > 0 {
			return format, true
		}
	}

	return youtube.Format{}, false
}

func pickThumbnail(thumbnails youtube.Thumbnails) string {
	if len(thumbnails) == 0 {
		return ""
	}
	return thumbnails[len(thumbnails)-1].URL
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}

	total := int64(d.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTPE"[exp])
}
