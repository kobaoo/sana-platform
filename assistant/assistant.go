package assistant

import (
	"context"
	"fmt"
	"strings"

	"encore.app/auth/authhandler"
	"encore.app/courses/events"
	"encore.app/learning/certificates"
	"encore.app/orgstructure/employees"
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
	employeeCtx := getEmployeeContext(ctx)
	coursesCtx := getCoursesContext(ctx, auth)
	certsCtx := getCertificatesContext(ctx, auth)
	return fmt.Sprintf("User ID: %s\nRole: %s\nMessage: %s\n\n%s\n\n%s\n\n%s", auth.KeycloakUserID, string(auth.Role), message, employeeCtx, coursesCtx, certsCtx)
}

func getEmployeeContext(ctx context.Context) string {
	resp, err := employees.GetMyEmployee(ctx)
	if err != nil || resp == nil {
		return "Employee:\n* No employee data"
	}
	emp := resp.Employee
	name := emp.FullName
	pos := ""
	if emp.Position != nil {
		pos = *emp.Position
	}
	if pos == "" {
		pos = "Not specified"
	}
	dept := strings.TrimSpace(emp.DzoName)
	if emp.Department != nil && strings.TrimSpace(*emp.Department) != "" {
		dept = strings.TrimSpace(*emp.Department)
	}
	return fmt.Sprintf("Employee:\n* Name: %s\n* Position: %s\n* Department: %s", name, pos, dept)
}

const coursesContextLimit = 5

func coursesFallback() string {
	return "Courses:\n- Introduction to LMS\n- Compliance basics\n- Soft skills workshop"
}

func getCoursesContext(ctx context.Context, auth *authhandler.AuthData) string {
	if auth == nil {
		return coursesFallback()
	}
	resp, err := events.ListEvents(ctx, &events.ListEventsParams{})
	if err != nil || resp == nil || len(resp.Events) == 0 {
		return coursesFallback()
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

const certificatesContextLimit = 5

func getCertificatesContext(ctx context.Context, auth *authhandler.AuthData) string {
	if auth == nil {
		return certificatesFallback()
	}
	resp, err := certificates.MyCertificates(ctx)
	if err != nil || resp == nil || len(resp.Certificates) == 0 {
		return certificatesFallback()
	}

	n := len(resp.Certificates)
	if n > certificatesContextLimit {
		n = certificatesContextLimit
	}
	var b strings.Builder
	b.WriteString("Certificates:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "* %s\n", resp.Certificates[i].Title)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func certificatesFallback() string {
	return "Certificates:\n* No certificates found"
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
