package audio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const audioChecksumPrefix = "sha256:"

var (
	ErrChecksumMissing  = errors.New("audio checksum missing")
	ErrChecksumMismatch = errors.New("audio checksum mismatch")
)

// ComputeAudioFileChecksum hashes the exact bytes of a finalized audio file
// without materializing the file in the Go heap. The persisted representation
// is "sha256:<lowercase hex>" so algorithm identity is explicit and future
// migrations can fail closed instead of guessing.
func ComputeAudioFileChecksum(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open audio for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := f.Read(buf)
		if n > 0 {
			if _, err := h.Write(buf[:n]); err != nil {
				return "", fmt.Errorf("hash audio: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read audio for checksum: %w", readErr)
		}
	}
	return audioChecksumPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyAudioFileChecksum provides the fail-honest boundary consumed by
// recovery/restore verification. Missing legacy checksums are explicitly
// unverified; malformed/unknown algorithms and byte mismatches fail closed.
func VerifyAudioFileChecksum(ctx context.Context, path, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return ErrChecksumMissing
	}
	if !strings.HasPrefix(expected, audioChecksumPrefix) {
		return fmt.Errorf("unsupported audio checksum format")
	}
	hexDigest := strings.TrimPrefix(expected, audioChecksumPrefix)
	if len(hexDigest) != sha256.Size*2 {
		return fmt.Errorf("invalid sha256 checksum length")
	}
	if _, err := hex.DecodeString(hexDigest); err != nil {
		return fmt.Errorf("invalid sha256 checksum encoding: %w", err)
	}
	actual, err := ComputeAudioFileChecksum(ctx, path)
	if err != nil {
		return err
	}
	if actual != expected {
		return ErrChecksumMismatch
	}
	return nil
}
