package common

type Question struct {
	Message        string `json:"message"`
	MessageId      string `json:"messageId"`
	ConversationId string `json:"conversationId"`
	OpenAIKey      string `json:"openaiKey"`
	Model          string `json:"model"`
}

type QA struct {
	Question       Question `json:"question"`
	AnswerRole     string   `json:"answerRole"`
	Answer         string   `json:"answer"`
	MessageId      string   `json:"messageId"`
	ConversationId string   `json:"conversationId"`
	Model          string   `json:"model"`
}
