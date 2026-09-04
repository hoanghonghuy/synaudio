package audio

import "testing"

func TestFFmpegProcessorValidateRejectsUnavailableBinary(t *testing.T) {
	processor := NewFFmpegProcessor("synaudio-definitely-missing-ffmpeg-binary")
	if err := processor.Validate(); err == nil {
		t.Fatal("expected unavailable FFmpeg binary to fail validation")
	}
}
