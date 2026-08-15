package planning

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrCharacterNotFound = errors.New("character not found")
)

// Character is a first-class, stable-identity entity.
type Character struct {
	ID                     string
	StoryID                string
	CanonicalName          string
	Importance             string
	CurrentProfileVersionID string
}

// CharacterProfileVersion is a versioned static profile.
type CharacterProfileVersion struct {
	ID          string
	CharacterID string
	VersionNo   int
	Profile     map[string]any
	CreatedBy   string
}

// CreateCharacter creates a new Character with an initial profile version.
func (s *Service) CreateCharacter(ctx context.Context, storyID, name, importance string, profile map[string]any, createdBy string) (Character, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Character{}, errors.New("character name must not be empty")
	}
	if importance == "" {
		importance = "MINOR"
	}

	c := Character{
		ID:            uuid.NewString(),
		StoryID:       storyID,
		CanonicalName: name,
		Importance:    importance,
	}

	created, err := s.store.CreateCharacter(ctx, c)
	if err != nil {
		return Character{}, err
	}

	versionNo, err := s.store.NextProfileVersion(ctx, created.ID)
	if err != nil {
		return Character{}, err
	}

	v := CharacterProfileVersion{
		ID:          uuid.NewString(),
		CharacterID: created.ID,
		VersionNo:   versionNo,
		Profile:     profile,
		CreatedBy:   createdBy,
	}

	if _, err := s.store.CreateProfileVersion(ctx, v); err != nil {
		return Character{}, err
	}

	return created, nil
}

// ListCharacters returns all characters for a story.
func (s *Service) ListCharacters(ctx context.Context, storyID string) ([]Character, error) {
	return s.store.ListCharacters(ctx, storyID)
}

// GetCharacter returns a single character by ID.
func (s *Service) GetCharacter(ctx context.Context, characterID string) (Character, error) {
	return s.store.GetCharacter(ctx, characterID)
}
