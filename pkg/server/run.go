// Package server
/*
 项目启动
*/
package server

import (
	"context"
	"errors"
	"fireboom-server/pkg/api"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/server"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fireboom-server/pkg/vscode"
	"fireboom-server/pkg/websocket"
	"fmt"
	"github.com/labstack/echo/v4/middleware"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	logger   *zap.Logger
	echoRoot *echo.Echo
)

func init() {
	utils.RegisterInitMethod(math.MaxInt, func() {
		logger = zap.L()
		echoRoot = initEchoRoot()
	})
}

func Run(beforeStarted func()) {
	websocket.AddOnFirstStartedHook(func() {
		fmt.Println(string(configs.BannerText.GetFirstCache()))
	})
	initHttpServer(beforeStarted)
}

// GenerateAuthenticationKey 生成认证密钥
func GenerateAuthenticationKey() (err error) {
	if !utils.GetBoolWithLockViper(consts.EnableAuth) {
		return nil
	}

	authText := configs.AuthenticationKeyText
	authenticationKey, _ := authText.Read(authText.Title)
	defer func() {
		if err != nil {
			logger.Error("generate key failed", zap.Error(err))
			return
		}

		utils.SetWithLockViper(utils.NormalizeName(consts.HeaderParamAuthentication), authenticationKey)
		websocket.AddOnFirstStartedHook(func() {
			logger.Info("AuthenticationKey", zap.String(consts.HeaderParamAuthentication, authenticationKey))
		})
	}()
	if utils.GetBoolWithLockViper(consts.RegenerateKey) || authenticationKey == "" {
		authenticationKey = utils.RandStr(32)
		err = authText.Write(authText.Title, fileloader.SystemUser, []byte(authenticationKey))
		return
	}
	return
}

func initHttpServer(beforeStarted func()) {
	if echoRoot == nil {
		return
	}

	e := echoRoot
	if utils.GetBoolWithLockViper(consts.EnableAuth) {
		authText := configs.AuthenticationKeyText
		authenticationKey, _ := authText.Read(authText.Title)
		e.Use(ProductionAuthentication(authenticationKey))
	}
	e.Use(middleware.Recover(), CORS, RequestLoggerWithConfig(), RequestID, GzipWithConfig())
	e.GET("/fb_health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			consts.EnableAuth: utils.GetBoolWithLockViper(consts.EnableAuth),
			consts.DevMode:    utils.GetBoolWithLockViper(consts.DevMode),
			consts.ActiveMode: utils.GetStringWithLockViper(consts.ActiveMode),
		})
	})

	e.HTTPErrorHandler = httpErrorHandler
	e.HideBanner, e.HidePort = true, true
	address := fmt.Sprintf("http://localhost:%s", utils.GetStringWithLockViper(consts.WebPort))
	websocket.AddOnFirstStartedHook(func() { logger.Info("web server started", zap.String("address", address)) })
	e.Server.BaseContext = func(listener net.Listener) context.Context {
		utils.SetWithLockViper(consts.GlobalStartTime, time.Now())
		return context.Background()
	}
	go beforeStarted()
	// 启动服务器
	go func() { _ = e.Start("") }()

	// 等待终止信号
	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	server.Shutdown()
}

func initEchoRoot() *echo.Echo {
	var listener net.Listener
	callbacks := initEmbedModelDatas()
	switch webPort := utils.GetStringWithLockViper(consts.WebPort); webPort {
	case "":
		// 不需要启动服务且无回调直接返回
		if len(callbacks) == 0 {
			return nil
		}
	default:
		// 服务设置了监听端口判断是否已占用
		var err error
		if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", webPort)); err != nil {
			logger.Error("port was used, exit!", zap.String(consts.WebPort, webPort))
			return nil
		}
	}

	e := echo.New()
	e.Listener = listener
	websocket.InitRouter(e)
	registerProfRouters(e)
	registerWebConsoleRouters(e)
	registerGeneratedStaticRouter(e)
	registerEngineForwardRequests(e)

	contextRouter := e.Group(configs.ApplicationData.ContextPath)
	registerContextBaseRouters(contextRouter)
	registerSwaggerRouter(contextRouter)
	api.HomeRouter(contextRouter)
	api.SystemRouter(contextRouter)
	api.EngineRouter(contextRouter)
	vscode.InitRouter(contextRouter)

	rewriteDynamicSwagger()
	for _, callback := range callbacks {
		callback()
	}
	return e
}

// 错误统一处理
// X-FB-Locale头部参数实现每个请求错误的国际化
func httpErrorHandler(err error, c echo.Context) {
	var customErr *i18n.CustomError
	if errors.As(err, &customErr) {
		customErr.ResetMessageWithLocale(c.Request().Header.Get(consts.HeaderParamLocale))
		c.Response().WriteHeader(http.StatusBadRequest)
		enc := json.NewEncoder(c.Response())
		if err = enc.Encode(&customErr); err != nil {
			logger.Error("error encoding customErr", zap.Error(err))
		}
		return
	}

	logger.Warn("not response error with i18n.CustomError", zap.Error(err))
	c.Response().WriteHeader(http.StatusInternalServerError)
	if _, err = c.Response().Write([]byte(err.Error())); err != nil {
		logger.Error("error write unknownErr", zap.Error(err))
	}
}
