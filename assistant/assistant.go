package assistant

import (
	"context"
	"fmt"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/courses/events"
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

	userContext := buildContext(ctx, ad, req.Message)
	prompt := buildPrompt(userContext)
	reply, err := gemini.chat(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Reply: reply}, nil
}

// ════ INTERNAL ════

func buildContext(ctx context.Context, auth *authhandler.AuthData, message string) string {
	coursesCtx := getCoursesContext(ctx, auth)
	return fmt.Sprintf("User ID: %s\nRole: %s\nMessage: %s\n\n%s", auth.KeycloakUserID, string(auth.Role), message, coursesCtx)
}

const coursesContextLimit = 5

func getCoursesContext(ctx context.Context, auth *authhandler.AuthData) string {
	if auth == nil {
		return "Courses:\n- Introduction to LMS\n- Compliance basics\n- Soft skills workshop"
	}
	resp, err := events.ListEvents(ctx, &events.ListEventsParams{})
	if err != nil || resp == nil || len(resp.Events) == 0 {
		return "Courses:\n- Introduction to LMS\n- Compliance basics\n- Soft skills workshop"
	}

	n := len(resp.Events)
	if n > coursesContextLimit {
		n = coursesContextLimit
	}
	var b strings.Builder
	b.WriteString("Courses:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "- %s\n", resp.Events[i].Title)
	}
	return strings.TrimSuffix(b.String(), "\n")
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