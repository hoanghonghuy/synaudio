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
		if _, err := out.Write(data); err != nil {
			return 0, fmt.Errorf("write mock segment %d: %w", i, err)
		}
	}
	if err := out.Close(); err != nil {
		return 0, fmt.Errorf("close mock output: %w", err)
	}
	return len(inputPaths) * 400, nil
}
