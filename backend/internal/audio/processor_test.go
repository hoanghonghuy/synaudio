package audio

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestMockAudioProcessorConcatenates(t *testing.T) {
	p := NewMockAudioProcessor()

	out, err := p.Process(context.Background(), ProcessInput{
		Segments: []SegmentAudio{
			{Data: []byte("a"), DurationMs: 100},
			{Data: []byte("b"), DurationMs: 200},
			{Data: []byte("c"), DurationMs: 300},
		},
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if string(out.Data) != "abc" {
		t.Fatalf("expected concatenated data abc, got %q", out.Data)
	}
	if out.DurationMs != 600 {
		t.Fatalf("expected total duration 600, got %d", out.DurationMs)
	}
}

func TestBuildConcatCommand(t *testing.T) {
	cmd := buildConcatCommand([]string{"seg-0.mp3", "seg-1.mp3", "seg-2.mp3"}, "out.mp3", 96)

	if len(cmd) == 0 {
		t.Fatal("expected non-empty command")
	}
	if cmd[0] != "-y" {
		t.Fatalf("expected -y first, got %q", cmd[0])
	}

	inputCount := 0
	for i, arg := range cmd {
		if arg == "-i" {
			inputCount++
			if i+1 >= len(cmd) {
				t.Fatal("missing input path after -i")
			}
		}
	}
	if inputCount != 3 {
		t.Fatalf("expected 3 inputs, got %d", inputCount)
	}

	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, spokenWordLoudnorm) {
		t.Fatalf("expected spoken-word loudness normalization in command, got %q", joined)
	}
	if !strings.Contains(joined, "-map [aout]") {
		t.Fatalf("expected normalized audio output mapping, got %q", joined)
	}
	if !strings.Contains(joined, "-b:a 96k") {
		t.Fatalf("expected 96 kbps encode target, got %q", joined)
	}

	if cmd[len(cmd)-1] != "out.mp3" {
		t.Fatalf("expected output path last, got %q", cmd[len(cmd)-1])
	}
}

func TestParseFFmpegDuration(t *testing.T) {
	got, err := parseFFmpegDuration([]byte("Duration: 01:02:03.45, start: 0.000000, bitrate: 96 kb/s"))
	if err != nil {
		t.Fatalf("parse duration: %v", err)
	}
	if got != 3723450 {
		t.Fatalf("expected 3723450ms, got %d", got)
	}
}

func TestFFmpegProcessorUsesProbedFinalDuration(t *testing.T) {
	processor := &FFmpegProcessor{
		bin: "ffmpeg-test",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if containsArgs(args, "-f", "null") {
				return []byte("Duration: 00:00:01.23, start: 0.000000, bitrate: 96 kb/s"), nil
			}
			if err := os.WriteFile(args[len(args)-1], []byte("encoded-final-media"), 0o600); err != nil {
				return nil, err
			}
			return nil, nil
		},
	}

	out, err := processor.Process(context.Background(), ProcessInput{Segments: []SegmentAudio{
		{Data: []byte("provider-segment"), DurationMs: 9999},
	}})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.DurationMs != 1230 {
		t.Fatalf("expected final-media duration 1230ms instead of provider sum 9999ms, got %d", out.DurationMs)
	}
	if string(out.Data) != "encoded-final-media" {
		t.Fatalf("unexpected final media bytes %q", out.Data)
	}
}

func TestFFmpegProcessorFailsClosedWhenFinalDurationCannotBeProbed(t *testing.T) {
	processor := &FFmpegProcessor{
		bin: "ffmpeg-test",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if containsArgs(args, "-f", "null") {
				return []byte("final media has no readable duration"), nil
			}
			if err := os.WriteFile(args[len(args)-1], []byte("encoded-final-media"), 0o600); err != nil {
				return nil, err
			}
			return nil, nil
		},
	}

	_, err := processor.Process(context.Background(), ProcessInput{Segments: []SegmentAudio{
		{Data: []byte("provider-segment"), DurationMs: 9999},
	}})
	if err == nil {
		t.Fatal("expected unreadable final duration to fail processing")
	}
	if !strings.Contains(err.Error(), "probe final audio") {
		t.Fatalf("expected probe failure, got %v", err)
	}
}

func containsArgs(args []string, want ...string) bool {
	for i := 0; i+len(want) <= len(args); i++ {
		matched := true
		for j := range want {
			if args[i+j] != want[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
