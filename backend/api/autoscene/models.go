package autoscene

type Clip struct {
	Rank          int      `json:"rank"`
	Category      string   `json:"category"`
	Title         string   `json:"title"`
	Hashtags      []string `json:"hashtags"`
	StartTime     string   `json:"start_time"`
	EndTime       string   `json:"end_time"`
	Reason        string   `json:"reason"`
	ViralityScore int      `json:"virality_score"`
}

type AnalysisResult struct {
	Clips []Clip `json:"clips"`
}

// FrameSample represents a single keyframe sample with timestamp in seconds.
type FrameSample struct {
	TimestampSec float64
	JPEG         []byte
}

// AudioSample represents a short audio snippet with timestamp in seconds.
type AudioSample struct {
	StartSec    float64
	DurationSec float64
	WAV         []byte
}
