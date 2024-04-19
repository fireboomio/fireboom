// Package server
/*
 路由注册
*/
package server

import (
	"fireboom-server/assets"
	"fireboom-server/pkg/api"
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
)

// pprof 性能监控
func registerProfRouters(baseRouter *echo.Echo) {
	if utils.GetBoolWithLockViper(consts.EnableDebugPprof) {
		// https://zhuanlan.zhihu.com/p/387779292
		profGroup := baseRouter.Group("/debug/pprof")
		profGroup.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
		profGroup.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))
		profGroup.GET("/block", echo.WrapHandler(pprof.Handler("block")))
		profGroup.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
		profGroup.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
		profGroup.GET("/mutex", echo.WrapHandler(pprof.Handler("mutex")))
		profGroup.GET("/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))

		profGroup.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
		profGroup.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
		profGroup.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
		profGroup.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	}
}

// 生成sdk的静态路由
func registerGeneratedStaticRouter(baseRouter *echo.Echo) {
	baseRouter.Static("/generated-sdk", "generated-sdk")
}

// 前端资源路由
func registerWebConsoleRouters(baseRouter *echo.Echo) {
	if utils.GetBoolWithLockViper(consts.EnableWebConsole) {
		// er图静态页面
		baseRouter.GET("/d*", echo.WrapHandler(http.StripPrefix("/d", http.FileServer(assets.GetDBMLFileSystem()))))

		// embed front
		baseRouter.GET("/*", echo.WrapHandler(http.FileServer(assets.GetFrontFileSystem())))
		return
	}

	baseRouter.GET("/", func(c echo.Context) error { return c.String(http.StatusNotFound, "web console disabled") })
}

// 转发引擎路由
func registerEngineForwardRequests(baseRouter *echo.Echo) {
	for _, request := range configs.ApplicationData.EngineForwardRequests {
		baseRouter.Any(request, func(c echo.Context) error {
			nodeOptions := configs.GlobalSettingRoot.FirstData().NodeOptions
			if nodeOptions == nil {
				return i18n.NewCustomError(nil, i18n.RequestProxyError)
			}

			forward, err := url.Parse(utils.GetVariableString(nodeOptions.NodeUrl))
			if err != nil {
				return err
			}

			c.Request().Header.Add(consts.HeaderParamTag, utils.RandomIdentifyCode)
			httputil.NewSingleHostReverseProxy(forward).ServeHTTP(c.Response(), c.Request())
			return nil
		})
	}
}

// swagger路由
func registerSwaggerRouter(contextRouter *echo.Group) {
	if utils.GetBoolWithLockViper(consts.EnableSwagger) {
		contextRouter.GET("/swagger/*", echoSwagger.EchoWrapHandler(func(config *echoSwagger.Config) {
			config.InstanceName = cloudInstanceName
		}))
	}
}

// 基于RegisterBaseRouter实现的路由
func registerContextBaseRouters(contextRouter *echo.Group) {
	base.RegisterBaseRouter(contextRouter, models.DatasourceRoot, api.DatasourceExtraRouter)
	base.RegisterBaseRouter(contextRouter, models.OperationRoot, api.OperationExtraRouter)
	base.RegisterBaseRouter(contextRouter, models.StorageRoot, api.StorageExtraRouter)
	base.RegisterBaseRouter(contextRouter, models.SdkRoot, api.SdkRouter)
	base.RegisterBaseRouter(contextRouter, models.AuthenticationRoot)
	base.RegisterBaseRouter(contextRouter, models.RoleRoot)
	base.RegisterBaseRouter(contextRouter, models.GlobalOperationRoot, api.GlobalOperationExtraRouter)

	base.RegisterBaseRouter(contextRouter, configs.GlobalSettingRoot)
	base.RegisterBaseRouter(contextRouter, configs.EnvEffectiveRoot)
}
