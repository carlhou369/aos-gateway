package chatapi

import (
	"context"
	"errors"
	"fmt"
	"gateway/common"
	"gateway/db"
	"gateway/log"
	"strings"
	"time"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

const waitForRateLimitRetry = time.Second
const RateLimitAnswer = "AI模型正忙，请稍后重试"
const PromtPrefix = "You are a helpful assistant."
const SystemRole = "system"
const UserRole = "user"
const AnswerRole = "assistant"
const MaxMsgLogLength = 20
const MaxPromtLength = 2048

var SystemMessage = openai.ChatCompletionMessage{
	Role:    SystemRole,
	Content: PromtPrefix,
}

var CheckMsgLogDuration = time.Minute * 2
var MaxConversactionSuspend = 60 * 20

type Client struct {
	apiKey        string
	logUpdateTime map[string]int64
	gptClient     *openai.Client
	Avalible      bool
}

func NewClient(apiKey string, ctx context.Context) *Client {
	gptClient := openai.NewClient(apiKey)
	c := &Client{
		apiKey:        apiKey,
		logUpdateTime: make(map[string]int64),
		gptClient:     gptClient,
		Avalible:      true,
	}
	//reset if client not avalible
	go func() {
		ticker := time.NewTimer(waitForRateLimitRetry)
		for {
			<-ticker.C
			c.Avalible = true
		}
	}()
	return c
}

func (c *Client) buildPromt(q *common.Question) []openai.ChatCompletionMessage {
	promtLen := 0
	promt := []openai.ChatCompletionMessage{}
	conversationFrom := time.Now().Unix() - int64(MaxConversactionSuspend)
	msgLog, err := db.GetResentConversation(q.ConversationId, conversationFrom)
	if err != nil {
		log.Warn("GetResentConversation conversationid", q.ConversationId, "from timestamp", conversationFrom, "error", err)
	}
	if err == nil {
		for i := len(msgLog) - 1; i >= 0; i-- {
			msg := msgLog[i]
			userMsg := openai.ChatCompletionMessage{
				Role:    UserRole,
				Content: msg.Prompt,
			}
			apiMsg := openai.ChatCompletionMessage{
				Role:    AnswerRole,
				Content: msg.Text,
			}
			promtLen = promtLen + len(msg.Prompt) + len(msg.Text)
			promt = append(promt, userMsg, apiMsg)
			//pop early msgs if promt over length
			for promtLen > MaxPromtLength {
				if len(promt) <= 2 {
					break
				}
				shortLen := len(promt[0].Content) + len(promt[1].Content)
				promtLen = promtLen - shortLen
				promt = promt[2:]
			}
		}
	}
	promt = append(promt, openai.ChatCompletionMessage{
		Role:    UserRole,
		Content: q.Message,
	})
	return promt
}

func (C *Client) GetAnswer(ctx context.Context, q common.Question) (*common.QA, error) {
	if !C.Avalible {
		qa := common.QA{
			Question:       q,
			AnswerRole:     "",
			Answer:         RateLimitAnswer,
			MessageId:      "",
			ConversationId: q.ConversationId,
			Model:          openai.GPT3Dot5Turbo,
		}
		return &qa, nil
	}
	prompt := C.buildPromt(&q)
	log.Debug("prompt:\n", fmt.Sprintf("%v", prompt))
	resp, err := C.gptClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: prompt,
	})
	if err != nil {
		log.Info(err.Error())
		//rate limit error
		if strings.Contains(err.Error(), "429") {
			//set api client not avaiable and response rate limit
			// C.Avalible = false
			qa := common.QA{
				Question:       q,
				AnswerRole:     "",
				Answer:         RateLimitAnswer,
				MessageId:      "",
				ConversationId: "",
				Model:          resp.Model,
			}
			return &qa, err
		}
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no answer choise")
	}
	if q.ConversationId == "" {
		q.ConversationId = uuid.New().String()
	}
	qa := common.QA{
		Question:       q,
		AnswerRole:     resp.Choices[0].Message.Role,
		Answer:         resp.Choices[0].Message.Content,
		MessageId:      resp.ID,
		ConversationId: q.ConversationId,
		Model:          resp.Model,
	}
	return &qa, nil
}
