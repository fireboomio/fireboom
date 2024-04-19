// Package websocket
/*
 webContainerHookProxy的websocket实现
*/
package websocket

import (
	"fireboom-server/pkg/common/configs"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"strings"
)

const (
	hookProxyChannel configs.WsChannel = "hookProxy"
	requestEvent     configs.WsEvent   = "request"
	dataEvent        configs.WsEvent   = "data"
	errorEvent       configs.WsEvent   = "error"
)

type hookProxyResult struct {
	Url       string `json:"url"`
	Method    string `json:"method"`
	Query     string `json:"query"`
	Body      string `json:"body"`
	RequestID string `json:"requestId"`
}

var (
	hookProxyEventHandlerMap = make(map[configs.WsEvent]func(body *configs.WsMsgBody))
	hookProxyDataChanMap     = make(map[string]chan interface{})
	hookProxyErrorChanMap    = make(map[string]chan error)
)

func (ws *wsCoreServer) webContainerHookProxyHandler(c echo.Context) error {
	request := c.Request()
	requestId := request.Header.Get(echo.HeaderXRequestID)
	body, _ := io.ReadAll(request.Body)
	result := &configs.WsMsgBody{
		Channel: hookProxyChannel,
		Event:   requestEvent,
		Data: &hookProxyResult{
			Url:       strings.ReplaceAll(c.Request().URL.Path, "/ws", ""),
			Method:    request.Method,
			Query:     request.URL.RawQuery,
			Body:      string(body),
			RequestID: requestId,
		},
	}

	content, _ := json.Marshal(result)
	_, _ = ws.core.Write(content)

	// 创建一个channel 用来作为结束代理请求用的
	dataChan := make(chan interface{})
	errorChan := make(chan error)
	hookProxyDataChanMap[requestId] = dataChan
	hookProxyErrorChanMap[requestId] = errorChan
	defer func() {
		close(dataChan)
		close(errorChan)
		delete(hookProxyDataChanMap, requestId)
		delete(hookProxyErrorChanMap, requestId)
	}()

	select {
	case data := <-dataChan:
		return c.JSON(http.StatusOK, data)
	case err := <-errorChan:
		return c.JSON(http.StatusInternalServerError, err)
	}
}

func init() {
	hookProxyEventHandlerMap[dataEvent] = func(msg *configs.WsMsgBody) {
		if msg.Data == nil {
			return
		}

		hookProxyDataChanMap[msg.RequestID] <- msg.Data.(map[string]any)["result"]
	}

	hookProxyEventHandlerMap[errorEvent] = func(msg *configs.WsMsgBody) {
		hookProxyErrorChanMap[msg.RequestID] <- fmt.Errorf("%#v", msg.Error)
	}

	configs.WsMsgHandlerMap[hookProxyChannel] = func(msg *configs.WsMsgBody) any {
		if msg.RequestID == "" {
			return nil
		}

		if handler, ok := hookProxyEventHandlerMap[msg.Event]; ok {
			handler(msg)
		}

		return nil
	}
}
