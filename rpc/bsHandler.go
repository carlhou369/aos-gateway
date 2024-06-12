package rpc

import (
	"context"
	"fmt"
	"gateway/log"
	selfdriving "gateway/self-driving"
	"math/rand"
	"time"
)

func (s *Service) handleBsQuestion(qu pendingQuestion) bool {
	clients := s.bsApiClient[qu.data.Model]
	var client *selfdriving.Client
	timer := time.NewTimer(time.Second * 60)
loop:
	for {
		select {
		case <-timer.C:
			log.Error("timeout no available client for model %s", qu.data.Model)
			qu.resp <- RelayResponse{
				Url:            client.Url,
				Text:           "",
				MessageId:      "",
				ConversationId: "",
				Model:          qu.data.Model,
			}
			close(qu.resp)
			return false
		default:
			idx := rand.Intn(len(clients))
			client = clients[idx]
			if client.Status == Avalible {
				break loop
			}
		}
	}

	qa, err := client.GetAnswer(context.Background(), qu.data)
	if err != nil {
		log.Error("handle bs question error")
		qu.resp <- RelayResponse{
			Url:            client.Url,
			Text:           "",
			MessageId:      "",
			ConversationId: "",
			Model:          qu.data.Model,
		}
		close(qu.resp)
		return false
	}
	res := RelayResponse{
		Url:            client.Url,
		Text:           qa.Answer,
		MessageId:      qa.MessageId,
		ConversationId: qa.ConversationId,
		Model:          qu.data.Model,
	}
	log.Info(fmt.Sprintf("question: %s \n answer: %s \n model: %s", qu.data.Message, res.Text, res.Model))
	qu.resp <- res
	close(qu.resp)
	return false
}
