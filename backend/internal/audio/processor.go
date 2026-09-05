package audio

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

type ffmpegCommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

func runFFmpegCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// FFmpegProcessor invokes the ffmpeg binary to concatenate, normalize, encode,
// and inspect the final MP3.
type FFmpegProcessor struct {
	bin string
	run ffmpegCommandRunner
}

func NewFFmpegProcessor(bin string) *FFmpegProcessor {
	if strings.TrimSpace(bin) == "" {
		bin = "ffmpeg"
	}
	return &FFmpegProcessor{bin: strings.TrimSpace(bin), run: runFFmpegCommand}
}

// Validate checks that the configured executable can be resolved before the
// runtime accepts narration work. Process still returns execution errors later
// if the binary disappears or becomes unusable after startup.
func (p *FFmpegProcessor) Validate() error {
	if p == nil || strings.TrimSpace(p.bin) == "" {
		return fmt.Errorf("ffmpeg binary is not configured")
	}
	if _, err := exec.LookPath(p.bin); err != nil {
		return fmt.Errorf("ffmpeg binary %q unavailable: %w", p.bin, err)
	}
	return nil
}

// Process writes each segment to a temp file, then runs ffmpeg to concatenate,
// normalize and encode them into a single MP3. The final media itself is then
// probed; provider-declared segment timings are intentionally not used as the
// production duration authority.
func (p *FFmpegProcessor) Process(ctx context.Context, in ProcessInput) (ProcessOutput, error) {
	if len(in.Segments) == 0 {
		return ProcessOutput{}, fmt.Errorf("no segments to process")
	}
	if p == nil || p.run == nil {
		return ProcessOutput{}, fmt.Errorf("ffmpeg processor is not configured")
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

	if out, err := p.run(ctx, p.bin, args...); err != nil {
		return ProcessOutput{}, fmt.Errorf("ffmpeg failed: %w: %s", err, string(out))
	}

	data, err := os.ReadFile(output)
	if err != nil {
		return ProcessOutput{}, fmt.Errorf("read output: %w", err)
	}
	if len(data) == 0 {
		return ProcessOutput{}, fmt.Errorf("ffmpeg produced empty output")
	}

	durationMs, err := p.probeDuration(ctx, output)
	if err != nil {
		return ProcessOutput{}, err
	}

	return ProcessOutput{Data: data, DurationMs: durationMs}, nil
}

// Spoken-word audiobook target: integrated -18 LUFS, loudness range 7 LU and
// true peak -1.5 dBTP. These explicit bounded targets keep finalization
// deterministic while leaving the existing 96 kbps MP3 encode target intact.
const spokenWordLoudnorm = "loudnorm=I=-18:LRA=7:TP=-1.5"

// buildConcatCommand builds the ffmpeg argument list to concatenate inputs,
// normalize spoken-word loudness, and encode a single MP3 at the given bitrate
// (kbps).
func buildConcatCommand(inputs []string, output string, bitrateKbps int) []string {
	args := []string{"-y"}
	for _, in := range inputs {
		args = append(args, "-i", in)
	}
	filter := fmt.Sprintf("concat=n=%d:v=0:a=1,%s[aout]", len(inputs), spokenWordLoudnorm)
	args = append(args,
		"-filter_complex", filter,
		"-map", "[aout]",
		"-b:a", fmt.Sprintf("%dk", bitrateKbps),
		"-f", "mp3",
		output,
	)
	return args
}

var ffmpegDurationPattern = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)

func (p *FFmpegProcessor) probeDuration(ctx context.Context, output string) (int, error) {
	probeOutput, err := p.run(ctx, p.bin, "-hide_banner", "-i", output, "-f", "null", "-")
	if err != nil {
		return 0, fmt.Errorf("probe final audio: %w: %s", err, string(probeOutput))
	}
	durationMs, err := parseFFmpegDuration(probeOutput)
	if err != nil {
		return 0, fmt.Errorf("probe final audio: %w", err)
	}
	if durationMs <= 0 {
		return 0, fmt.Errorf("probe final audio: non-positive duration %dms", durationMs)
	}
	return durationMs, nil
}

func parseFFmpegDuration(output []byte) (int, error) {
	match := ffmpegDurationPattern.FindSubmatch(output)
	if len(match) != 4 {
		return 0, fmt.Errorf("duration not found in ffmpeg output")
	}

	hours, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, fmt.Errorf("parse duration hours: %w", err)
	}
	minutes, err := strconv.Atoi(string(match[2]))
	if err != nil {
		return 0, fmt.Errorf("parse duration minutes: %w", err)
	}
	seconds, err := strconv.ParseFloat(string(match[3]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration seconds: %w", err)
	}

	totalMs := int(math.Round((float64(hours*3600+minutes*60)+seconds)*1000))
	return totalMs, nil
}
