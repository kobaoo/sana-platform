package assistant

// ChatRequest — тело входящего запроса к эндпоинту POST /assistant/chat.
type ChatRequest struct {
	// Message — вопрос пользователя. Максимум 500 символов Unicode.
	// Пустая строка и строка из пробелов отклоняются с HTTP 400.
	Message string `json:"message"`
}

// ChatResponse — тело ответа эндпоинта POST /assistant/chat.
type ChatResponse struct {
	// Reply — текст ответа от Sana Expert.
	// Всегда на том языке, на котором был задан вопрос.
	// Прошёл post-processing: убраны шаблонные вступления.
	Reply string `json:"reply"`

	// Blocked — true, если ответ был заблокирован фильтрами безопасности Gemini.
	// При Blocked=true Reply содержит дружественное сообщение для пользователя,
	// а не пустую строку. Фронтенд может использовать флаг для особого отображения
	// (например, жёлтый баннер вместо красной ошибки).
	// omitempty позволяет не включать поле в ответ при нормальной работе.
	Blocked bool `json:"blocked,omitempty"`
}
