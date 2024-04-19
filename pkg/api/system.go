// Package api
/*
 注册代理路由，为了解决部分用户仓库地址访问不了的问题
 注册目录字典路由，返回所有本地存储文件的目录，依赖于fileload实现
*/
package api

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"time"
)

func SystemRouter(router *echo.Group) {
	handler := &system{}
	systemRouter := router.Group("/system")
	systemRouter.GET("/proxy", handler.proxyRequest)
	systemRouter.GET("/directories", handler.getDirectories)
}

type system struct{}

// @Tags system
// @Description "代理请求"
// @Param url query string true "请求url"
// @Success 200 {object} any "成功"
// @Failure 400 {object} i18n.CustomError
// @Router /system/proxy [get]
func (s *system) proxyRequest(c echo.Context) error {
	url := c.QueryParam(consts.QueryParamUrl)
	if url == "" {
		return i18n.NewCustomError(nil, i18n.QueryParamEmptyError, consts.QueryParamUrl)
	}

	url = models.ReplaceGithubProxyUrl(url)
	client := &http.Client{Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}}
	resp, err := client.Get(url)
	if err != nil {
		return i18n.NewCustomError(err, i18n.RequestProxyError)
	}

	if _, err = io.Copy(c.Response().Writer, resp.Body); err != nil {
		return i18n.NewCustomError(err, i18n.RequestProxyError)
	}

	return nil
}

// @Tags system
// @Description "获取所有配置目录"
// @Success 200 {object} map[string]string "成功"
// @Failure 400 {object} i18n.CustomError
// @Router /system/directories [get]
func (s *system) getDirectories(c echo.Context) error {
	return c.JSON(http.StatusOK, fileloader.GetRootDirectories())
}
