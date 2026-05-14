package assistant

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"encore.app/auth/authhandler"
	"encore.app/courses/events"
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

// ════ КОНСТАНТЫ ════

const (
	// maxMessageLen — максимальная длина входящего сообщения в символах Unicode.
	// Защищает от prompt-injection через аномально длинные сообщения
	// и от лишних расходов токенов.
	maxMessageLen = 500

	// coursesContextLimit — максимальное число актуальных событий/курсов,
	// передаваемых в запрос к LLM как "живой контекст".
	coursesContextLimit = 5
)

// ════ ИНИЦИАЛИЗАЦИЯ ════

// gemini — единственный экземпляр клиента, создаётся один раз при запуске.
var gemini = newGeminiClient()

// ════ ЭНДПОИНТ ════

// Chat — основной эндпоинт AI-ассистента Sana Expert.
//
// Архитектура запроса к LLM (два независимых слоя):
//
//	┌─ system_instruction (СТАТИЧЕСКИЙ, собирается один раз в context.go) ────────┐
//	│  Поведенческие инструкции + База знаний платформы (роли, навигация, флоу)   │
//	└────────────────────────────────────────────────────────────────────────────┘
//	┌─ user message (ДИНАМИЧЕСКИЙ, формируется в rbac.go BuildUserMessage) ───────┐
//	│  RBAC-директива для роли  │  UserID  │  Live context  │  Вопрос            │
//	└────────────────────────────────────────────────────────────────────────────┘
//
//encore:api auth method=POST path=/assistant/chat
func Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// ── Шаг 1: Валидация входящего сообщения ──────────────────────────────────
	// Выполняется до любых сетевых вызовов — быстрый fail-fast.
	// validateRequest возвращает errs.InvalidArgument → HTTP 400.
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// ── Шаг 2: Извлечение данных аутентификации из JWT ────────────────────────
	// auth.Data() возвращает *authhandler.AuthData, заполненную authhandler.go
	// из JWT-claims: Role (SA/ADM/HR/EMP), KeycloakUserID, CompanyID, DzoID.
	// Роль берётся ТОЛЬКО из токена — не из тела запроса, не из заголовков.
	ad, err := getAuthData()
	if err != nil {
		return nil, err
	}

	// ── Шаг 3: Живой контекст платформы ──────────────────────────────────────
	// Актуальные мероприятия из БД. Ошибка не блокирует запрос —
	// ассистент работает и без них, просто не сможет отвечать на
	// вопросы вида "какие мероприятия сейчас есть?".
	liveCtx := getLiveContext(ctx, ad)

	// ── Шаг 4: Формирование user message ──────────────────────────────────────
	// BuildUserMessage (rbac.go) инжектирует:
	//   • RBAC-директиву (allowlist/denylist для конкретной роли)
	//   • ID сессии (для трассировки)
	//   • Живой контекст (если есть)
	//   • Вопрос пользователя под маркером [ВОПРОС ПОЛЬЗОВАТЕЛЯ]
	userMessage := BuildUserMessage(ad.Role, ad.KeycloakUserID, liveCtx, req.Message)

	// ── Шаг 5: Запрос к Gemini ────────────────────────────────────────────────
	// chat() возвращает три значения:
	//   reply   — текст ответа (после stripFillerOpener)
	//   blocked — true если Safety filters заблокировали ответ (не ошибка)
	//   err     — инфраструктурная проблема (сеть, API-ключ, decode)
	//
	// Разделение blocked/err важно для фронтенда:
	//   blocked=true → показываем предупреждение, не красную ошибку
	//   err != nil   → показываем "попробуйте позже"
	reply, blocked, err := gemini.chat(ctx, systemInstruction, userMessage)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Reply: reply, Blocked: blocked}, nil
}

// ════ ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ════

// validateRequest проверяет корректность входящего запроса.
//
// Правила:
//   - Пустая строка или строка только из пробелов → HTTP 400
//   - Длина > maxMessageLen символов Unicode → HTTP 400
//
// Считаем длину в Unicode runes (не байтах), чтобы корректно
// обрабатывать кириллицу, казахские символы и эмодзи.
func validateRequest(req *ChatRequest) error {
	trimmed := strings.TrimSpace(req.Message)

	if trimmed == "" {
		return errs.B().
			Code(errs.InvalidArgument).
			Msg("Сообщение не может быть пустым").
			Err()
	}

	if utf8.RuneCountInString(trimmed) > maxMessageLen {
		return errs.B().
			Code(errs.InvalidArgument).
			Msgf("Сообщение слишком длинное. Максимум %d символов", maxMessageLen).
			Err()
	}

	return nil
}

// getAuthData извлекает AuthData из контекста Encore.
// Эндпоинт помечен //encore:api auth, поэтому в теории этот путь
// не достижим без валидного токена — но защита нужна на случай
// нестандартных сценариев вызова (service-to-service без auth).
func getAuthData() (*authhandler.AuthData, error) {
	ad, ok := auth.Data().(*authhandler.AuthData)
	if !ok {
		return nil, errs.B().Code(errs.Unauthenticated).Msg("not authenticated").Err()
	}
	return ad, nil
}

// getLiveContext запрашивает актуальные мероприятия из БД для обогащения
// контекста LLM. При любой ошибке возвращает пустую строку — деградация
// graceful, ассистент продолжает работать.
func getLiveContext(ctx context.Context, ad *authhandler.AuthData) string {
	if ad == nil {
		return ""
	}

	resp, err := events.ListEvents(ctx, &events.ListEventsParams{})
	if err != nil || resp == nil || len(resp.Events) == 0 {
		return ""
	}

	n := len(resp.Events)
	if n > coursesContextLimit {
		n = coursesContextLimit
	}

	var b strings.Builder
	b.WriteString("Актуальные мероприятия на платформе:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "- %s\n", resp.Events[i].Title)
	}

	return strings.TrimSuffix(b.String(), "\n")
}
