package assistant

import (
	"embed"
	"log/slog"
	"strings"
)

// dataFS встраивает всю директорию data/ в бинарник на этапе компиляции.
// Используем embed.FS (не прямой //go:embed var string), чтобы читать
// каждый файл через dataFS.ReadFile() и обрабатывать ошибки per-file —
// сервис не упадёт, если один из файлов отсутствует.
//
//go:embed data
var dataFS embed.FS

// defaultSystemInstruction — резервный промпт, используемый когда ни один
// из data-файлов не был встроен или успешно прочитан.
// Гарантирует, что сервис остаётся работоспособным даже при неполной сборке.
const defaultSystemInstruction = "Ты — корпоративный ассистент платформы Sana LMS. " +
	"Помогай пользователям ориентироваться на платформе. " +
	"Отвечай на том же языке, на котором задан вопрос."

// systemInstruction — итоговый текст для поля system_instruction в Gemini API.
// Собирается ОДИН РАЗ при инициализации пакета в функции init().
//
// Структура (порядок важен — LLM придаёт больший вес началу контекста):
//  1. system_prompt.md  — поведение, тон, язык, формат, запреты
//  2. context.md        — бизнес-логика: роли, процессы, термины платформы
//  3. navigations.md    — карта разделов платформы с уровнями доступа
var systemInstruction string

func init() {
	sysPrompt, ok1 := loadDataFile("system_prompt.md")
	ctx, ok2 := loadDataFile("context.md")
	nav, ok3 := loadDataFile("navigations.md")

	// Все три файла отсутствуют → используем дефолтный промпт.
	// Сервис продолжает работать, но с минимальным контекстом.
	if !ok1 && !ok2 && !ok3 {
		slog.Warn("assistant: все data-файлы отсутствуют, используется дефолтный промпт")
		systemInstruction = defaultSystemInstruction
		return
	}

	var b strings.Builder

	// Блок 1: поведенческие инструкции (system_prompt.md).
	// Идут первыми — максимальный приоритет в window attention LLM.
	if ok1 {
		b.WriteString(sysPrompt)
	} else {
		slog.Warn("assistant: system_prompt.md не найден, используется встроенный дефолт")
		b.WriteString(defaultSystemInstruction)
	}

	// Блок 2: бизнес-логика и контекст (context.md).
	if ok2 {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## БИЗНЕС-ЛОГИКА И КОНТЕКСТ ПЛАТФОРМЫ\n\n")
		b.WriteString(ctx)
	} else {
		slog.Warn("assistant: context.md не найден, бизнес-контекст не загружен")
	}

	// Блок 3: карта навигации (navigations.md).
	if ok3 {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## КАРТА НАВИГАЦИИ ПЛАТФОРМЫ\n\n")
		b.WriteString(nav)
	} else {
		slog.Warn("assistant: navigations.md не найден, карта навигации не загружена")
	}

	systemInstruction = b.String()
}

// loadDataFile читает файл из встроенного embed.FS.
//
// Возвращает (содержимое, true) при успехе или ("", false) если файл
// отсутствует в FS или повреждён. Ошибка логируется, паники не происходит —
// вызывающий код обязан проверить второй возвращаемый параметр.
func loadDataFile(name string) (string, bool) {
	data, err := dataFS.ReadFile("data/" + name)
	if err != nil {
		slog.Error("assistant: не удалось прочитать data-файл",
			"file", name,
			"err", err,
		)
		return "", false
	}
	return string(data), true
}
