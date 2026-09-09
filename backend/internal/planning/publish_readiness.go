package planning

import "context"

// PublishReadiness is the backend-authoritative result of evaluating chapter
// publish prerequisites without mutating chapter state.
type PublishReadiness struct {
	Ready   bool
	Missing []string
}

// PublishContentAuthority reports whether a chapter has approved content.
type PublishContentAuthority interface {
	HasApprovedContent(ctx context.Context, chapterID string) (bool, error)
}

// PublishAudioAuthority reports missing audio dependencies blocking publish.
type PublishAudioAuthority interface {
	CheckPublishAudioReady(ctx context.Context, chapterID string) (missing []string, err error)
}

// PublishStoryAuthority reports missing story-level dependencies blocking publish.
type PublishStoryAuthority interface {
	CheckStoryPermitsChapterPublish(ctx context.Context, storyID string) (missing []string, err error)
}

type compositePublishChecker struct {
	chapters Store
	content  PublishContentAuthority
	audio    PublishAudioAuthority
	story    PublishStoryAuthority
}

// NewCompositePublishChecker evaluates authoritative publish prerequisites across
// content, audio, and story domains.
func NewCompositePublishChecker(
	chapters Store,
	content PublishContentAuthority,
	audio PublishAudioAuthority,
	story PublishStoryAuthority,
) PublishChecker {
	return &compositePublishChecker{
		chapters: chapters,
		content:  content,
		audio:    audio,
		story:    story,
	}
}

func (c *compositePublishChecker) CheckPublishReady(ctx context.Context, chapterID string) ([]string, error) {
	ch, err := c.chapters.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	missing := []string{}

	if c.content == nil {
		missing = append(missing, "approved_content_authority")
	} else {
		hasApproved, err := c.content.HasApprovedContent(ctx, chapterID)
		if err != nil {
			return nil, err
		}
		if !hasApproved {
			missing = append(missing, "approved_content")
		}
	}

	if c.audio == nil {
		missing = append(missing, "active_audio_authority")
	} else {
		audioMissing, err := c.audio.CheckPublishAudioReady(ctx, chapterID)
		if err != nil {
			return nil, err
		}
		missing = append(missing, audioMissing...)
	}

	if c.story == nil {
		missing = append(missing, "story_publish_authority")
	} else {
		storyMissing, err := c.story.CheckStoryPermitsChapterPublish(ctx, ch.StoryID)
		if err != nil {
			return nil, err
		}
		missing = append(missing, storyMissing...)
	}

	return missing, nil
}

// CheckPublishReadiness evaluates the same dependencies used by PublishChapter
// and returns actionable missing prerequisite identifiers.
func (s *Service) CheckPublishReadiness(ctx context.Context, chapterID string) (PublishReadiness, error) {
	if s.publishChecker == nil {
		return PublishReadiness{Missing: []string{"publish_authority"}}, nil
	}

	missing, err := s.publishChecker.CheckPublishReady(ctx, chapterID)
	if err != nil {
		return PublishReadiness{}, err
	}

	return PublishReadiness{Ready: len(missing) == 0, Missing: missing}, nil
}
