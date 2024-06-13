package selfdriving

import (
	"context"
	"encoding/json"
	"fmt"
	"gateway/common"
	"gateway/db"
	"gateway/log"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

const MaxMsgLogLength = 20
const MaxPromtLength = 2048
const MaxTimeOut = 180

var CheckMsgLogDuration = time.Minute * 2
var MaxConversactionSuspend = 60 * 60

const RateLimitAnswer = "AI模型正忙，请稍后重试"

const SystemRole = "system"
const UserRole = "user"
const AnswerRole = "assistant"

const (
	ModelDown int = iota
	ModelAvalible
	ModelBusy
)

type BSRequest struct {
	Content string     `json:"content"`
	History [][]string `json:"history"`
	Model   string     `json:"model,omitempty"`
}

type BSResponse struct {
	ErrCode  int    `json:"errcode"`
	Response string `json:"response"`
	Ret      int    `json:"ret"`
}

type Client struct {
	Status        int
	Url           string
	logUpdateTime map[string]int64
	ModelName     string
}

func NewClient(url, modelName string, ctx context.Context) *Client {
	c := &Client{
		Status:        ModelDown,
		Url:           url,
		logUpdateTime: make(map[string]int64),
		ModelName:     modelName,
	}
	go c.checkHealth()
	return c
}

func (c *Client) checkHealth() {
	for {
		_, err := common.HttpPost(c.Url+"/health", "", 10, nil)
		if err != nil {
			c.Status = ModelDown
		} else {
			if c.Status == ModelDown {
				c.Status = ModelAvalible
			}
		}
		time.Sleep(time.Second * 10)
	}
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

func (c *Client) GetAnswer(ctx context.Context, q common.Question) (*common.QA, error) {
	c.Status = ModelBusy
	defer func() {
		c.Status = ModelAvalible
	}()
	prompt := c.buildPromt(&q)
	log.Debug("self driving model", q.Model, "prompt:\n", fmt.Sprintf("%v", prompt))
	req := openai.ChatCompletionRequest{
		Model:       q.Model,
		Messages:    prompt,
		Temperature: 0,
		TopP:        0,
		N:           0,
		Stream:      false,
		User:        UserRole,
	}
	promptData, err := json.Marshal(&req)
	if err != nil {
		log.Info(err.Error())
		return nil, err
	}
	resp, err := common.HttpPost(c.Url, string(promptData), MaxTimeOut, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "",
	})

	if err != nil {
		log.Info(err.Error())
		return nil, err
	}
	var bsResp BSResponse
	err = json.Unmarshal(resp, &bsResp)
	if err != nil {
		log.Info(err.Error())
		return nil, err
	}
	if q.ConversationId == "" {
		q.ConversationId = uuid.New().String()
	}
	qa := common.QA{
		Question:       q,
		AnswerRole:     "assitant",
		Answer:         bsResp.Response,
		MessageId:      uuid.NewString(),
		ConversationId: q.ConversationId,
		Model:          c.ModelName,
	}
	return &qa, nil
}

// func (c *Client) buildPromt(q *common.Question) BSRequest {
// 	maxLength := MaxPromtLength
// 	if q.Model == "self-driving-v3" {
// 		maxLength = maxLength * 10
// 	}
// 	promtHistoryLen := 0
// 	promt := BSRequest{
// 		Content: q.Message,
// 		History: [][]string{},
// 		//TODO: add model name
// 	}
// 	conversationFrom := time.Now().Unix() - int64(MaxConversactionSuspend)
// 	msgLog, err := db.GetResentConversation(q.ConversationId, conversationFrom)
// 	if err != nil {
// 		log.Warn("GetResentConversation conversationid", q.ConversationId, "from timestamp", conversationFrom, "error", err)
// 	}
// 	if err == nil {
// 		for i := len(msgLog) - 1; i >= 0; i-- {
// 			msg := msgLog[i]
// 			promt.History = append(promt.History, []string{msg.Prompt, msg.Text})
// 			promtHistoryLen = promtHistoryLen + len(msg.Prompt) + len(msg.Text)
// 			//pop early msgs if promt over length
// 			for promtHistoryLen > maxLength {
// 				if len(promt.History) <= 1 {
// 					break
// 				}
// 				shortLen := len(promt.History[0][0]) + len(promt.History[0][1])
// 				promtHistoryLen = promtHistoryLen - shortLen
// 				promt.History = promt.History[1:]
// 			}
// 		}
// 	}
// 	return promt
// }
