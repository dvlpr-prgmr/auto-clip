package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"auto-clip-backend/api/autoscene"
	"github.com/kkdai/youtube/v2"
)

const (
	defaultFrameCount  = 6
	maxFrameCount      = 8
	minFrameCount      = 3
	frameScaleWidth    = 512
	audioSampleSeconds = 18
	audioSampleRate    = 16000
)

const (
	maxTranscriptChars = 12000
	maxTranscriptLines = 1200
)

func detectSceneCutsAI(video *youtube.Video, streamURL string, start float64, end float64, language string, userAgent string, cookie string, proxyCfg proxyConfig, minDuration float64, maxDuration float64, maxSegments int) ([]float64, []SceneSegment, error) {
	if end <= start {
		return nil, nil, fmt.Errorf("invalid time range")
	}
	if video == nil {
		return nil, nil, fmt.Errorf("video is nil")
	}
	if strings.TrimSpace(streamURL) == "" {
		return nil, nil, fmt.Errorf("stream url is empty")
	}

	var transcript string
	var captionErr error
	cues, err := fetchYouTubeCaptions(video, language, userAgent, cookie, proxyCfg)
	if err == nil {
		transcript = buildTranscriptForRange(cues, start, end)
	} else {
		captionErr = err
	}

	ctx := context.Background()
	apiKey := strings.TrimSpace(getEnv("GEMINI_API_KEY"))
	if apiKey == "" {
		return nil, nil, fmt.Errorf("GEMINI_API_KEY not configured")
	}
	client, err := autoscene.NewClient(ctx, apiKey)
	if err != nil {
		return nil, nil, err
	}

	options := autoscene.AnalyzeOptions{
		MaxClips:        maxSegments,
		MinDurationSecs: minDuration,
		MaxDurationSecs: maxDuration,
		LanguageHint:    language,
	}

	if strings.TrimSpace(transcript) != "" {
		result, err := client.AnalyzeWithOptions(ctx, transcript, options)
		if err != nil {
			return nil, nil, err
		}
		segments := clipsToSceneSegments(result.Clips)
		return nil, segments, nil
	}

	frames, frameErr := sampleFrames(streamURL, start, end, userAgent, cookie, proxyCfg)
	if frameErr != nil {
		if captionErr != nil {
			return nil, nil, fmt.Errorf("captions unavailable (%v); multimodal failed: %w", captionErr, frameErr)
		}
		return nil, nil, frameErr
	}

	audioSamples, audioErr := sampleAudioSnippets(streamURL, start, end, userAgent, cookie, proxyCfg)
	if audioErr != nil {
		audioSamples = nil
	}

	result, err := client.AnalyzeMultimodalWithOptions(ctx, frames, audioSamples, options)
	if err != nil {
		if captionErr != nil {
			return nil, nil, fmt.Errorf("captions unavailable (%v); multimodal failed: %w", captionErr, err)
		}
		return nil, nil, err
	}
	segments := clipsToSceneSegments(result.Clips)
	return nil, segments, nil
}

func buildTranscriptForRange(cues []subtitleCue, start float64, end float64) string {
	if end <= start || len(cues) == 0 {
		return ""
	}
	startTime := time.Duration(start * float64(time.Second))
	endTime := time.Duration(end * float64(time.Second))
	if endTime <= startTime {
		return ""
	}

	var builder strings.Builder
	lines := 0
	for _, cue := range cues {
		if cue.End <= startTime || cue.Start >= endTime {
			continue
		}
		clipStart := maxDuration(cue.Start, startTime)
		clipEnd := minDurationTime(cue.End, endTime)
		if clipEnd <= clipStart {
			continue
		}
		text := sanitizeCaptionText(cue.Text)
		if text == "" {
			continue
		}
		relStart := clipStart - startTime
		relEnd := clipEnd - startTime
		line := fmt.Sprintf("[%.2fs - %.2fs] %s\n", relStart.Seconds(), relEnd.Seconds(), text)
		if builder.Len()+len(line) > maxTranscriptChars {
			break
		}
		builder.WriteString(line)
		lines++
		if lines >= maxTranscriptLines {
			break
		}
	}
	return strings.TrimSpace(builder.String())
}

func clipsToSceneSegments(clips []autoscene.Clip) []SceneSegment {
	if len(clips) == 0 {
		return nil
	}
	segments := make([]SceneSegment, 0, len(clips))
	for _, clip := range clips {
		startSec, err := parseTimecodeSeconds(clip.StartTime)
		if err != nil {
			continue
		}
		endSec, err := parseTimecodeSeconds(clip.EndTime)
		if err != nil {
			continue
		}
		if endSec <= startSec {
			continue
		}
		segments = append(segments, SceneSegment{
			Start:    startSec,
			End:      endSec,
			Title:    strings.TrimSpace(clip.Title),
			Hashtags: normalizeHashtags(clip.Hashtags),
		})
	}
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].Start < segments[j].Start
	})
	return segments
}

func parseTimecodeSeconds(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("empty timecode")
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) > 3 {
		return 0, fmt.Errorf("invalid timecode: %s", value)
	}
	seconds := 0.0
	multiplier := 1.0
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			return 0, fmt.Errorf("invalid timecode: %s", value)
		}
		num, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timecode: %s", value)
		}
		seconds += num * multiplier
		multiplier *= 60
	}
	return seconds, nil
}

func normalizeHashtags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	cleaned := make([]string, 0, len(tags))
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
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func sampleFrames(streamURL string, start float64, end float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]autoscene.FrameSample, error) {
	duration := end - start
	if duration <= 0 {
		return nil, fmt.Errorf("invalid range duration")
	}

	frameCount := defaultFrameCount
	switch {
	case duration < 30:
		frameCount = minFrameCount
	case duration > 300:
		frameCount = maxFrameCount
	}

	interval := duration / float64(frameCount+1)
	if interval <= 0 {
		interval = duration / float64(frameCount)
	}

	frames := make([]autoscene.FrameSample, 0, frameCount)
	var lastErr error
	for i := 1; i <= frameCount; i++ {
		ts := start + interval*float64(i)
		if ts <= start || ts >= end {
			continue
		}
		data, err := captureFrameJPEG(streamURL, ts, userAgent, cookie, proxyCfg)
		if err != nil {
			lastErr = err
			continue
		}
		frames = append(frames, autoscene.FrameSample{
			TimestampSec: ts - start,
			JPEG:         data,
		})
	}

	if len(frames) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no frames captured")
	}
	return frames, nil
}

func captureFrameJPEG(streamURL string, timestamp float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]byte, error) {
	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-user_agent", userAgent,
		"-headers", ffmpegHeaders,
	}
	if proxyCfg.proxyURLStr != "" {
		args = append(args, "-http_proxy", proxyCfg.proxyURLStr)
	}
	args = append(args,
		"-ss", fmt.Sprintf("%f", timestamp),
		"-i", streamURL,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale=%d:-1", frameScaleWidth),
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"pipe:1",
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg frame failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	data := stdout.Bytes()
	if len(data) == 0 {
		return nil, fmt.Errorf("ffmpeg frame output empty")
	}
	return data, nil
}

func sampleAudioSnippets(streamURL string, start float64, end float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]autoscene.AudioSample, error) {
	duration := end - start
	if duration <= 0 {
		return nil, fmt.Errorf("invalid range duration")
	}

	snippetDuration := float64(audioSampleSeconds)
	if duration < snippetDuration {
		snippetDuration = duration
	}
	if snippetDuration <= 2 {
		return nil, fmt.Errorf("range too short for audio sample")
	}

	mid := start + duration/2
	snippetStart := mid - snippetDuration/2
	if snippetStart < start {
		snippetStart = start
	}
	if snippetStart+snippetDuration > end {
		snippetStart = end - snippetDuration
	}
	if snippetStart < start {
		snippetStart = start
	}

	data, err := captureAudioWAV(streamURL, snippetStart, snippetDuration, userAgent, cookie, proxyCfg)
	if err != nil {
		return nil, err
	}

	return []autoscene.AudioSample{
		{
			StartSec:    snippetStart - start,
			DurationSec: snippetDuration,
			WAV:         data,
		},
	}, nil
}

func captureAudioWAV(streamURL string, start float64, duration float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]byte, error) {
	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
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
		"-vn",
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", audioSampleRate),
		"-f", "wav",
		"pipe:1",
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg audio failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	data := stdout.Bytes()
	if len(data) == 0 {
		return nil, fmt.Errorf("ffmpeg audio output empty")
	}
	return data, nil
}

func sanitizeCaptionText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func maxDuration(a time.Duration, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func minDurationTime(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func getEnv(key string) string {
	return strings.TrimSpace(strings.Trim(os.Getenv(key), `"'`))
}
