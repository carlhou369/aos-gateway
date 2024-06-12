package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInsert(t *testing.T) {
	defer MgoCli.Disconnect(context.Background())
	msg := Message{
		ConversationId: "123",
		MessageId:      "123",
		Prompt:         "test prompt",
		Text:           "test answer",
		StartTime:      time.Now().Unix(),
	}
	err := InsertSingleConversation(msg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	defer MgoCli.Disconnect(context.Background())
	msgLog, err := GetResentConversation("3ad2929f-ff4c-4a28-8c28-d81abe3b7d7d", 1679399242)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%v", msgLog)
}
