// Package configs
/*
 websocket实现日志记录器writeSyncer接口
 applicationRoot主要是为了验证对properties文件的支持，添加代码级别的配置支持，包括跟路径、认证路由、日志白名单等
 BannerText读取内置的文本，即控制台直接输出的飞布LOGO
 IntrospectText读取内置的文本，用作graphql数据源的内省查询的请求体
*/
package configs

import (
	"fireboom-server/pkg/common/utils"
	"github.com/gorilla/websocket"
	json "github.com/json-iterator/go"
	"sync"
)

type (
	WsMsgBody struct {
		Channel   WsChannel   `json:"channel"`
		Event     WsEvent     `json:"event"`
		Data      interface{} `json:"data,omitempty"`
		Error     interface{} `json:"error,omitempty"`
		RequestID string      `json:"requestId,omitempty"`
	}

	WsChannel string
	WsEvent   string
)

type (
	Websocket struct {
		Conns utils.SyncMap[*WebsocketConn, bool]
	}
	WebsocketConn struct {
		Conn *websocket.Conn
		*sync.Mutex
	}
)

func (ws *Websocket) Sync() error {
	return nil
}

func (ws *Websocket) Write(p []byte) (n int, err error) {
	n = len(p)
	ws.WriteWsMsgBodyForAll(&WsMsgBody{
		Channel: logChannel,
		Event:   PushEvent,
		Data:    string(removeColorText(p)),
	})
	return
}

func (ws *Websocket) WriteWsMsgBodyForAll(body *WsMsgBody) {
	var bodyBytes []byte
	ws.Conns.Range(func(item *WebsocketConn, _ bool) bool {
		if bodyBytes == nil {
			bodyBytes, _ = json.Marshal(body)
		}
		item.WriteMessage(bodyBytes)
		return true
	})
}

func (ws *WebsocketConn) WriteMessage(bodyBytes []byte) {
	ws.Lock()
	defer ws.Unlock()
	_ = ws.Conn.WriteMessage(websocket.TextMessage, bodyBytes)
}

func (ws *WebsocketConn) WriteWsMsgBody(body *WsMsgBody) {
	bodyBytes, _ := json.Marshal(body)
	ws.WriteMessage(bodyBytes)
}

const (
	PushEvent WsEvent = "push"
	PullEvent WsEvent = "pull"
)

var (
	WebsocketInstance *Websocket
	WsMsgHandlerMap   = make(map[WsChannel]func(body *WsMsgBody) any)
)

func init() {
	utils.RegisterInitMethod(12, func() {
		WebsocketInstance = &Websocket{}
		addLoggerWriteSyncer(WebsocketInstance)
	})
}
