// Package api
/*
 在基础路由上进行扩展
 注册graphql文本，proxy,function定义的路由
 注册开放api接口，角色绑定/解绑，角色权限查询，可以在admin项目中使用
*/
package api

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/labstack/echo/v4"
	"github.com/subosito/gotenv"
	"net/http"
)

func EnvExtraRouter(_, envRouter *echo.Group, baseHandler *base.Handler[gotenv.Env], modelRoot *fileloader.Model[gotenv.Env]) {
	handler := &env{
		modelRoot.GetModelName(),
		modelRoot,
		baseHandler,
	}
	envRouter.GET("/getEnvValue/:key", handler.getEnvValue)
}

type env struct {
	modelName   string
	modelRoot   *fileloader.Model[gotenv.Env]
	baseHandler *base.Handler[gotenv.Env]
}

// @Tags env
// @Description "GetEnvValue"
// @Param key path string true "key"
// @Success 200 {string} string "OK"
// @Router /env/getEnvValue/{key} [get]
func (e *env) getEnvValue(c echo.Context) (err error) {
	key, err := e.baseHandler.GetPathParam(c, "key")
	if err != nil {
		return
	}

	return c.String(http.StatusOK, (*e.modelRoot.FirstData())[key])
}
