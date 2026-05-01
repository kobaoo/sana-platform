package assistant

import (
	"context"

	"encore.app/auth/authhandler"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

// ════ DATABASE ════

// DB section intentionally left empty for skeleton stage.

// ════ ENDPOINTS ════

var gemini = newGeminiClient()

// Chat sends a message to the AI assistant and returns its reply.
//
//encore:api auth method=POST path=/assistant/chat
func Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	_, err := getAuthData()
	if err != nil {
		return nil, err
	}

	reply, err := gemini.chat(ctx, req.Message)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Reply: reply}, nil
}

// ════ INTERNAL ════

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}