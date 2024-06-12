package rpc

import (
	"context"
	"errors"
	"fmt"
	"gateway/common"
	"gateway/log"
	"time"
)

var (
	ErrModelNotAvalible = errors.New("mode still in use")
	ErrNoFreeModel      = errors.New("no free model")
)

func (s *Service) handleOpenAIQuestion(qu pendingQuestion) error {
	log.Debug("handle openai question")
	//have former url
	if qu.data.OpenAIKey != "" {
		if res := s.queryRelay(&qu); res != nil {
			qu.resp <- *res
			close(qu.resp)
			return nil
		} else {
			s.questionCh <- qu
			return nil
		}
	}
	//find free relay and reply
	for apiKey, client := range s.gptApiClients {
		if !client.Avalible {
			log.Info("client not avalible ", apiKey)
			continue
		}
		qu.data.OpenAIKey = apiKey
		if res := s.queryRelay(&qu); res != nil {
			qu.resp <- *res
			close(qu.resp)
			return nil
		} else {
			s.questionCh <- qu
			return nil
		}
	}
	return nil
}

func (s *Service) queryRelay(qu *pendingQuestion) *RelayResponse {
	apiKey := qu.data.OpenAIKey
	log.Debug("sending to relay apiKey", apiKey, "\n", qu.data)
	qa, err := s.gptGetAnswer(apiKey, qu.data)
	qu.TriedTimes++
	if err != nil || qa == nil {
		log.Warn("relay res err  %v", apiKey)
		qu.data.ConversationId = ""
		qu.data.MessageId = ""
		qu.data.OpenAIKey = ""
		return nil
	}
	//decode relay reponse put into channel
	relayResponse := RelayResponse{
		Url:            apiKey,
		Text:           qa.Answer,
		MessageId:      qa.MessageId,
		ConversationId: qa.ConversationId,
		Model:          "gpt",
	}
	log.Debug(fmt.Sprintf("question: %s \n answer: %s \n model: %s", qu.data.Message, relayResponse.Text, relayResponse.Model))
	return &relayResponse
}

func (s *Service) gptGetAnswer(apiKey string, qu common.Question) (*common.QA, error) {
	cli, ok := s.gptApiClients[apiKey]
	if !ok {
		return nil, errors.New("no client at " + apiKey)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	return cli.GetAnswer(ctx, qu)
}
