package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	AudioProcessorMock   = "mock"
	AudioProcessorFFmpeg = "ffmpeg"
)

type AudioProcessorSettings struct {
	Mode   string
	Binary string
}

// LoadAudioProcessorSettings keeps deterministic mock processing available only
// through an explicit non-production mode. Production defaults to and requires
// the real FFmpeg processor.
func LoadAudioProcessorSettings(appEnv string) (AudioProcessorSettings, error) {
	fallback := AudioProcessorMock
	if strings.EqualFold(strings.TrimSpace(appEnv), EnvProduction) {
		fallback = AudioProcessorFFmpeg
	}
	mode := strings.ToLower(strings.TrimSpace(getenv("AUDIO_PROCESSOR_MODE", fallback)))
	binary := strings.TrimSpace(os.Getenv("FFMPEG_BINARY"))
	if binary == "" {
		binary = "ffmpeg"
	}

	switch mode {
	case AudioProcessorMock:
		if strings.EqualFold(strings.TrimSpace(appEnv), EnvProduction) {
			return AudioProcessorSettings{}, fmt.Errorf("mock AUDIO_PROCESSOR_MODE is not allowed in production")
		}
	case AudioProcessorFFmpeg:
		// Valid; binary availability is checked by the runtime before serving.
	default:
		return AudioProcessorSettings{}, fmt.Errorf("unsupported AUDIO_PROCESSOR_MODE %q", mode)
	}

	return AudioProcessorSettings{Mode: mode, Binary: binary}, nil
}
