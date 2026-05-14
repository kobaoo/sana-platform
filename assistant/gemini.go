package assistant

import (
	"context"
	"strings"
	"time"

	"encore.dev/beta/errs"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Encore загружает GeminiAPIKey из .secrets.local.cue (local) или
// через `encore secret set GeminiAPIKey` (prod).
var secrets struct {
	GeminiAPIKey string
}

const (
	// geminiModelName — Gemini 1.5 Pro с контекстным окном до 2 млн токенов.
	// Алиас "-latest" всегда указывает на последнюю стабильную версию модели.
	geminiModelName = "models/gemini-3.1-flash-lite"

	// maxOutputTokens — жёсткий лимит токенов в ответе.
	// 2048 достаточно для развёрнутых ответов ассистента.
	maxOutputTokens = 2048

	// temperature — степень детерминизма модели.
	// 0.2 → стабильные фактические ответы без галлюцинаций.
	temperature = 0.2

	// httpTimeout — таймаут одного запроса к Gemini.
	// Увеличен до 120с: Pro-модель с большим контекстом отвечает дольше Flash.
	httpTimeout = 120 * time.Second
)

// ════ SAFETY SETTINGS ════

// defaultSafetySettings — фильтры вредоносного контента.
// BLOCK_LOW_AND_ABOVE — корпоративный стандарт.
// BLOCK_MEDIUM_AND_ABOVE для DANGEROUS_CONTENT — иначе блокируются легитимные
// запросы вида "как удалить сотрудника?" из-за слова "удалить".
var defaultSafetySettings = []*genai.SafetySetting{
	{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockLowAndAbove},
	{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockLowAndAbove},
	{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockLowAndAbove},
	{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockMediumAndAbove},
}

// ════ КЛИЕНТ ════

type geminiClientImpl struct {
	client *genai.Client
}

// newGeminiClient инициализирует официальный Gemini Go SDK клиент.
// Вызывается один раз при старте сервиса через package-level var.
// Паникует при отсутствии API-ключа или ошибке инициализации — fast-fail до приёма трафика.
func newGeminiClient() *geminiClientImpl {
	if secrets.GeminiAPIKey == "" {
		panic("assistant: GeminiAPIKey secret не задан; " +
			"запустите `encore secret set GeminiAPIKey` или заполните .secrets.local.cue")
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(secrets.GeminiAPIKey))
	if err != nil {
		panic("assistant: не удалось создать Gemini клиент: " + err.Error())
	}

	return &geminiClientImpl{client: client}
}

// ════ POST-PROCESSING ════

// fillerPrefixes — шаблонные вступительные фразы, которые LLM генерирует
// несмотря на инструкцию их избегать. Обрезаем программно как страховку.
// Покрывает русский, казахский и английский языки.
var fillerPrefixes = []string{
	// Русский
	"Конечно!", "Конечно,", "Конечно.",
	"Отличный вопрос!", "Хороший вопрос!",
	"Я рад помочь", "Я готов помочь",
	"Безусловно,", "Безусловно!", "Безусловно.",
	"Разумеется,", "Разумеется!", "Разумеется.",
	"Само собой,", "Само собой.",
	"Понял вас.", "Понял,", "Понял.",
	"Спасибо за вопрос.", "Спасибо за ваш вопрос.",
	// Казахский
	"Әрине,", "Әрине!", "Сізге көмектесемін.",
	// Английский
	"Certainly!", "Certainly,",
	"Of course!", "Of course,",
	"Sure!", "Sure,",
	"Great question!", "Good question!",
	"Happy to help", "I'd be happy to help",
	"Absolutely!", "Absolutely,",
}

// stripFillerOpener удаляет шаблонные вступления из начала ответа LLM.
// Программная страховка: даже если модель нарушает инструкцию
// "не использовать вступления", ответ будет чистым.
func stripFillerOpener(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range fillerPrefixes {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimLeft(s[len(prefix):], " \t\n.,!"))
			break
		}
	}
	return s
}

// ════ ОСНОВНОЙ МЕТОД ════

// chat отправляет запрос в Gemini через официальный Go SDK.
//
// Возвращает три значения:
//   - reply string  — текст ответа (после stripFillerOpener)
//   - blocked bool  — true если Safety filters заблокировали ответ (не ошибка).
//     reply содержит дружественное сообщение. Фронтенд показывает предупреждение,
//     не красную ошибку.
//   - err error     — инфраструктурная проблема (сеть, API-ключ, decode).
//     Фронтенд показывает "попробуйте позже".
func (c *geminiClientImpl) chat(ctx context.Context, systemInstr, userMessage string) (reply string, blocked bool, err error) {
	// Создаём экземпляр модели для каждого запроса.
	// GenerativeModel — лёгкий объект без сетевого состояния; создание дёшево.
	// Это позволяет безопасно использовать одного *genai.Client из нескольких горутин.
	model := c.client.GenerativeModel(geminiModelName)
	model.SetTemperature(float32(temperature))
	model.SetMaxOutputTokens(int32(maxOutputTokens))
	model.SafetySettings = defaultSafetySettings

	// System instruction передаётся как отдельное поле модели — не часть диалога.
	// Модель воспринимает его как «кем я являюсь и что знаю».
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemInstr)},
	}

	// Добавляем таймаут к входящему контексту.
	// 120s даёт Pro-модели достаточно времени для обработки больших документов.
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	resp, err := model.GenerateContent(ctx, genai.Text(userMessage))
	if err != nil {
		return "", false, errs.B().
			Code(errs.Unavailable).
			Msg("Ассистент временно недоступен. Попробуйте повторить запрос через несколько секунд.").
			Cause(err).
			Err()
	}

	// Блокировка на уровне prompt (Safety filters отклонили сам запрос).
	if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != genai.BlockReasonUnspecified {
		return "Запрос заблокирован фильтрами безопасности. Пожалуйста, переформулируйте вопрос.", true, nil
	}

	if len(resp.Candidates) == 0 {
		return "", false, errs.B().Code(errs.Internal).Msg("empty response from gemini").Err()
	}

	candidate := resp.Candidates[0]

	// Блокировка на уровне candidate (Safety filters остановили генерацию).
	if candidate.FinishReason == genai.FinishReasonSafety {
		return "Ответ заблокирован фильтрами безопасности. Пожалуйста, переформулируйте вопрос.", true, nil
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", false, errs.B().Code(errs.Internal).Msg("empty parts in gemini response").Err()
	}

	// SDK возвращает Part как интерфейс — приводим к конкретному типу genai.Text.
	text, ok := candidate.Content.Parts[0].(genai.Text)
	if !ok {
		return "", false, errs.B().Code(errs.Internal).Msg("unexpected response type from gemini").Err()
	}

	return stripFillerOpener(string(text)), false, nil
}
