// Package ai wraps the Anthropic (Claude) Messages API to generate quiz tests
// from a natural-language description. It is a thin HTTP client with no extra
// SDK dependency.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// apiURL is the Anthropic Messages endpoint.
	apiURL = "https://api.anthropic.com/v1/messages"
	// apiVersion is the required anthropic-version header value.
	apiVersion = "2023-06-01"
	// DefaultModel is used when ANTHROPIC_MODEL is not set. Claude Haiku 4.5 is
	// fast and cheap and great at returning strict JSON. Override via
	// ANTHROPIC_MODEL for other models (e.g. a Sonnet model for higher quality).
	DefaultModel = "claude-haiku-4-5-20251001"
	// maxTokens caps the generated response size.
	maxTokens = 4096
)

// systemPrompt instructs Claude to return only a strict JSON test document that
// matches the schema the bot already validates and stores.
const systemPrompt = `You are a quiz author. From the user's description you generate a quiz "test": a titled set of multiple-choice questions.

Return ONLY a single valid JSON object, with no markdown, no code fences, and no prose around it. The object MUST match this schema exactly:

{
  "title": "<short test name, 2-6 words>",
  "questions": [
    {
      "Name": "<the correct answer as a concept name, 2-5 words>",
      "Overview": "<one plain-English sentence summarising the concept>",
      "Question": "<a clear question ending with '?'>",
      "Explanation": "<2-4 sentence plain-English explanation, ideally with an analogy>",
      "Layer": "<a short grouping/category label for the question>"
    }
  ]
}

Rules:
- Generate exactly the number of questions the user asks for. If they do not specify a number, generate 5.
- Every field must be non-empty. "Question" must end with a question mark.
- "Name" is the single correct answer; the bot builds wrong options from other questions, so keep each "Name" distinct.
- Output the JSON object and nothing else.`

// Client talks to the Anthropic Messages API.
type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

// NewClient creates a Claude client. model may be empty to use DefaultModel.
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

type messageReq struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateTestJSON sends the description to Claude and returns the raw JSON test
// document (cleaned of any stray markdown fences). The caller validates and
// stores it. The returned bytes are intended to match the bot's test schema:
// { "title": ..., "questions": [...] }.
func (c *Client) GenerateTestJSON(ctx context.Context, description string) ([]byte, error) {
	reqBody, err := json.Marshal(messageReq{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: description}},
	})
	if err != nil {
		return nil, fmt.Errorf("ai: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: call claude: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("ai: read response: %w", err)
	}

	var parsed messageResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("ai: decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return nil, fmt.Errorf("ai: claude error (status %d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return nil, fmt.Errorf("ai: claude returned status %d", resp.StatusCode)
	}

	var sb strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return nil, fmt.Errorf("ai: claude returned no text content")
	}

	return []byte(extractJSON(text)), nil
}

// extractJSON strips Markdown code fences and any text outside the outermost
// JSON object, so minor formatting deviations from the model still parse.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Remove a leading ```json / ``` fence and trailing ``` if present.
	if strings.HasPrefix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl != -1 {
			s = s[nl+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	// Fall back to the substring between the first '{' and the last '}'.
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
