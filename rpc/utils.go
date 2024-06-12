package rpc

import (
	"gateway/common"
	"io/ioutil"
	"net/url"
	"strings"
)

func post(relay_url string, qu common.Question) ([]byte, error) {
	postData := url.Values{}
	postData.Add("message", qu.Message)
	postData.Add("messageId", qu.MessageId)
	postData.Add("conversationId", qu.ConversationId)
	resp, err := client.Post(relay_url, "application/x-www-form-urlencoded", strings.NewReader(postData.Encode()))
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, err
}

func get(relay_url string) ([]byte, error) {
	resp, err := client.Get(relay_url)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, err
}
