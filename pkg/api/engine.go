// Package api
/*
 注册引擎相关的路由，包括重启引擎及swagger文档
 飞布提供了两份swagger文档，这里是引擎即9991端口的文档
*/
package api

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"github.com/labstack/echo/v4"
	"net/http"
)

func EngineRouter(contextRouter *echo.Group) {
	handler := &engine{}
	engineRouter := contextRouter.Group("/engine")
	engineRouter.GET("/restart", handler.restart)
	if utils.GetBoolWithLockViper(consts.EnableSwagger) {
		engineRouter.GET("/swagger", handler.getSwaggerJsonFile)
	}
}

type engine struct{}

// @Tags engine
// @Description "引擎swagger.json"
// @Success 200 "成功"
// @Router /engine/swagger [get]
func (s *engine) getSwaggerJsonFile(c echo.Context) error {
	base.SetHeaderCacheControlNoCache(c)
	return c.File(build.GeneratedSwaggerText.GetPath(build.GeneratedSwaggerText.Title))
}

// @Tags engine
// @Description "引擎重启"
// @Success 200 "成功"
// @Router /engine/restart [get]
func (s *engine) restart(c echo.Context) error {
	if utils.GetBoolWithLockViper(consts.DevMode) {
		go utils.BuildAndStart()
	}
	return c.NoContent(http.StatusOK)
}
