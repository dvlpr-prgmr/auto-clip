package api

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateThumbnail(clipPath string) (string, error) {
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

	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-ss", "0.2",
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
