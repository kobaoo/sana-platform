package assistant

import (
	"context"
	"fmt"

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
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	userContext := buildContext(ad, req.Message)
	prompt := buildPrompt(userContext)
	reply, err := gemini.chat(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Reply: reply}, nil
}

// ════ INTERNAL ════

func buildContext(auth *authhandler.AuthData, message string) string {
	return fmt.Sprintf("User ID: %s\nRole: %s\nMessage: %s", auth.KeycloakUserID, string(auth.Role), message)
}

func buildPrompt(message string) string {
	return "You are an assistant for LMS.\n" + message
}

func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}