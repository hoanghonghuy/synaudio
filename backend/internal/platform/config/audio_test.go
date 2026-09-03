package config

import "testing"

func TestProductionAudioProcessorDefaultsToFFmpeg(t *testing.T) {
	t.Setenv("AUDIO_PROCESSOR_MODE", "")
	t.Setenv("FFMPEG_BINARY", "")
	settings, err := LoadAudioProcessorSettings(EnvProduction)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Mode != AudioProcessorFFmpeg || settings.Binary != "ffmpeg" {
		t.Fatalf("settings = %#v, want production ffmpeg default", settings)
	}
}

func TestProductionRejectsMockAudioProcessor(t *testing.T) {
	t.Setenv("AUDIO_PROCESSOR_MODE", "mock")
	if _, err := LoadAudioProcessorSettings(EnvProduction); err == nil {
		t.Fatal("production must reject mock audio processor")
	}
}

func TestDevelopmentAllowsExplicitMockAudioProcessor(t *testing.T) {
	t.Setenv("AUDIO_PROCESSOR_MODE", "mock")
	settings, err := LoadAudioProcessorSettings(EnvDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Mode != AudioProcessorMock {
		t.Fatalf("mode = %q, want mock", settings.Mode)
	}
}
