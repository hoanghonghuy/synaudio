package story

import "context"

// ActivationReadiness is the backend-authoritative result of evaluating the
// Story Activation Gate without mutating Story state.
type ActivationReadiness struct {
	Ready   bool
	Missing []string
}

// CheckActivationReadiness evaluates the same dependencies used by
// ActivateStory and returns actionable missing prerequisite identifiers.
func (s *Service) CheckActivationReadiness(ctx context.Context, storyID string) (ActivationReadiness, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return ActivationReadiness{}, err
	}

	missing := []string{}
	if st.PlanningMode == "" {
		missing = append(missing, "planning_mode")
	}

	hasPolicy, err := s.store.HasGenerationPolicy(ctx, storyID)
	if err != nil {
		return ActivationReadiness{}, err
	}
	if !hasPolicy {
		missing = append(missing, "generation_policy")
	}

	if _, err := s.store.GetCurrentContentProfile(ctx, storyID); err != nil {
		missing = append(missing, "content_profile")
	}

	if s.activation == nil {
		missing = append(missing, "planning_foundation")
	} else {
		planningMissing, err := s.activation.CheckActivationReady(ctx, storyID)
		if err != nil {
			return ActivationReadiness{}, err
		}
		missing = append(missing, planningMissing...)
	}

	return ActivationReadiness{Ready: len(missing) == 0, Missing: missing}, nil
}
