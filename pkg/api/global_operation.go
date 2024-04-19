// Package api
/*
 在基础路由上进行扩展
 注册全局钩子相关的配置路由
*/
package api

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/labstack/echo/v4"
	"net/http"
)

func GlobalOperationExtraRouter(_, globalOperationRouter *echo.Group, baseHandler *base.Handler[models.GlobalOperation], modelRoot *fileloader.Model[models.GlobalOperation]) {
	handler := &globalOperation{modelRoot.GetModelName(), modelRoot, baseHandler}
	globalOperationRouter.GET("/httpTransportHookOptions", handler.httpTransportHookOptions)
	globalOperationRouter.GET("/authenticationHookOptions", handler.authenticationHookOptions)
}

type globalOperation struct {
	modelName   string
	modelRoot   *fileloader.Model[models.GlobalOperation]
	baseHandler *base.Handler[models.GlobalOperation]
}

// @Tags globalOperation
// @Description "httpTransportHookOptions"
// @Success 200 {object} models.HookOptions "httpTransportHookOptions配置"
// @Failure 400 {object} i18n.CustomError
// @Router /globalOperation/httpTransportHookOptions [get]
func (o *globalOperation) httpTransportHookOptions(c echo.Context) error {
	return c.JSON(http.StatusOK, models.GetHttpTransportHookOptions())
}

// @Tags globalOperation
// @Description "authenticationHookOptions"
// @Success 200 {object} models.HookOptions "authenticationHookOptions配置"
// @Failure 400 {object} i18n.CustomError
// @Router /globalOperation/authenticationHookOptions [get]
func (o *globalOperation) authenticationHookOptions(c echo.Context) error {
	return c.JSON(http.StatusOK, models.GetAuthenticationHookOptions())
}
