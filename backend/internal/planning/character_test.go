package planning

import (
	"context"
	"errors"
	"testing"
)

type charFakeStore struct {
	*fakeStore
	characters map[string][]Character
	profiles   map[string][]CharacterProfileVersion
	nextProf   map[string]int
}

func newCharFakeStore() *charFakeStore {
	return &charFakeStore{
		fakeStore:  newFakeStore(),
		characters: map[string][]Character{},
		profiles:   map[string][]CharacterProfileVersion{},
		nextProf:   map[string]int{},
	}
}

func (s *charFakeStore) CreateCharacter(ctx context.Context, c Character) (Character, error) {
	s.characters[c.StoryID] = append(s.characters[c.StoryID], c)
	return c, nil
}

func (s *charFakeStore) NextProfileVersion(ctx context.Context, characterID string) (int, error) {
	s.nextProf[characterID]++
	return s.nextProf[characterID], nil
}

func (s *charFakeStore) CreateProfileVersion(ctx context.Context, v CharacterProfileVersion) (CharacterProfileVersion, error) {
	s.profiles[v.CharacterID] = append(s.profiles[v.CharacterID], v)
	return v, nil
}

func (s *charFakeStore) ListCharacters(ctx context.Context, storyID string) ([]Character, error) {
	return s.characters[storyID], nil
}

func (s *charFakeStore) GetCharacter(ctx context.Context, characterID string) (Character, error) {
	for _, cs := range s.characters {
		for _, c := range cs {
			if c.ID == characterID {
				return c, nil
			}
		}
	}
	return Character{}, ErrCharacterNotFound
}

func TestCreateCharacterWithProfile(t *testing.T) {
	store := newCharFakeStore()
	svc := NewService(store)

	c, err := svc.CreateCharacter(context.Background(), "s1", "Alice", "MAJOR", map[string]any{"personality": "brave"}, "u1")
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	if c.CanonicalName != "Alice" {
		t.Fatalf("expected name Alice, got %q", c.CanonicalName)
	}
	if c.Importance != "MAJOR" {
		t.Fatalf("expected MAJOR, got %q", c.Importance)
	}
}

func TestCreateCharacterRejectsEmptyName(t *testing.T) {
	store := newCharFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateCharacter(context.Background(), "s1", "  ", "MAJOR", map[string]any{}, "u1"); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestListCharactersReturnsAll(t *testing.T) {
	store := newCharFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateCharacter(context.Background(), "s1", "Alice", "MAJOR", map[string]any{}, "u1")
	_, _ = svc.CreateCharacter(context.Background(), "s1", "Bob", "MINOR", map[string]any{}, "u1")

	chars, err := svc.ListCharacters(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(chars))
	}
}

func TestGetCharacterReturnsNotFound(t *testing.T) {
	store := newCharFakeStore()
	svc := NewService(store)

	if _, err := svc.GetCharacter(context.Background(), "missing"); !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("expected ErrCharacterNotFound, got %v", err)
	}
}
