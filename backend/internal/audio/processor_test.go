package audio

import (
	"context"
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

	// Every input must be preceded by -i.
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

	// Output path must be the last argument.
	if cmd[len(cmd)-1] != "out.mp3" {
		t.Fatalf("expected output path last, got %q", cmd[len(cmd)-1])
	}
}
