package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"encore.dev/beta/errs"
)

// Encore loads GeminiAPIKey from .secrets.local.cue (local) or encore secret set (prod).
var secrets struct {
	GeminiAPIKey string
}

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

type geminiClient struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

func newGeminiClient() *geminiClient {
	return &geminiClient{
		apiKey:   secrets.GeminiAPIKey,
		endpoint: geminiEndpoint,
		http:     defaultHTTPClient,
	}
}

// ════ GEMINI REST TYPES ════

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ════ CLIENT ════

func (c *geminiClient) chat(ctx context.Context, message string) (string, error) {
	body, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: message}}},
		},
	})
	if err != nil {
		return "", errs.B().Code(errs.Internal).Msg("failed to encode request").Err()
	}

	url := fmt.Sprintf("%s?key=%s", c.endpoint, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", errs.B().Code(errs.Internal).Msg("failed to build request").Err()
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", errs.B().Code(errs.Unavailable).Msg("gemini unreachable").Cause(err).Err()
	}
	defer resp.Body.Close()

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errs.B().Code(errs.Internal).Msg("failed to decode gemini response").Err()
	}

	if result.Error != nil {
		return "", errs.B().Code(errs.Internal).Msgf("gemini error: %s", result.Error.Message).Err()
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", errs.B().Code(errs.Internal).Msg("empty response from gemini").Err()
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
