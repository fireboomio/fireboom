// Package websocket
/*
 websocket路由注册
 结合configs.WsMsgHandlerMap和msg.channel实现不同功能
*/
package websocket

import (
	"bytes"
	"errors"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"github.com/gorilla/websocket"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"sync"
)

type wsCoreServer struct {
	core *configs.Websocket
}

var (
	logger     *zap.Logger
	wsUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	pingMsgBytes = []byte("ping")
	pongMsgBytes = []byte("pong")
)

func InitRouter(router *echo.Echo) {
	handler := &wsCoreServer{configs.WebsocketInstance}
	router.Any("/ws", handler.open)
	router.Any("/ws/*", handler.webContainerHookProxyHandler)
}

func (ws *wsCoreServer) open(c echo.Context) error {
	wsConn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	conn := &configs.WebsocketConn{Conn: wsConn, Mutex: &sync.Mutex{}}
	ws.core.Conns.Store(conn, true)

	defer ws.close(wsConn)
	var receiveMsg []byte
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			if utils.GetTimeWithLockViper(consts.GlobalStartTime).IsZero() {
				break
			}

			_, receiveMsg, err = conn.Conn.ReadMessage()
			if err != nil {
				var closeError *websocket.CloseError
				if errors.As(err, &closeError) {
					return nil
				}

				logger.Warn("read websocket message failed", zap.Error(err))
				break
			}

			if bytes.Equal(receiveMsg, pingMsgBytes) {
				conn.WriteMessage(pongMsgBytes)
				break
			}

			var msgData *configs.WsMsgBody
			if err = json.Unmarshal(receiveMsg, &msgData); err != nil {
				logger.Warn("unmarshal wsMsgBody failed", zap.Error(err), zap.ByteString("message", receiveMsg))
				break
			}

			handler, ok := configs.WsMsgHandlerMap[msgData.Channel]
			if !ok {
				logger.Warn("unsupported ws channel", zap.Any("channel", msgData.Channel))
				break
			}

			msgData.Data = handler(msgData)
			conn.WriteWsMsgBody(msgData)
		}
	}
}

func (ws *wsCoreServer) close(conn *websocket.Conn) {
	if _, ok := ws.core.Conns.LoadAndDelete(conn); ok {
		_ = conn.Close()
	}
}

func init() {
	utils.RegisterInitMethod(40, func() { logger = zap.L() })
}
