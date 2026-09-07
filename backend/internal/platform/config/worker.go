package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const maxWorkerIDLength = 64

// LoadWorkerID returns the durable runtime identity used for generation-job
// ownership and operational attribution. Production requires explicit injection;
// development gets a per-process random identity so zero-setup local workers do
// not share a misleading constant owner.
func LoadWorkerID(appEnv string) (string, error) {
	workerID := strings.TrimSpace(os.Getenv("WORKER_ID"))
	if workerID == "" {
		if appEnv != EnvDevelopment {
			return "", fmt.Errorf("WORKER_ID is required outside development")
		}
		generated, err := randomDevelopmentWorkerID()
		if err != nil {
			return "", fmt.Errorf("generate development WORKER_ID: %w", err)
		}
		return generated, nil
	}
	if err := validateWorkerID(workerID); err != nil {
		return "", fmt.Errorf("WORKER_ID: %w", err)
	}
	return workerID, nil
}

func randomDevelopmentWorkerID() (string, error) {
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	return "dev-" + hex.EncodeToString(entropy[:]), nil
}

func validateWorkerID(workerID string) error {
	if workerID == "" || len(workerID) > maxWorkerIDLength {
		return fmt.Errorf("must contain 1 to %d characters", maxWorkerIDLength)
	}
	for _, r := range workerID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return fmt.Errorf("contains unsupported characters")
		}
	}
	return nil
}
