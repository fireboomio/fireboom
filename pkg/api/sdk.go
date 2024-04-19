// Package api
/*
 在基础路由上进行扩展
 注册服务端钩子查询，打包下载钩子生成目录路由
*/
package api

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/labstack/echo/v4"
	"net/http"
)

func SdkRouter(_, sdkRouter *echo.Group, baseHandler *base.Handler[models.Sdk], modelRoot *fileloader.Model[models.Sdk]) {
	handler := &sdk{baseHandler, modelRoot, modelRoot.GetModelName()}
	sdkRouter.GET("/enabledServer", handler.enabledServer)
	sdkRouter.GET("/downloadOutput"+base.DataNamePath, handler.downloadOutput)
}

type sdk struct {
	baseHandler *base.Handler[models.Sdk]
	modelRoot   *fileloader.Model[models.Sdk]
	modelName   string
}

// @Tags sdk
// @Description "开启的服务端钩子"
// @Success 200 {object} models.Sdk "OK"
// @Router /sdk/enabledServer [get]
func (d *sdk) enabledServer(c echo.Context) error {
	return c.JSON(http.StatusOK, models.GetEnabledServerSdk())
}

// @Description "下载生成output压缩包"
// @Param dataName path string true "dataName"
// @Success 200 "下载成功"
// @Failure 400 {object} i18n.CustomError
// @Router /sdk/downloadOutput/{dataName} [get]
func (d *sdk) downloadOutput(c echo.Context) error {
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return err
	}

	zipBuffer, err := utils.ZipFilesWithBuffer([]string{data.OutputPath}, true)
	if err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.FileZipError)
	}

	base.SetHeaderContentDisposition(c, data.Name+utils.ExtensionZip)
	return c.Stream(http.StatusOK, "application/zip", zipBuffer)
}
