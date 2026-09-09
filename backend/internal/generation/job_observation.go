package generation

// JobView is the authoritative admin projection of one durable GenerationJob.
type JobView struct {
	ID               string
	RunID            string
	JobType          string
	Status           string
	AttemptCount     int
	MaxAttempts      int
	LastErrorClass   string
	LastErrorCode    string
	Observation      string
	Retryable        bool
	AttemptsExhausted bool
}

// ObserveJob derives UI-safe observation state from durable job fields.
func ObserveJob(job GenerationJob) JobView {
	view := JobView{
		ID:             job.ID,
		RunID:          job.RunID,
		JobType:        job.JobType,
		Status:         job.Status,
		AttemptCount:   job.AttemptCount,
		MaxAttempts:    job.MaxAttempts,
		LastErrorClass: job.LastErrorClass,
		LastErrorCode:  job.LastErrorCode,
	}

	exhausted := attemptsExhausted(job)
	view.AttemptsExhausted = exhausted

	switch job.Status {
	case "SUCCEEDED":
		view.Observation = "succeeded"
	case "RUNNING":
		view.Observation = "running"
	case "PENDING":
		if exhausted {
			view.Observation = "exhausted"
			return view
		}
		view.Observation = "queued"
	case "FAILED":
		if exhausted {
			view.Observation = "exhausted"
			return view
		}
		if adminRetryPermitted(job) {
			view.Observation = "retryable"
			view.Retryable = true
			return view
		}
		view.Observation = "failed"
	default:
		view.Observation = "failed"
	}

	return view
}

func attemptsExhausted(job GenerationJob) bool {
	if job.AttemptCount >= job.MaxAttempts {
		return true
	}
	return job.LastErrorClass == "RETRY_EXHAUSTED" || job.LastErrorCode == "MAX_ATTEMPTS_EXHAUSTED"
}

func adminRetryPermitted(job GenerationJob) bool {
	if job.Status != "FAILED" {
		return false
	}
	if attemptsExhausted(job) {
		return false
	}
	switch job.LastErrorClass {
	case "TRANSIENT", "INTERRUPTED":
		return true
	default:
		return false
	}
}
