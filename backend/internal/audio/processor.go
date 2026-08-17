package audio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SegmentAudio is the synthesized audio for a single TTS segment.
type SegmentAudio struct {
	Data       []byte
	DurationMs int
}

// ProcessInput is the input to the audio processor (FFmpeg).
type ProcessInput struct {
	Segments []SegmentAudio
}

// ProcessOutput is the result of audio processing.
type ProcessOutput struct {
	Data       []byte
	DurationMs int
}

// AudioProcessor concatenates, normalizes, and encodes segment audio into a
// single final audio file.
type AudioProcessor interface {
	Process(ctx context.Context, in ProcessInput) (ProcessOutput, error)
}

// MockAudioProcessor concatenates segment bytes without invoking FFmpeg.
type MockAudioProcessor struct{}

func NewMockAudioProcessor() *MockAudioProcessor {
	return &MockAudioProcessor{}
}

func (MockAudioProcessor) Process(_ context.Context, in ProcessInput) (ProcessOutput, error) {
	var buf bytes.Buffer
	total := 0
	for _, seg := range in.Segments {
		buf.Write(seg.Data)
		total += seg.DurationMs
	}
	return ProcessOutput{Data: buf.Bytes(), DurationMs: total}, nil
}

// FFmpegProcessor invokes the ffmpeg binary to concatenate, normalize, and
// encode segment audio into a single MP3.
type FFmpegProcessor struct {
	bin string
}

func NewFFmpegProcessor(bin string) *FFmpegProcessor {
	if bin == "" {
		bin = "ffmpeg"
	}
	return &FFmpegProcessor{bin: bin}
}

// Process writes each segment to a temp file, then runs ffmpeg to concatenate
// and encode them into a single MP3.
func (p *FFmpegProcessor) Process(ctx context.Context, in ProcessInput) (ProcessOutput, error) {
	if len(in.Segments) == 0 {
		return ProcessOutput{}, fmt.Errorf("no segments to process")
	}

	dir, err := os.MkdirTemp("", "synaudio-ffmpeg-*")
	if err != nil {
		return ProcessOutput{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	inputs := make([]string, 0, len(in.Segments))
	for i, seg := range in.Segments {
		path := filepath.Join(dir, fmt.Sprintf("seg-%d.mp3", i))
		if err := os.WriteFile(path, seg.Data, 0o600); err != nil {
			return ProcessOutput{}, fmt.Errorf("write segment %d: %w", i, err)
		}
		inputs = append(inputs, path)
	}

	output := filepath.Join(dir, "out.mp3")
	args := buildConcatCommand(inputs, output, 96)

	cmd := exec.CommandContext(ctx, p.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return ProcessOutput{}, fmt.Errorf("ffmpeg failed: %w: %s", err, string(out))
	}

	data, err := os.ReadFile(output)
	if err != nil {
		return ProcessOutput{}, fmt.Errorf("read output: %w", err)
	}

	total := 0
	for _, seg := range in.Segments {
		total += seg.DurationMs
	}

	return ProcessOutput{Data: data, DurationMs: total}, nil
}

// buildConcatCommand builds the ffmpeg argument list to concatenate inputs and
// encode a single MP3 at the given bitrate (kbps).
func buildConcatCommand(inputs []string, output string, bitrateKbps int) []string {
	args := []string{"-y"}
	for _, in := range inputs {
		args = append(args, "-i", in)
	}
	args = append(args,
		"-filter_complex", fmt.Sprintf("concat=n=%d:v=0:a=1", len(inputs)),
		"-b:a", fmt.Sprintf("%dk", bitrateKbps),
		"-f", "mp3",
		output,
	)
	return args
}
