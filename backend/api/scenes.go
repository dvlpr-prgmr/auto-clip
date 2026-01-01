package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type SceneDetectRequest struct {
	URL         string  `json:"url"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Threshold   float64 `json:"threshold,omitempty"`
	MinDuration float64 `json:"min_duration,omitempty"`
	MaxDuration float64 `json:"max_duration,omitempty"`
	MaxSegments int     `json:"max_segments,omitempty"`
}

type SceneSegment struct {
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	ThumbnailAt float64 `json:"thumbnail_at,omitempty"`
}

type SceneDetectResponse struct {
	Cuts     []float64     `json:"cuts"`
	Segments []SceneSegment `json:"segments"`
}

var showInfoRegex = regexp.MustCompile(`pts_time:([0-9]+(?:\.[0-9]+)?)`)

func (h *Handler) SceneDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SceneDetectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "Missing url field", http.StatusBadRequest)
		return
	}
	if req.End <= req.Start {
		http.Error(w, "End time must be greater than start time", http.StatusBadRequest)
		return
	}

	threshold := req.Threshold
	if threshold <= 0 {
		threshold = 0.4
	}
	if threshold < 0.1 {
		threshold = 0.1
	}
	if threshold > 0.9 {
		threshold = 0.9
	}

	video, streamURL, userAgent, cookie, proxyCfg, err := h.getVideoStream(req.URL)
	if err != nil {
		h.Logger.Error("Scene detect stream failed", map[string]string{"Error": err.Error()})
		http.Error(w, "Failed to get stream URL", http.StatusInternalServerError)
		return
	}

	duration := req.End - req.Start
	cuts, err := detectSceneCuts(streamURL, req.Start, duration, threshold, userAgent, cookie, proxyCfg)
	if err != nil {
		h.Logger.Error("Scene detect failed", map[string]string{"Error": err.Error()})
		http.Error(w, "Scene detection failed", http.StatusInternalServerError)
		return
	}

	segments := buildSceneSegments(req.Start, req.End, cuts, req.MinDuration, req.MaxDuration, req.MaxSegments)
	response := SceneDetectResponse{
		Cuts:     cuts,
		Segments: segments,
	}

	h.Logger.Info("Scene Detect Success", map[string]string{
		"URL":      req.URL,
		"VideoID":  video.ID,
		"Segments": fmt.Sprintf("%d", len(segments)),
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func detectSceneCuts(streamURL string, start float64, duration float64, threshold float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]float64, error) {
	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}
	filter := fmt.Sprintf("select='gt(scene,%0.2f)',showinfo", threshold)
	args := []string{
		"-hide_banner",
		"-loglevel", "info",
		"-user_agent", userAgent,
		"-headers", ffmpegHeaders,
	}
	if proxyCfg.proxyURLStr != "" {
		args = append(args, "-http_proxy", proxyCfg.proxyURLStr)
	}
	args = append(args,
		"-ss", fmt.Sprintf("%f", start),
		"-i", streamURL,
		"-t", fmt.Sprintf("%f", duration),
		"-vf", filter,
		"-f", "null",
		"-",
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg scene detect: %w", err)
	}

	output := stderr.String()
	matches := showInfoRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	relativeCuts := make([]float64, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		relativeCuts = append(relativeCuts, value)
	}

	cuts := normalizeSceneCuts(relativeCuts, start, duration)
	return cuts, nil
}

func normalizeSceneCuts(cuts []float64, start float64, duration float64) []float64 {
	if len(cuts) == 0 {
		return nil
	}
	sort.Float64s(cuts)
	unique := make([]float64, 0, len(cuts))
	last := -1.0
	for _, cut := range cuts {
		if cut < 0 {
			continue
		}
		normalized := cut
		if cut <= duration+1 {
			normalized = start + cut
		}
		if last >= 0 && math.Abs(normalized-last) < 0.2 {
			continue
		}
		unique = append(unique, normalized)
		last = normalized
	}
	return unique
}

func buildSceneSegments(start float64, end float64, cuts []float64, minDuration float64, maxDuration float64, maxSegments int) []SceneSegment {
	if end <= start {
		return nil
	}
	if minDuration < 0 {
		minDuration = 0
	}
	if maxDuration < 0 {
		maxDuration = 0
	}
	if maxSegments < 0 {
		maxSegments = 0
	}

	bounds := []float64{start}
	for _, cut := range cuts {
		if cut <= start || cut >= end {
			continue
		}
		if minDuration > 0 && cut-bounds[len(bounds)-1] < minDuration {
			continue
		}
		bounds = append(bounds, cut)
	}
	bounds = append(bounds, end)

	if len(bounds) < 2 {
		return nil
	}

	segments := make([]SceneSegment, 0, len(bounds)-1)
	for i := 0; i < len(bounds)-1; i++ {
		segStart := bounds[i]
		segEnd := bounds[i+1]
		if segEnd <= segStart {
			continue
		}
		segments = append(segments, SceneSegment{
			Start: segStart,
			End:   segEnd,
		})
	}

	segments = splitLongSegments(segments, maxDuration)
	segments = mergeShortSegments(segments, minDuration)
	segments = applyThumbnailTimes(segments)

	if maxSegments > 0 && len(segments) > maxSegments {
		return segments[:maxSegments]
	}
	if len(segments) > 500 {
		return segments[:500]
	}
	return segments
}

func splitLongSegments(segments []SceneSegment, maxDuration float64) []SceneSegment {
	if maxDuration <= 0 {
		return segments
	}
	split := make([]SceneSegment, 0, len(segments))
	for _, seg := range segments {
		duration := seg.End - seg.Start
		if duration <= maxDuration {
			split = append(split, seg)
			continue
		}
		count := int(math.Ceil(duration / maxDuration))
		for i := 0; i < count; i++ {
			start := seg.Start + float64(i)*maxDuration
			end := math.Min(seg.Start+float64(i+1)*maxDuration, seg.End)
			if end <= start {
				continue
			}
			split = append(split, SceneSegment{
				Start: start,
				End:   end,
			})
		}
	}
	return split
}

func mergeShortSegments(segments []SceneSegment, minDuration float64) []SceneSegment {
	if minDuration <= 0 || len(segments) < 2 {
		return segments
	}
	if segments[0].End-segments[0].Start < minDuration && len(segments) > 1 {
		segments[1].Start = segments[0].Start
		segments = segments[1:]
	}
	merged := make([]SceneSegment, 0, len(segments))
	for _, seg := range segments {
		if len(merged) == 0 {
			merged = append(merged, seg)
			continue
		}
		duration := seg.End - seg.Start
		if duration < minDuration {
			merged[len(merged)-1].End = seg.End
			continue
		}
		merged = append(merged, seg)
	}
	return merged
}

func applyThumbnailTimes(segments []SceneSegment) []SceneSegment {
	for i := range segments {
		duration := segments[i].End - segments[i].Start
		segments[i].ThumbnailAt = math.Min(defaultThumbnailTimestamp, math.Max(duration-0.05, 0))
	}
	return segments
}
