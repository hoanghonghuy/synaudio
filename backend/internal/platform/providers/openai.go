package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

type openAIClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func newOpenAIClient(baseURL, apiKey, model string) (*openAIClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("OpenAI model is required")
	}
	apiKey = strings.TrimSpace(apiKey)
	return &openAIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 90 * time.Second},
	}, nil
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIChatResponse struct {
	ID      string         `json:"id"`
	Choices []openAIChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type openAIAttemptResult struct {
	response   openAIChatResponse
	statusCode int
	headers    http.Header
	err        error
}

func (c *openAIClient) generate(ctx context.Context, prompt string, systemPrompt string) (openAIChatResponse, error) {
	var lastClass string
	var lastRetryAfter time.Duration

	for attempt := 0; attempt <= MaxInCallRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt, lastRetryAfter)
			if err := providerSleep(ctx, delay); err != nil {
				return openAIChatResponse{}, err
			}
		}

		started := time.Now()
		result := c.doGenerateOnce(ctx, prompt, systemPrompt)
		latency := time.Since(started)

		if result.err == nil {
			observeCall(CallEvent{
				Provider:      "openai",
				Operation:     OpGenerateContent,
				Outcome:       CallSucceeded,
				FailureClass:  "none",
				FailureCode:   "none",
				HTTPStatus:    result.statusCode,
				RetryDecision: retryDecisionForSuccess(attempt),
				Attempt:       attempt + 1,
				Latency:       latency,
			})
			return result.response, nil
		}

		if ctx.Err() != nil && errors.Is(result.err, ctx.Err()) {
			return openAIChatResponse{}, result.err
		}

		class, code, retryable, retryAfter := classifyAttemptFailure(attemptResult{
			statusCode: result.statusCode,
			headers:    result.headers,
			err:        result.err,
		})
		lastClass = class
		lastRetryAfter = retryAfter

		decision := RetryNotAllowed
		if retryable {
			if attempt < MaxInCallRetries {
				decision = RetryScheduled
			} else {
				decision = RetryExhausted
				code = "PROVIDER_RETRY_EXHAUSTED"
			}
		}

		observeCall(CallEvent{
			Provider:      "openai",
			Operation:     OpGenerateContent,
			Outcome:       CallFailed,
			FailureClass:  boundedFailureClass(class),
			FailureCode:   boundedFailureCode(code),
			HTTPStatus:    result.statusCode,
			RetryDecision: decision,
			Attempt:       attempt + 1,
			Latency:       latency,
		})

		if !retryable || attempt == MaxInCallRetries {
			return openAIChatResponse{}, classifiedProviderError(class, code, result.err)
		}
	}

	return openAIChatResponse{}, classifiedProviderError(lastClass, "PROVIDER_RETRY_EXHAUSTED", errors.New("provider in-call retries exhausted"))
}

func (c *openAIClient) doGenerateOnce(ctx context.Context, prompt string, systemPrompt string) openAIAttemptResult {
	messages := make([]openAIMessage, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: strings.TrimSpace(systemPrompt)})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: prompt})

	payload, err := json.Marshal(openAIChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return openAIAttemptResult{err: fmt.Errorf("encode OpenAI request: %w", err)}
	}

	endpoint := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return openAIAttemptResult{err: fmt.Errorf("create OpenAI request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return openAIAttemptResult{err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return openAIAttemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        &responseReadError{cause: err},
		}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return openAIAttemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        fmt.Errorf("provider HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	var out openAIChatResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return openAIAttemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", err),
		}
	}

	if out.Error != nil && out.Error.Message != "" {
		return openAIAttemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        classifiedProviderError("TRANSIENT", "PROVIDER_DOWNSTREAM_ERROR", errors.New(out.Error.Message)),
		}
	}

	if len(out.Choices) == 0 {
		return openAIAttemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("provider returned no choices")),
		}
	}

	return openAIAttemptResult{response: out, statusCode: resp.StatusCode, headers: resp.Header}
}

func openAIResponseText(resp openAIChatResponse) (string, error) {
	if len(resp.Choices) == 0 {
		return "", classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("provider returned no choices"))
	}
	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	if text == "" {
		return "", classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("provider returned no text"))
	}
	return text, nil
}

func cleanJSONMarkdown(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
	}
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
	}
	return strings.TrimSpace(text)
}

type openAIAI struct {
	client *openAIClient
}

func (a *openAIAI) GenerateText(ctx context.Context, in generation.TextAIInput) (generation.TextAIOutput, error) {
	resp, err := a.client.generate(ctx, in.Prompt, "")
	if err != nil {
		return generation.TextAIOutput{}, err
	}
	text, err := openAIResponseText(resp)
	if err != nil {
		return generation.TextAIOutput{}, err
	}
	return generation.TextAIOutput{Text: text, Provider: "openai", Model: a.client.model}, nil
}

func (a *openAIAI) ProposeFoundation(ctx context.Context, in planning.FoundationInput) (planning.FoundationProposal, error) {
	prompt := fmt.Sprintf(`Create a story foundation for this premise: %s
Return JSON only with this shape:
{"bible":{},"ending":{},"arcs":[{}],"characters":[{"name":"","importance":"MAJOR","profile":{}}]}
Use only MAJOR or MINOR for character importance.`, in.Premise)
	systemPrompt := "You are a professional story architect. You must respond with valid JSON only, without any markdown formatting or commentary."

	resp, err := a.client.generate(ctx, prompt, systemPrompt)
	if err != nil {
		return planning.FoundationProposal{}, err
	}
	text, err := openAIResponseText(resp)
	if err != nil {
		return planning.FoundationProposal{}, err
	}

	cleaned := cleanJSONMarkdown(text)

	type rawCharacter struct {
		Name       string          `json:"name"`
		Importance string          `json:"importance"`
		Profile    json.RawMessage `json:"profile"`
	}

	var proposal struct {
		Bible      map[string]any   `json:"bible"`
		Ending     map[string]any   `json:"ending"`
		Arcs       []map[string]any `json:"arcs"`
		Characters []rawCharacter   `json:"characters"`
	}
	if err := json.Unmarshal([]byte(cleaned), &proposal); err != nil {
		return planning.FoundationProposal{}, classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", err)
	}
	if proposal.Bible == nil || proposal.Ending == nil || len(proposal.Arcs) == 0 {
		return planning.FoundationProposal{}, classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("provider foundation response is incomplete"))
	}

	characters := make([]planning.CharacterProposal, 0, len(proposal.Characters))
	for _, c := range proposal.Characters {
		importance := strings.ToUpper(strings.TrimSpace(c.Importance))
		if importance != "MINOR" {
			importance = "MAJOR"
		}
		profileMap := make(map[string]any)
		if len(c.Profile) > 0 {
			if err := json.Unmarshal(c.Profile, &profileMap); err != nil {
				var strProfile string
				if errStr := json.Unmarshal(c.Profile, &strProfile); errStr == nil {
					profileMap["description"] = strProfile
				}
			}
		}
		characters = append(characters, planning.CharacterProposal{
			Name:       strings.TrimSpace(c.Name),
			Importance: importance,
			Profile:    profileMap,
		})
	}

	return planning.FoundationProposal{
		Bible:      proposal.Bible,
		Ending:     proposal.Ending,
		Arcs:       proposal.Arcs,
		Characters: characters,
	}, nil
}

func (a *openAIAI) ExtractMemory(ctx context.Context, in planning.MemoryExtractionInput) (planning.MemoryExtraction, error) {
	prompt := fmt.Sprintf(`Extract durable canonical facts from the approved chapter content below.
Return JSON only with this shape:
{"facts":[{"subject_type":"CHARACTER","subject_id":"...","fact_type":"...","value":{}}]}
Story ID: %s
Chapter ID: %s
Content:
%s`, in.StoryID, in.ChapterID, in.ContentText)
	systemPrompt := "You are a story continuity analyzer. You must respond with valid JSON only."

	resp, err := a.client.generate(ctx, prompt, systemPrompt)
	if err != nil {
		return planning.MemoryExtraction{}, err
	}
	text, err := openAIResponseText(resp)
	if err != nil {
		return planning.MemoryExtraction{}, err
	}

	cleaned := cleanJSONMarkdown(text)

	type rawFact struct {
		SubjectType    string         `json:"subject_type"`
		SubjectTypeAlt string         `json:"subjectType"`
		SubjectID      string         `json:"subject_id"`
		SubjectIDAlt   string         `json:"subjectId"`
		FactType       string         `json:"fact_type"`
		FactTypeAlt    string         `json:"factType"`
		Value          map[string]any `json:"value"`
	}

	var extraction struct {
		Facts []rawFact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(cleaned), &extraction); err != nil {
		return planning.MemoryExtraction{}, classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", err)
	}

	facts := make([]planning.ExtractedFact, 0, len(extraction.Facts))
	for _, f := range extraction.Facts {
		subjectType := f.SubjectType
		if subjectType == "" {
			subjectType = f.SubjectTypeAlt
		}
		subjectID := f.SubjectID
		if subjectID == "" {
			subjectID = f.SubjectIDAlt
		}
		factType := f.FactType
		if factType == "" {
			factType = f.FactTypeAlt
		}
		val := f.Value
		if val == nil {
			val = make(map[string]any)
		}
		facts = append(facts, planning.ExtractedFact{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			FactType:    factType,
			Value:       val,
		})
	}

	return planning.MemoryExtraction{Facts: facts}, nil
}

