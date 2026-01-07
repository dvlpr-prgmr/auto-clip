package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type SocialUploadRequest struct {
	Platform    string   `json:"platform"`
	VideoPath   string   `json:"video_path"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Hashtags    []string `json:"hashtags,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
}

type SocialUploadResponse struct {
	Platform  string `json:"platform"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func (h *Handler) SocialUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SocialUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	platform := normalizePlatform(req.Platform)
	if platform == "" {
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.VideoPath) == "" {
		http.Error(w, "Missing video_path field", http.StatusBadRequest)
		return
	}

	absPath, err := resolveUploadPath(req.VideoPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.VideoPath = absPath
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Hashtags = normalizeTags(req.Hashtags)

	if isDryRun() {
		writeUploadResponse(w, platform, "ok", "Dry run enabled", "")
		return
	}

	var uploadErr error
	switch platform {
	case "youtube":
		uploadErr = h.uploadYouTubeShorts(req)
	case "tiktok":
		uploadErr = h.uploadTikTok(req)
	case "instagram":
		uploadErr = h.uploadInstagramReels(req)
	default:
		uploadErr = errors.New("Unsupported platform")
	}

	if uploadErr != nil {
		h.Logger.Error("Social upload failed", map[string]string{
			"Platform": platform,
			"Error":    uploadErr.Error(),
		})
		http.Error(w, uploadErr.Error(), http.StatusBadRequest)
		return
	}

	writeUploadResponse(w, platform, "ok", "Upload queued", "")
}

func normalizePlatform(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "youtube", "youtube_shorts", "shorts", "yt":
		return "youtube"
	case "tiktok", "tik_tok":
		return "tiktok"
	case "instagram", "ig", "reels":
		return "instagram"
	default:
		return ""
	}
}

func resolveUploadPath(path string) (string, error) {
	normalized := filepath.Clean(strings.TrimSpace(path))
	if normalized == "" || normalized == "." {
		return "", errors.New("Invalid video_path")
	}
	if filepath.IsAbs(normalized) {
		if _, err := os.Stat(normalized); err != nil {
			return "", fmt.Errorf("Video file not found: %s", normalized)
		}
		return normalized, nil
	}

	if _, err := os.Stat(normalized); err == nil {
		abs, err := filepath.Abs(normalized)
		if err != nil {
			return normalized, nil
		}
		return abs, nil
	}

	base := strings.TrimSpace(os.Getenv("CLIP_OUTPUT_DIR"))
	if base == "" {
		base = filepath.Join("output", "clips")
	}
	candidate := filepath.Join(base, normalized)
	if _, err := os.Stat(candidate); err == nil {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			return candidate, nil
		}
		return abs, nil
	}

	return "", fmt.Errorf("Video file not found: %s", normalized)
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "#") {
			value = "#" + value
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func isDryRun() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("SOCIAL_UPLOAD_DRY_RUN")))
	return value == "1" || value == "true" || value == "yes"
}

func writeUploadResponse(w http.ResponseWriter, platform string, status string, detail string, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SocialUploadResponse{
		Platform:  platform,
		Status:    status,
		Detail:    detail,
		RequestID: requestID,
	})
}

func (h *Handler) uploadYouTubeShorts(req SocialUploadRequest) error {
	if strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_ID")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_SECRET")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_REFRESH_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_CHANNEL_ID")) == "" {
		return errors.New("YouTube upload not configured; set YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET, YOUTUBE_REFRESH_TOKEN, YOUTUBE_CHANNEL_ID")
	}
	return errors.New("YouTube upload not implemented yet")
}

func (h *Handler) uploadTikTok(req SocialUploadRequest) error {
	if strings.TrimSpace(os.Getenv("TIKTOK_ACCESS_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("TIKTOK_OPEN_ID")) == "" {
		return errors.New("TikTok upload not configured; set TIKTOK_ACCESS_TOKEN and TIKTOK_OPEN_ID")
	}
	return errors.New("TikTok upload not implemented yet")
}

func (h *Handler) uploadInstagramReels(req SocialUploadRequest) error {
	if strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("INSTAGRAM_USER_ID")) == "" {
		return errors.New("Instagram upload not configured; set INSTAGRAM_ACCESS_TOKEN and INSTAGRAM_USER_ID")
	}
	return errors.New("Instagram upload not implemented yet")
}
