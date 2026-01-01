package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type subtitleCue struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

type captionAsset struct {
	Cues []subtitleCue
}

func resolveCaptionAsset(settings *OutputSettings, videoID string, userAgent string, cookie string, proxyCfg proxyConfig) (*captionAsset, error) {
	if settings == nil || !settings.Captions {
		return nil, nil
	}

	source := strings.ToLower(strings.TrimSpace(settings.CaptionSource))
	switch source {
	case "", "youtube":
		cues, err := fetchYouTubeCaptions(videoID, settings.CaptionLanguage, userAgent, cookie, proxyCfg)
		if err != nil {
			return nil, err
		}
		return &captionAsset{Cues: cues}, nil
	case "upload":
		text := strings.TrimSpace(settings.CaptionText)
		if text == "" {
			return nil, errors.New("uploaded subtitle text is empty")
		}
		format := strings.ToLower(strings.TrimSpace(settings.CaptionFormat))
		if format == "" {
			format = "srt"
		}
		if format == "vtt" {
			text = convertVTTToSRT(text)
		}
		cues, err := parseSRT(text)
		if err != nil {
			return nil, err
		}
		return &captionAsset{Cues: cues}, nil
	case "speech":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown caption source: %s", source)
	}
}

func validateSpeechConfig(settings *OutputSettings) error {
	if settings == nil || !settings.Captions {
		return nil
	}
	source := strings.ToLower(strings.TrimSpace(settings.CaptionSource))
	if source != "speech" {
		return nil
	}
	bin := strings.TrimSpace(os.Getenv("WHISPER_CPP_PATH"))
	if bin == "" {
		return errors.New("WHISPER_CPP_PATH is not set")
	}
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("WHISPER_CPP_PATH is invalid: %w", err)
	}
	model := strings.TrimSpace(os.Getenv("WHISPER_MODEL_PATH"))
	if model == "" {
		return errors.New("WHISPER_MODEL_PATH is not set")
	}
	if _, err := os.Stat(model); err != nil {
		return fmt.Errorf("WHISPER_MODEL_PATH is invalid: %w", err)
	}
	return nil
}

func buildSegmentSubtitleFileWithSpeech(asset *captionAsset, settings *OutputSettings, segmentStart float64, segmentEnd float64, streamURL string, userAgent string, cookie string, proxyCfg proxyConfig) (string, func(), error) {
	if settings == nil || !settings.Captions {
		return "", nil, nil
	}
	source := strings.ToLower(strings.TrimSpace(settings.CaptionSource))
	if source != "speech" {
	return buildSegmentSubtitleFile(asset, settings, segmentStart, segmentEnd)
	}

	cues, err := transcribeSegmentSpeech(settings, streamURL, segmentStart, segmentEnd, userAgent, cookie, proxyCfg)
	if err != nil {
		return "", nil, err
	}
	if len(cues) == 0 {
		return "", nil, nil
	}
	duration := time.Duration((segmentEnd - segmentStart) * float64(time.Second))
	if duration <= 0 {
		return "", nil, nil
	}
	tmpAsset := &captionAsset{Cues: cues}
	return buildSegmentSubtitleFile(tmpAsset, settings, 0, duration.Seconds())
}

func transcribeSegmentSpeech(settings *OutputSettings, streamURL string, segmentStart float64, segmentEnd float64, userAgent string, cookie string, proxyCfg proxyConfig) ([]subtitleCue, error) {
	if segmentEnd <= segmentStart {
		return nil, errors.New("invalid segment duration")
	}
	bin := strings.TrimSpace(os.Getenv("WHISPER_CPP_PATH"))
	model := strings.TrimSpace(os.Getenv("WHISPER_MODEL_PATH"))

	tmpDir, err := os.MkdirTemp("", "whisper-segment-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	audioPath := filepath.Join(tmpDir, "segment.wav")
	ffmpegHeaders := fmt.Sprintf("User-Agent: %s\r\n", userAgent)
	if cookie != "" {
		ffmpegHeaders += fmt.Sprintf("Cookie: %s\r\n", cookie)
	}

	ffmpegArgs := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-user_agent", userAgent,
		"-headers", ffmpegHeaders,
	}
	if proxyCfg.proxyURLStr != "" {
		ffmpegArgs = append(ffmpegArgs, "-http_proxy", proxyCfg.proxyURLStr)
	}
	ffmpegArgs = append(ffmpegArgs,
		"-ss", fmt.Sprintf("%f", segmentStart),
		"-i", streamURL,
		"-t", fmt.Sprintf("%f", segmentEnd-segmentStart),
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-f", "wav",
		audioPath,
	)

	cmdFfmpeg := exec.Command("ffmpeg", ffmpegArgs...)
	cmdFfmpeg.Env = buildFFmpegEnv(proxyCfg.proxyURLStr, proxyCfg.proxyDisabled, proxyCfg.httpProxyEnv)
	if err := cmdFfmpeg.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg audio extract failed: %w", err)
	}

	outputPrefix := filepath.Join(tmpDir, "whisper")
	whisperArgs := []string{
		"-m", model,
		"-f", audioPath,
		"-of", outputPrefix,
		"-osrt",
	}
	lang := strings.TrimSpace(settings.CaptionLanguage)
	if lang != "" && !strings.EqualFold(lang, "auto") {
		whisperArgs = append(whisperArgs, "-l", lang)
	}

	cmdWhisper := exec.Command(bin, whisperArgs...)
	cmdWhisper.Dir = tmpDir
	if err := cmdWhisper.Run(); err != nil {
		return nil, fmt.Errorf("whisper.cpp failed: %w", err)
	}

	srtPath := outputPrefix + ".srt"
	content, err := os.ReadFile(srtPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read whisper srt: %w", err)
	}
	return parseSRT(string(content))
}

func fetchYouTubeCaptions(videoID string, language string, userAgent string, cookie string, proxyCfg proxyConfig) ([]subtitleCue, error) {
	lang := strings.TrimSpace(language)
	if lang == "" || strings.EqualFold(lang, "auto") {
		lang = "en"
	}

	client := &http.Client{
		Transport: headerRoundTripper{
			base: proxyCfg.baseTransport,
			headers: http.Header{
				"User-Agent": []string{userAgent},
			},
		},
		Timeout: 15 * time.Second,
	}
	if cookie != "" {
		client.Transport = headerRoundTripper{
			base: proxyCfg.baseTransport,
			headers: http.Header{
				"User-Agent": []string{userAgent},
				"Cookie":     []string{cookie},
			},
		}
	}

	srt, err := fetchTimedTextSRT(client, videoID, lang, false)
	if err == nil {
		return parseSRT(srt)
	}
	srtAuto, autoErr := fetchTimedTextSRT(client, videoID, lang, true)
	if autoErr != nil {
		return nil, fmt.Errorf("youtube captions unavailable: %v / %v", err, autoErr)
	}
	return parseSRT(srtAuto)
}

func fetchTimedTextSRT(client *http.Client, videoID string, lang string, auto bool) (string, error) {
	if videoID == "" {
		return "", errors.New("missing video id")
	}

	params := url.Values{}
	params.Set("v", videoID)
	params.Set("lang", lang)
	params.Set("fmt", "srt")
	if auto {
		params.Set("kind", "asr")
	}

	req, err := http.NewRequest(http.MethodGet, "https://www.youtube.com/api/timedtext?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("timedtext status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", errors.New("timedtext response is empty")
	}
	return text, nil
}

func convertVTTToSRT(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "WEBVTT") || strings.HasPrefix(trimmed, "NOTE") {
			continue
		}
		if strings.Contains(line, "-->") {
			line = strings.ReplaceAll(line, ".", ",")
		}
		out = append(out, line)
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}

func parseSRT(raw string) ([]subtitleCue, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	if raw == "" {
		return nil, errors.New("subtitle content is empty")
	}

	blocks := strings.Split(raw, "\n\n")
	cues := make([]subtitleCue, 0, len(blocks))
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 2 {
			continue
		}

		timeLine := lines[0]
		textLines := lines[1:]
		if !strings.Contains(timeLine, "-->") && len(lines) >= 3 {
			timeLine = lines[1]
			textLines = lines[2:]
		}
		parts := strings.Split(timeLine, "-->")
		if len(parts) != 2 {
			continue
		}
		start, err := parseSRTTime(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		end, err := parseSRTTime(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}

		text := strings.TrimSpace(strings.Join(textLines, "\n"))
		if text == "" {
			continue
		}
		cues = append(cues, subtitleCue{
			Start: start,
			End:   end,
			Text:  text,
		})
	}

	if len(cues) == 0 {
		return nil, errors.New("no valid subtitles found")
	}
	return cues, nil
}

func parseSRTTime(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", value)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	secParts := strings.SplitN(parts[2], ".", 2)
	seconds, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, err
	}
	millis := 0
	if len(secParts) == 2 {
		msStr := secParts[1]
		for len(msStr) < 3 {
			msStr += "0"
		}
		if len(msStr) > 3 {
			msStr = msStr[:3]
		}
		millis, _ = strconv.Atoi(msStr)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(millis)*time.Millisecond, nil
}

func buildSegmentSubtitleFile(asset *captionAsset, settings *OutputSettings, startSec float64, endSec float64) (string, func(), error) {
	if asset == nil || len(asset.Cues) == 0 {
		return "", nil, nil
	}
	start := time.Duration(startSec * float64(time.Second))
	end := time.Duration(endSec * float64(time.Second))
	if end <= start {
		return "", nil, nil
	}

	style := ""
	if settings != nil {
		style = settings.CaptionStyle
	}
	normalizedStyle := strings.ToLower(strings.TrimSpace(style))
	isKaraoke := normalizedStyle == "karaoke" || normalizedStyle == "word_highlight"
	fontSize := 56
	highlightColor := "#F2C14E"
	if settings != nil {
		if settings.FontSize > 0 {
			fontSize = settings.FontSize
		}
		if strings.TrimSpace(settings.HighlightColor) != "" {
			highlightColor = settings.HighlightColor
		}
	}

	var content string
	if isKaraoke {
		content = buildASSForSegment(asset.Cues, start, end, normalizedStyle, fontSize, highlightColor)
	} else {
		content = buildSRTForSegment(asset.Cues, start, end)
	}

	if strings.TrimSpace(content) == "" {
		return "", nil, nil
	}

	ext := ".srt"
	if isKaraoke {
		ext = ".ass"
	}
	file, err := os.CreateTemp("", "clip-subtitles-*"+ext)
	if err != nil {
		return "", nil, err
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		return "", nil, err
	}

	cleanup := func() {
		_ = os.Remove(file.Name())
	}
	return file.Name(), cleanup, nil
}

func buildSRTForSegment(cues []subtitleCue, segmentStart time.Duration, segmentEnd time.Duration) string {
	var b strings.Builder
	index := 1
	for _, cue := range cues {
		if cue.End <= segmentStart || cue.Start >= segmentEnd {
			continue
		}
		start := cue.Start
		if start < segmentStart {
			start = segmentStart
		}
		end := cue.End
		if end > segmentEnd {
			end = segmentEnd
		}
		if end <= start {
			continue
		}
		b.WriteString(strconv.Itoa(index))
		b.WriteString("\n")
		b.WriteString(formatSRTTime(start - segmentStart))
		b.WriteString(" --> ")
		b.WriteString(formatSRTTime(end - segmentStart))
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(cue.Text))
		b.WriteString("\n\n")
		index++
	}
	return strings.TrimSpace(b.String())
}

func buildASSForSegment(cues []subtitleCue, segmentStart time.Duration, segmentEnd time.Duration, style string, fontSize int, highlightHex string) string {
	var b strings.Builder
	b.WriteString("[Script Info]\n")
	b.WriteString("ScriptType: v4.00+\n")
	b.WriteString("PlayResX: 1080\n")
	b.WriteString("PlayResY: 1920\n")
	b.WriteString("WrapStyle: 2\n")
	b.WriteString("ScaledBorderAndShadow: yes\n\n")
	b.WriteString("[V4+ Styles]\n")
	b.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	if fontSize <= 0 {
		fontSize = 56
	}
	secondary := assColorFromHex(highlightHex)
	b.WriteString(fmt.Sprintf("Style: Default,Arial,%d,&H00FFFFFF,%s,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,3,0,2,80,80,80,1\n\n", fontSize, secondary))
	b.WriteString("[Events]\n")
	b.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	index := 0
	for _, cue := range cues {
		if cue.End <= segmentStart || cue.Start >= segmentEnd {
			continue
		}
		start := cue.Start
		if start < segmentStart {
			start = segmentStart
		}
		end := cue.End
		if end > segmentEnd {
			end = segmentEnd
		}
		if end <= start {
			continue
		}
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			continue
		}
		line := buildASSKaraokeLine(text, end-start, style)
		if line == "" {
			continue
		}
		b.WriteString("Dialogue: 0,")
		b.WriteString(formatASSTime(start - segmentStart))
		b.WriteString(",")
		b.WriteString(formatASSTime(end - segmentStart))
		b.WriteString(",Default,,0,0,0,,")
		b.WriteString(line)
		b.WriteString("\n")
		index++
	}

	if index == 0 {
		return ""
	}
	return strings.TrimSpace(b.String())
}

func buildASSKaraokeLine(text string, duration time.Duration, style string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	totalCs := int(duration.Seconds() * 100)
	if totalCs <= 0 {
		return ""
	}
	perWord := totalCs / len(words)
	if perWord <= 0 {
		perWord = 1
	}
	remaining := totalCs
	var b strings.Builder
	tag := "\\k"
	if style == "word_highlight" {
		tag = "\\kf"
	}
	for i, word := range words {
		length := perWord
		if i == len(words)-1 {
			length = remaining
		}
		if length <= 0 {
			length = 1
		}
		remaining -= length
		b.WriteString("{")
		b.WriteString(tag)
		b.WriteString(strconv.Itoa(length))
		b.WriteString("}")
		b.WriteString(assEscape(word))
		if i != len(words)-1 {
			b.WriteString(" ")
		}
	}
	return b.String()
}

func assEscape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "{", "\\{")
	value = strings.ReplaceAll(value, "}", "\\}")
	value = strings.ReplaceAll(value, "\n", "\\N")
	return value
}

func assColorFromHex(hex string) string {
	hex = strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if len(hex) != 6 {
		return "&H0000FFFF"
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return "&H0000FFFF"
	}
	return fmt.Sprintf("&H00%02X%02X%02X", b, g, r)
}

func formatSRTTime(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	totalMillis := int(value.Milliseconds())
	hours := totalMillis / 3600000
	minutes := (totalMillis % 3600000) / 60000
	seconds := (totalMillis % 60000) / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}

func formatASSTime(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	totalCentis := int(value.Milliseconds() / 10)
	hours := totalCentis / 360000
	minutes := (totalCentis % 360000) / 6000
	seconds := (totalCentis % 6000) / 100
	centis := totalCentis % 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centis)
}

func subtitleFilterForPath(path string, settings *OutputSettings) string {
	if path == "" {
		return ""
	}
	escaped := filepath.ToSlash(path)
	escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, ":", "\\:")
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".srt" {
		fontSize := 56
		if settings != nil && settings.FontSize > 0 {
			fontSize = settings.FontSize
		}
		style := fmt.Sprintf("Fontsize=%d,Alignment=2,MarginL=80,MarginR=80,MarginV=80,WrapStyle=2", fontSize)
		return fmt.Sprintf("subtitles=%s:force_style='%s'", escaped, style)
	}
	return fmt.Sprintf("subtitles=%s", escaped)
}
