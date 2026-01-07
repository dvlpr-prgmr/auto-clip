package api

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultThumbnailTimestamp = 0.2

func generateThumbnail(clipPath string) (string, error) {
	return generateThumbnailAt(clipPath, defaultThumbnailTimestamp)
}

func generateThumbnailAt(clipPath string, timestampSec float64) (string, error) {
	if strings.TrimSpace(clipPath) == "" {
		return "", fmt.Errorf("clip path is empty")
	}

	outputDir := strings.TrimSpace(os.Getenv("THUMBNAIL_OUTPUT_DIR"))
	if outputDir == "" {
		outputDir = filepath.Join("output", "thumbnails")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create thumbnail dir: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(clipPath), filepath.Ext(clipPath))
	thumbPath := filepath.Join(outputDir, baseName+".jpg")

	if timestampSec < 0 {
		timestampSec = 0
	}

	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-ss", fmt.Sprintf("%.3f", timestampSec),
		"-i", clipPath,
		"-frames:v", "1",
		"-q:v", "2",
		thumbPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return thumbPath, fmt.Errorf("ffmpeg thumbnail: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return thumbPath, nil
}
