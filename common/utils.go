package common

import (
	"bytes"
	"gateway/log"
	"io/ioutil"
	"net/http"
	"time"
)

func HttpPost(requrl, body string, timeoutS int, headerMap map[string]string) (payload []byte, err error) {
	req, err := http.NewRequest("POST", requrl, bytes.NewBufferString(body))
	if err != nil {
		log.Error("Failed to new http request:", err.Error())
		return
	}

	for k, v := range headerMap {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: time.Second * time.Duration(timeoutS)}

	resp, err := client.Do(req)
	if resp == nil {
		return
	}

	defer resp.Body.Close()

	payload, err = ioutil.ReadAll(resp.Body)
	return
}
