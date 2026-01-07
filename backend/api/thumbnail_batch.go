package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ThumbnailSegmentResponse struct {
	Index          int     `json:"index"`
	Start          float64 `json:"start"`
	End            float64 `json:"end"`
	ThumbnailAt    float64 `json:"thumbnail_at,omitempty"`
	ThumbnailPath  string  `json:"thumbnail_path,omitempty"`
	ThumbnailError string  `json:"thumbnail_error,omitempty"`
	Error          string  `json:"error,omitempty"`
}

type ThumbnailBatchResponse struct {
	Results []ThumbnailSegmentResponse `json:"results"`
	Total   int                       `json:"total"`
	Success int                       `json:"success"`
	Failed  int                       `json:"failed"`
}

func (h *Handler) ThumbnailBatch(w http.ResponseWriter, r *http.Request) {
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

	h.Logger.Info("Batch Thumbnail Request Received", map[string]string{
		"URL":       req.URL,
		"Segments":  fmt.Sprintf("%d", len(req.Segments)),
		"Requester": r.RemoteAddr,
	})

	video, streamURL, userAgent, cookie, proxyCfg, err := h.getVideoStream(req.URL)
	if err != nil {
		h.Logger.Error("Thumbnail stream lookup failed", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to get stream URL", http.StatusInternalServerError)
		return
	}

	outputDir := strings.TrimSpace(os.Getenv("THUMBNAIL_OUTPUT_DIR"))
	if outputDir == "" {
		outputDir = filepath.Join("output", "thumbnails")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		h.Logger.Error("Failed to create thumbnail directory", map[string]string{"Error": err.Error(), "Path": outputDir})
		http.Error(w, "Failed to create thumbnail directory", http.StatusInternalServerError)
		return
	}

	clipID := sanitizeVideoID(video.ID)
	batchTimestamp := time.Now().UnixNano()

	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}
	videoFilter := buildVideoFilter(req.OutputSettings)

	results := make([]ThumbnailSegmentResponse, 0, len(req.Segments))
	successCount := 0
	failureCount := 0

	for idx, segment := range req.Segments {
		segmentIndex := idx + 1
		result := ThumbnailSegmentResponse{
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
		sourceTimestamp := segment.Start + thumbAt
		if sourceTimestamp < 0 {
			sourceTimestamp = 0
		}

		thumbName := buildThumbnailBaseName(clipID, batchTimestamp, segmentIndex)
		thumbPath := filepath.Join(outputDir, thumbName)

		ffmpegArgs := []string{
			"-hide_banner",
			"-loglevel", "error",
			"-y",
			"-user_agent", userAgent,
			"-headers", ffmpegHeaders,
		}
		if proxyCfg.proxyURLStr != "" {
			ffmpegArgs = append(ffmpegArgs, "-http_proxy", proxyCfg.proxyURLStr)
		}
		ffmpegArgs = append(ffmpegArgs,
			"-ss", fmt.Sprintf("%.3f", sourceTimestamp),
			"-i", streamURL,
		)
		if videoFilter != "" {
			ffmpegArgs = append(ffmpegArgs, "-vf", videoFilter)
		}
		ffmpegArgs = append(ffmpegArgs,
			"-frames:v", "1",
			"-q:v", "2",
			thumbPath,
		)

		h.Logger.Info("FFmpeg Args (Thumb Batch)", map[string]string{
			"Args":    sanitizeFFmpegArgs(ffmpegArgs),
			"Segment": fmt.Sprintf("%d", segmentIndex),
		})

		cmdFfmpeg := exec.Command("ffmpeg", ffmpegArgs...)
		cmdFfmpeg.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)

		var ffmpegErr bytes.Buffer
		cmdFfmpeg.Stderr = &ffmpegErr

		if err := cmdFfmpeg.Run(); err != nil {
			_ = os.Remove(thumbPath)
			h.Logger.Error("Batch Thumbnail Failed", map[string]string{
				"Error":   err.Error(),
				"Stderr":  ffmpegErr.String(),
				"Output":  thumbPath,
				"Segment": fmt.Sprintf("%d", segmentIndex),
			})
			result.ThumbnailError = "Failed to generate thumbnail"
			failureCount++
			results = append(results, result)
			continue
		}

		result.ThumbnailPath = thumbPath
		successCount++
		results = append(results, result)
	}

	response := ThumbnailBatchResponse{
		Results: results,
		Total:   len(req.Segments),
		Success: successCount,
		Failed:  failureCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
