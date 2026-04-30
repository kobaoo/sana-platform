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

// Chat returns a minimal mock assistant response.
//
//encore:api auth method=POST path=/assistant/chat
func Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	_, err := getAuthData()
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Reply: "Mock assistant response",
	}, nil
}

// ════ INTERNAL ════

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}
