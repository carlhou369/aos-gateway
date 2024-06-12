package db

type Message struct {
	ConversationId string `json:"conversationId" bson:"conversationId"`
	MessageId      string `json:"messageId" bson:"messageId"`
	Prompt         string `json:"prompt" bson:"prompt"`
	Text           string `json:"text" bson:"text"`
	StartTime      int64  `json:"startTime" bson:"startTime"`
	Model          string `json:"model" bson:"model"`
	Url            string `json:"url" bson:"url"`
}
