package assistant

import (
	"strings"
	"testing"

	"encore.app/auth/authhandler"
)

// ════ ТЕСТЫ ВАЛИДАЦИИ ЗАПРОСА ════

// TestValidateRequest_EmptyMessage проверяет отклонение пустой строки.
func TestValidateRequest_EmptyMessage(t *testing.T) {
	err := validateRequest(&ChatRequest{Message: ""})
	if err == nil {
		t.Fatal("expected error for empty message, got nil")
	}
}

// TestValidateRequest_WhitespaceOnly проверяет отклонение строки из пробелов.
// Строка "   " после TrimSpace становится пустой — должна отклоняться.
func TestValidateRequest_WhitespaceOnly(t *testing.T) {
	for _, msg := range []string{" ", "\t", "\n", "   \t\n   "} {
		err := validateRequest(&ChatRequest{Message: msg})
		if err == nil {
			t.Errorf("expected error for whitespace-only message %q, got nil", msg)
		}
	}
}

// TestValidateRequest_TooLong проверяет отклонение сообщений длиннее maxMessageLen.
// Важно: длина считается в Unicode runes, не байтах.
// "а" (кириллица) — 2 байта, но 1 rune.
func TestValidateRequest_TooLong(t *testing.T) {
	msg := strings.Repeat("а", maxMessageLen+1)
	err := validateRequest(&ChatRequest{Message: msg})
	if err == nil {
		t.Fatalf("expected error for message of %d runes (max %d)", maxMessageLen+1, maxMessageLen)
	}
}

// TestValidateRequest_ExactlyMaxLen проверяет, что ровно maxMessageLen символов принимается.
func TestValidateRequest_ExactlyMaxLen(t *testing.T) {
	msg := strings.Repeat("а", maxMessageLen)
	err := validateRequest(&ChatRequest{Message: msg})
	if err != nil {
		t.Fatalf("expected no error for message of exactly %d runes, got: %v", maxMessageLen, err)
	}
}

// TestValidateRequest_Valid проверяет принятие обычных вопросов на всех поддерживаемых языках.
func TestValidateRequest_Valid(t *testing.T) {
	for _, msg := range []string{
		"Как добавить сотрудника?",
		"Қызметкерді қалай қосуға болады?",
		"How do I upload a SCORM course?",
	} {
		err := validateRequest(&ChatRequest{Message: msg})
		if err != nil {
			t.Errorf("expected no error for valid message %q, got: %v", msg, err)
		}
	}
}

// ════ ТЕСТЫ POST-PROCESSING ════

// TestStripFillerOpener проверяет удаление шаблонных вступлений.
// Покрывает RU, KK, EN и edge-cases.
func TestStripFillerOpener(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "russian opener with exclamation",
			input: "Конечно! Перейдите в /scorm и нажмите Добавить.",
			want:  "Перейдите в /scorm и нажмите Добавить.",
		},
		{
			name:  "russian opener with comma",
			input: "Конечно, для этого нужно перейти в /profile.",
			want:  "для этого нужно перейти в /profile.",
		},
		{
			name:  "english opener",
			input: "Certainly! To upload a SCORM course, go to /scorm.",
			want:  "To upload a SCORM course, go to /scorm.",
		},
		{
			name:  "kazakh opener",
			input: "Әрине, бұл бетке өту керек.",
			want:  "бұл бетке өту керек.",
		},
		{
			name:  "no opener — unchanged",
			input: "Перейдите в /contracts для просмотра бюджета.",
			want:  "Перейдите в /contracts для просмотра бюджета.",
		},
		{
			name:  "leading whitespace trimmed",
			input: "   \n\nПерейдите в /dzo.",
			want:  "Перейдите в /dzo.",
		},
		{
			name:  "only first opener removed",
			input: "Конечно! Разумеется, вы можете сделать это.",
			want:  "Разумеется, вы можете сделать это.",
		},
		{
			name:  "Разумеется opener",
			input: "Разумеется, вот шаги для импорта.",
			want:  "вот шаги для импорта.",
		},
		{
			name:  "Absolutely opener",
			input: "Absolutely! Here's how to assign a SCORM course.",
			want:  "Here's how to assign a SCORM course.",
		},
		{
			name:  "Безусловно opener",
			input: "Безусловно, перейдите в /employees.",
			want:  "перейдите в /employees.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFillerOpener(tt.input)
			if got != tt.want {
				t.Errorf("stripFillerOpener(%q)\n got  %q\n want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ════ ТЕСТЫ RBAC ════

// TestBuildUserMessage_RoleDirectives проверяет, что каждая роль получает
// правильную директиву в финальном user message.
//
// Ключевые инварианты:
//   - SA получает allowlist без запретов (полный доступ)
//   - EMP получает явный allowlist с "ТОЛЬКО к следующим разделам"
//   - ADM и HR получают и разрешения, и явные ЗАПРЕЩЕНО
func TestBuildUserMessage_RoleDirectives(t *testing.T) {
	tests := []struct {
		role       authhandler.UserRole
		roleName   string
		mustHave   string
		mustAbsent string
	}{
		{
			role:       authhandler.RoleSA,
			roleName:   "SA",
			mustHave:   "полный доступ",
			mustAbsent: "ЗАПРЕЩЕНО",
		},
		{
			role:       authhandler.RoleADM,
			roleName:   "ADM",
			mustHave:   "ЗАПРЕЩЕНО",
			mustAbsent: "ТОЛЬКО к следующим разделам",
		},
		{
			role:       authhandler.RoleHR,
			roleName:   "HR",
			mustHave:   "ЗАПРЕЩЕНО",
			mustAbsent: "ТОЛЬКО к следующим разделам",
		},
		{
			role:       authhandler.RoleEMP,
			roleName:   "EMP",
			mustHave:   "ТОЛЬКО к следующим разделам",
			mustAbsent: "полный доступ",
		},
	}

	for _, tt := range tests {
		t.Run("role_"+tt.roleName, func(t *testing.T) {
			msg := BuildUserMessage(tt.role, "user-test-id", "", "тестовый вопрос")

			if !strings.Contains(msg, tt.mustHave) {
				t.Errorf("role %s: mandatory phrase %q not found", tt.roleName, tt.mustHave)
			}
			if tt.mustAbsent != "" && strings.Contains(msg, tt.mustAbsent) {
				t.Errorf("role %s: forbidden phrase %q must not appear", tt.roleName, tt.mustAbsent)
			}

			// Код роли должен присутствовать в директиве
			if !strings.Contains(msg, tt.roleName) {
				t.Errorf("role code %q must appear in directive", tt.roleName)
			}
			// Вопрос пользователя всегда присутствует
			if !strings.Contains(msg, "тестовый вопрос") {
				t.Error("user question must always be present")
			}
			// ID сессии всегда присутствует (для трассировки)
			if !strings.Contains(msg, "user-test-id") {
				t.Error("user ID must always be present for tracing")
			}
			// Явный маркер вопроса защищает от путаницы инструкции с вопросом
			if !strings.Contains(msg, "[ВОПРОС ПОЛЬЗОВАТЕЛЯ]") {
				t.Error("question marker must always be present")
			}
		})
	}
}

// TestBuildUserMessage_WithLiveContext проверяет включение живого контекста.
func TestBuildUserMessage_WithLiveContext(t *testing.T) {
	liveCtx := "Актуальные мероприятия:\n- Тренинг по Go"
	msg := BuildUserMessage(authhandler.RoleEMP, "uid-1", liveCtx, "что сейчас есть?")

	if !strings.Contains(msg, "Тренинг по Go") {
		t.Error("live context content must be present")
	}
	if !strings.Contains(msg, "АКТУАЛЬНЫЙ КОНТЕКСТ") {
		t.Error("live context section header must be present")
	}
}

// TestBuildUserMessage_EmptyLiveContext проверяет, что пустой контекст
// не добавляет лишний раздел.
func TestBuildUserMessage_EmptyLiveContext(t *testing.T) {
	msg := BuildUserMessage(authhandler.RoleHR, "uid-2", "", "вопрос")

	if strings.Contains(msg, "АКТУАЛЬНЫЙ КОНТЕКСТ") {
		t.Error("empty live context must not add context section")
	}
}

// TestBuildUserMessage_QuestionAlwaysLast проверяет, что вопрос пользователя
// идёт ПОСЛЕ всех инструкций — важно для приоритизации контекста LLM.
func TestBuildUserMessage_QuestionAlwaysLast(t *testing.T) {
	msg := BuildUserMessage(authhandler.RoleADM, "uid-3", "live data", "мой вопрос")

	questionIdx := strings.Index(msg, "мой вопрос")
	markerIdx := strings.Index(msg, "[ВОПРОС ПОЛЬЗОВАТЕЛЯ]")
	rbacIdx := strings.Index(msg, "РОЛЬ ТЕКУЩЕГО")

	if questionIdx < markerIdx {
		t.Error("question must appear after the [ВОПРОС ПОЛЬЗОВАТЕЛЯ] marker")
	}
	if markerIdx < rbacIdx {
		t.Error("RBAC directive must appear before the question marker")
	}
}

// ════ ТЕСТЫ CONTEXT LOADER ════

// TestLoadDataFile_EmbeddedFiles проверяет, что встроенные data-файлы читаются корректно.
// system_prompt.md, context.md, navigations.md должны присутствовать в embed.FS.
func TestLoadDataFile_EmbeddedFiles(t *testing.T) {
	for _, name := range []string{"system_prompt.md", "context.md", "navigations.md"} {
		content, ok := loadDataFile(name)
		if !ok {
			t.Errorf("embedded file %q must be readable, got ok=false", name)
		}
		if content == "" {
			t.Errorf("embedded file %q must not be empty", name)
		}
	}
}

// TestLoadDataFile_MissingFile проверяет graceful degradation при отсутствии файла.
// Несуществующий файл должен вернуть ("", false) без паники.
func TestLoadDataFile_MissingFile(t *testing.T) {
	content, ok := loadDataFile("nonexistent_file_that_should_not_exist.md")
	if ok {
		t.Error("expected ok=false for nonexistent file, got true")
	}
	if content != "" {
		t.Errorf("expected empty content for nonexistent file, got %q", content)
	}
}

// TestSystemInstruction_NotEmpty проверяет, что systemInstruction собран
// (даже если один из файлов отсутствовал).
func TestSystemInstruction_NotEmpty(t *testing.T) {
	if systemInstruction == "" {
		t.Error("systemInstruction must not be empty after package init")
	}
}
