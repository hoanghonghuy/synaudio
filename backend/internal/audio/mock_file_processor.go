package audio

import (
	"context"
	"fmt"
	"os"
)

// ProcessFiles keeps the deterministic mock usable with the production
// file-oriented orchestration boundary. It is intentionally simple and is not a
// production media implementation.
func (MockAudioProcessor) ProcessFiles(ctx context.Context, inputPaths []string, outputPath string) (int, error) {
	if len(inputPaths) == 0 {
		return 0, fmt.Errorf("no segments to process")
	}
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, fmt.Errorf("create mock output: %w", err)
	}
	defer out.Close()

	allMock := true
	for i, path := range inputPaths {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, fmt.Errorf("read mock segment %d: %w", i, err)
		}
		if len(data) == 0 {
			return 0, fmt.Errorf("mock segment %d is empty", i)
		}
		if string(data) != "MOCK-AUDIO" {
			allMock = false
		}
		if _, err := out.Write(data); err != nil {
			return 0, fmt.Errorf("write mock segment %d: %w", i, err)
		}
	}
	if allMock {
		// When input segments are mock data ("MOCK-AUDIO"), write a valid, playable MP3
		// so browser media decoders can decode and play the audio smoothly.
		_ = out.Truncate(0)
		_, _ = out.Seek(0, 0)
		_, _ = out.Write(getMockPlayableMP3())
	}
	if err := out.Close(); err != nil {
		return 0, fmt.Errorf("close mock output: %w", err)
	}
	return len(inputPaths) * 400, nil
}
