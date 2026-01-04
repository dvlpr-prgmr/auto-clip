package autoscene

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
	model  string
}

// NewClient menginisialisasi client menggunakan google.golang.org/genai
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	// Inisialisasi client dengan konfigurasi API Key
	// BackendGoogleAI adalah default untuk penggunaan via AI Studio (API Key)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &Client{
		client: client,
		model:  resolveModel(),
	}, nil
}

type AnalyzeOptions struct {
	MaxClips        int
	MinDurationSecs float64
	MaxDurationSecs float64
	LanguageHint    string
}

func (c *Client) Analyze(ctx context.Context, transcript string) (*AnalysisResult, error) {
	return c.AnalyzeWithOptions(ctx, transcript, AnalyzeOptions{})
}

func (c *Client) AnalyzeWithOptions(ctx context.Context, transcript string, options AnalyzeOptions) (*AnalysisResult, error) {
	if transcript == "" {
		return nil, errors.New("transcript cannot be empty")
	}

	// 1. Menyusun Prompt
	maxClips := options.MaxClips
	if maxClips <= 0 {
		maxClips = 3
	}
	minHint := "no minimum"
	if options.MinDurationSecs > 0 {
		minHint = fmt.Sprintf("%.1fs", options.MinDurationSecs)
	}
	maxHint := "no maximum"
	if options.MaxDurationSecs > 0 {
		maxHint = fmt.Sprintf("%.1fs", options.MaxDurationSecs)
	}
	langHint := strings.TrimSpace(options.LanguageHint)
	if langHint == "" || strings.EqualFold(langHint, "auto") {
		langHint = "match the transcript language"
	}

	promptText := fmt.Sprintf(`You are an expert Video Editor AI.
Analyze the following video transcript and identify the best scenes suitable for viral short clips (Shorts/Reels/TikTok).

TRANSCRIPT (time-coded, starts at 00:00):
%s

INSTRUCTIONS:
- Identify scenes with high emotion, humor, or insight.
- Return at most %d clips.
- Clip duration: min %s, max %s.
- Use timecodes relative to transcript start (00:00), format "MM:SS".
- Write reasons in %s.
- Return strictly a JSON object matching this schema:
{
  "clips": [
    {
      "rank": 1,
      "category": "Emotional/Funny",
      "start_time": "MM:SS",
      "end_time": "MM:SS",
      "reason": "Brief explanation",
      "virality_score": 85
    }
  ]
}`, transcript, maxClips, minHint, maxHint, langHint)

	// 2. Konfigurasi Request
	// Di library baru ini, Config dipisah agar lebih rapi.
	// Kita set ResponseMimeType ke "application/json" agar output valid.
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr(float32(0.2)), // Low temp untuk konsistensi
	}

	return c.generate(ctx, genai.Text(promptText), config)
}

func (c *Client) AnalyzeMultimodalWithOptions(ctx context.Context, frames []FrameSample, audio []AudioSample, options AnalyzeOptions) (*AnalysisResult, error) {
	if len(frames) == 0 && len(audio) == 0 {
		return nil, errors.New("frames and audio cannot be empty")
	}

	maxClips := options.MaxClips
	if maxClips <= 0 {
		maxClips = 3
	}
	minHint := "no minimum"
	if options.MinDurationSecs > 0 {
		minHint = fmt.Sprintf("%.1fs", options.MinDurationSecs)
	}
	maxHint := "no maximum"
	if options.MaxDurationSecs > 0 {
		maxHint = fmt.Sprintf("%.1fs", options.MaxDurationSecs)
	}
	langHint := strings.TrimSpace(options.LanguageHint)
	if langHint == "" || strings.EqualFold(langHint, "auto") {
		langHint = "match the transcript language"
	}

	promptText := fmt.Sprintf(`You are an expert Video Editor AI.
You will receive keyframes sampled across a video range (time starts at 00:00).
You may also receive short audio snippets to provide additional context.
Identify the best scenes suitable for viral short clips (Shorts/Reels/TikTok).

INSTRUCTIONS:
- Use the visual cues in the frames to infer scene changes and emotional peaks.
- Return at most %d clips.
- Clip duration: min %s, max %s.
- Use timecodes relative to 00:00, format "MM:SS".
- Write reasons in %s.
- Return strictly a JSON object matching this schema:
{
  "clips": [
    {
      "rank": 1,
      "category": "Emotional/Funny",
      "start_time": "MM:SS",
      "end_time": "MM:SS",
      "reason": "Brief explanation",
      "virality_score": 85
    }
  ]
}`, maxClips, minHint, maxHint, langHint)

	parts := make([]*genai.Part, 0, 1+len(frames)*2+len(audio)*2)
	parts = append(parts, genai.NewPartFromText(promptText))
	for _, frame := range frames {
		if len(frame.JPEG) == 0 {
			continue
		}
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("Frame at %s", formatTimecode(frame.TimestampSec))))
		parts = append(parts, genai.NewPartFromBytes(frame.JPEG, "image/jpeg"))
	}
	for _, sample := range audio {
		if len(sample.WAV) == 0 || sample.DurationSec <= 0 {
			continue
		}
		startLabel := formatTimecode(sample.StartSec)
		endLabel := formatTimecode(sample.StartSec + sample.DurationSec)
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("Audio snippet from %s to %s", startLabel, endLabel)))
		parts = append(parts, genai.NewPartFromBytes(sample.WAV, "audio/wav"))
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr(float32(0.2)),
	}

	return c.generate(ctx, []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}, config)
}

func (c *Client) generate(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*AnalysisResult, error) {
	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generate content error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, errors.New("gemini returned no candidates")
	}

	jsonString := extractResponseText(resp.Candidates[0].Content.Parts)
	jsonString = cleanJSONMarkdown(jsonString)
	if jsonString == "" {
		return nil, errors.New("failed to extract text from response")
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (raw: %s)", err, jsonString)
	}
	return &result, nil
}

func extractResponseText(parts []*genai.Part) string {
	var jsonString string
	for _, part := range parts {
		text := strings.TrimSpace(part.Text)
		if text != "" {
			jsonString += text
		}
	}
	return strings.TrimSpace(jsonString)
}

func formatTimecode(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds + 0.5)
	min := total / 60
	sec := total % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

// Helper untuk membersihkan format markdown code block ```json ... ```
func cleanJSONMarkdown(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "```json") {
		input = strings.TrimPrefix(input, "```json")
		input = strings.TrimSuffix(input, "```")
	} else if strings.HasPrefix(input, "```") {
		input = strings.TrimPrefix(input, "```")
		input = strings.TrimSuffix(input, "```")
	}
	return strings.TrimSpace(input)
}

func resolveModel() string {
	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return model
}
