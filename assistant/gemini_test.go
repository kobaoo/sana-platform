package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func makeTestClient(srv *httptest.Server) *geminiClient {
	return &geminiClient{
		apiKey:   "test-key",
		endpoint: srv.URL,
		http:     srv.Client(),
	}
}

func TestGeminiChat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req geminiRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Contents[0].Parts[0].Text != "hello" {
			t.Errorf("unexpected message: %+v", req)
		}
		json.NewEncoder(w).Encode(geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: "world"}}}},
			},
		})
	}))
	defer srv.Close()

	got, err := makeTestClient(srv).chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestGeminiChat_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(geminiResponse{
			Error: &geminiError{Code: 400, Message: "invalid key", Status: "INVALID_ARGUMENT"},
		})
	}))
	defer srv.Close()

	_, err := makeTestClient(srv).chat(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGeminiChat_EmptyCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(geminiResponse{Candidates: []geminiCandidate{}})
	}))
	defer srv.Close()

	_, err := makeTestClient(srv).chat(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
}

func TestGeminiChat_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("hijacking not supported")
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	_, err := makeTestClient(srv).chat(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}
