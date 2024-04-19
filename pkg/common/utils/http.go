package utils

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"
)

// HttpPost 发送post请求，支持超时设置
func HttpPost(url string, reqBody []byte, headers map[string]string, timeout ...int) (respBody []byte, err error) {
	r, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return
	}

	for k, v := range headers {
		r.Header.Add(k, v)
	}

	client := http.DefaultClient
	if len(timeout) > 0 {
		client.Timeout = time.Duration(timeout[0]) * time.Second
	}

	resp, err := client.Do(r)
	if err != nil {
		return
	}

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New(string(respBody))
		respBody = nil
		return
	}

	return
}
