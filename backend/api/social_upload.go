package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type youtubeTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

type youtubeVideoSnippet struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CategoryID  string   `json:"categoryId,omitempty"`
}

type youtubeVideoStatus struct {
	PrivacyStatus           string `json:"privacyStatus"`
	SelfDeclaredMadeForKids bool   `json:"selfDeclaredMadeForKids"`
}

type youtubeVideoResource struct {
	Snippet youtubeVideoSnippet `json:"snippet"`
	Status  youtubeVideoStatus  `json:"status"`
}

type youtubeUploadResponse struct {
	ID string `json:"id"`
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
	requestID := ""
	switch platform {
	case "youtube":
		requestID, uploadErr = h.uploadYouTubeShorts(req)
	case "tiktok":
		requestID, uploadErr = h.uploadTikTok(req)
	case "instagram":
		requestID, uploadErr = h.uploadInstagramReels(req)
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

	writeUploadResponse(w, platform, "ok", "Upload complete", requestID)
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

func (h *Handler) uploadYouTubeShorts(req SocialUploadRequest) (string, error) {
	if strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_ID")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_SECRET")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_REFRESH_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("YOUTUBE_CHANNEL_ID")) == "" {
		return "", errors.New("YouTube upload not configured; set YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET, YOUTUBE_REFRESH_TOKEN, YOUTUBE_CHANNEL_ID")
	}
	videoID, err := h.uploadToYouTube(req)
	if err != nil {
		return "", err
	}
	return videoID, nil
}

func (h *Handler) uploadTikTok(req SocialUploadRequest) (string, error) {
	if strings.TrimSpace(os.Getenv("TIKTOK_ACCESS_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("TIKTOK_OPEN_ID")) == "" {
		return "", errors.New("TikTok upload not configured; set TIKTOK_ACCESS_TOKEN and TIKTOK_OPEN_ID")
	}
	return "", errors.New("TikTok upload not implemented yet")
}

func (h *Handler) uploadInstagramReels(req SocialUploadRequest) (string, error) {
	if strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN")) == "" ||
		strings.TrimSpace(os.Getenv("INSTAGRAM_USER_ID")) == "" {
		return "", errors.New("Instagram upload not configured; set INSTAGRAM_ACCESS_TOKEN and INSTAGRAM_USER_ID")
	}
	return "", errors.New("Instagram upload not implemented yet")
}

func (h *Handler) uploadToYouTube(req SocialUploadRequest) (string, error) {
	clientID := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_SECRET"))
	refreshToken := strings.TrimSpace(os.Getenv("YOUTUBE_REFRESH_TOKEN"))
	categoryID := strings.TrimSpace(os.Getenv("YOUTUBE_CATEGORY_ID"))

	accessToken, err := refreshYouTubeAccessToken(clientID, clientSecret, refreshToken)
	if err != nil {
		return "", err
	}

	fileInfo, err := os.Stat(req.VideoPath)
	if err != nil {
		return "", fmt.Errorf("video file not found: %w", err)
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(req.VideoPath), filepath.Ext(req.VideoPath))
	}
	if title == "" {
		title = "Auto Clip"
	}

	description := buildYouTubeDescription(req.Description, req.Hashtags)
	tags := hashtagsToTags(req.Hashtags)

	privacy := normalizeVisibility(req.Visibility)
	resource := youtubeVideoResource{
		Snippet: youtubeVideoSnippet{
			Title:       title,
			Description: description,
			Tags:        tags,
			CategoryID:  categoryID,
		},
		Status: youtubeVideoStatus{
			PrivacyStatus:           privacy,
			SelfDeclaredMadeForKids: false,
		},
	}

	uploadURL, err := createYouTubeUploadSession(accessToken, fileInfo.Size(), resource)
	if err != nil {
		return "", err
	}

	videoID, err := uploadYouTubeFile(accessToken, uploadURL, req.VideoPath, fileInfo.Size())
	if err != nil {
		return "", err
	}

	return videoID, nil
}

func refreshYouTubeAccessToken(clientID string, clientSecret string, refreshToken string) (string, error) {
	form := make(url.Values)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("YouTube token error: %s", strings.TrimSpace(string(body)))
	}

	var payload youtubeTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("YouTube token parse failed: %w", err)
	}
	if payload.AccessToken == "" {
		return "", errors.New("YouTube access_token missing")
	}
	return payload.AccessToken, nil
}

func createYouTubeUploadSession(accessToken string, contentLength int64, resource youtubeVideoResource) (string, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", "video/mp4")
	req.Header.Set("X-Upload-Content-Length", fmt.Sprintf("%d", contentLength))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("YouTube upload session error: %s", strings.TrimSpace(string(body)))
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", errors.New("YouTube upload session missing Location header")
	}
	return location, nil
}

func uploadYouTubeFile(accessToken string, uploadURL string, path string, contentLength int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodPut, uploadURL, file)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "video/mp4")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", contentLength))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("YouTube upload failed: %s", strings.TrimSpace(string(body)))
	}

	var payload youtubeUploadResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("YouTube response parse failed: %w", err)
	}
	if payload.ID == "" {
		return "", errors.New("YouTube upload returned empty video id")
	}
	return payload.ID, nil
}

func normalizeVisibility(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "private":
		return "private"
	case "unlisted":
		return "unlisted"
	default:
		return "public"
	}
}

func buildYouTubeDescription(description string, hashtags []string) string {
	base := strings.TrimSpace(description)
	tags := normalizeTags(hashtags)
	if len(tags) == 0 {
		return base
	}
	tagLine := strings.Join(tags, " ")
	if base == "" {
		return tagLine
	}
	return base + "\n\n" + tagLine
}

func hashtagsToTags(hashtags []string) []string {
	if len(hashtags) == 0 {
		return nil
	}
	tags := make([]string, 0, len(hashtags))
	for _, tag := range hashtags {
		value := strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if value == "" {
			continue
		}
		tags = append(tags, value)
		if len(tags) >= 15 {
			break
		}
	}
	return tags
}
