// Package api
/*
 在基础路由上进行扩展
 注册存储客户端相关路由，在web界面的存储功能中使用
*/
package api

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/labstack/echo/v4"
	"net/http"
)

func StorageExtraRouter(rootRouter, storageRouter *echo.Group, baseHandler *base.Handler[models.Storage], modelRoot *fileloader.Model[models.Storage]) {
	handler := &storage{modelRoot.GetModelName(), modelRoot, baseHandler, models.ClientCache}
	storageRouter.GET(hookOptionsWithDataNamePath, handler.getHookOptions)

	clientRouter := rootRouter.Group("/storageClient")
	clientRouter.POST("/ping", handler.ping)
	clientRouter.POST(base.DataNamePath+"/mkdir", handler.mkdir)
	clientRouter.POST(base.DataNamePath+"/upload", handler.upload)
	clientRouter.POST(base.DataNamePath+"/remove", handler.remove)
	clientRouter.POST(base.DataNamePath+"/rename", handler.rename)
	clientRouter.GET(base.DataNamePath+"/list", handler.list)
	clientRouter.GET(base.DataNamePath+"/detail", handler.detail)
	clientRouter.GET(base.DataNamePath+"/download", handler.download)
}

type storage struct {
	modelName   string
	modelRoot   *fileloader.Model[models.Storage]
	baseHandler *base.Handler[models.Storage]
	clientCache *models.StorageClientCache
}

// @Tags storage
// @Description "ping"
// @Param data body models.Storage true "data"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/ping [post]
func (s *storage) ping(c echo.Context) (err error) {
	data, err := s.baseHandler.BindBodyParam(c)
	if err != nil {
		return
	}

	err = s.clientCache.Ping(c.Request().Context(), data)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StoragePingError)
		return
	}

	return
}

// @Tags storage
// @Description "创建目录"
// @Param dataName path string true "dataName"
// @Param dirname query string true "dirname"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/mkdir [post]
func (s *storage) mkdir(c echo.Context) (err error) {
	data, dirname, err := s.getStorageAndQueryParam(c, consts.QueryParamDirname, true)
	if err != nil {
		return
	}

	err = s.clientCache.PutDirObject(c.Request().Context(), data, dirname)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageMkdirError)
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags storage
// @Description "上传"
// @Param dataName path string true "dataName"
// @Param dirname query string true "dirname"
// @Param file formData file true "file"
// @Success 200  "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/upload [post]
func (s *storage) upload(c echo.Context) (err error) {
	file, err := s.baseHandler.GetFormParamFile(c)
	if err != nil {
		return
	}

	data, dirname, err := s.getStorageAndQueryParam(c, consts.QueryParamDirname, false)
	if err != nil {
		return
	}

	err = s.clientCache.PutObject(c.Request().Context(), data, dirname, file)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageTouchError)
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags storage
// @Description "移除"
// @Param dataName path string true "dataName"
// @Param filename query string true "filename"
// @Success 200  "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/remove [post]
func (s *storage) remove(c echo.Context) (err error) {
	data, filename, err := s.getStorageAndQueryParam(c, consts.QueryParamFilename, true)
	if err != nil {
		return
	}

	err = s.clientCache.RemoveObjects(c.Request().Context(), data, filename)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageRemoveError)
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags storage
// @Description "重命名"
// @Param dataName path string true "dataName"
// @Param data body fileloader.DataMutation true "DataMutation"
// @Success 200  "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/rename [post]
func (s *storage) rename(c echo.Context) (err error) {
	modify, err := s.baseHandler.BindModify(c)
	if err != nil {
		return
	}

	data, err := s.baseHandler.GetOneByDataName(c)
	if err != nil {
		return
	}

	err = s.clientCache.RenameObject(c.Request().Context(), data, modify)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageRenameError)
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags storage
// @Description "存储列表"
// @Param dataName path string true "dataName"
// @Param dirname query string true "dirname"
// @Success 200 {object} []models.StorageFile "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/list [get]
func (s *storage) list(c echo.Context) (err error) {
	data, dirname, err := s.getStorageAndQueryParam(c, consts.QueryParamDirname, false)
	if err != nil {
		return
	}

	files, err := s.clientCache.ListObjects(c.Request().Context(), data, dirname)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageListError)
		return
	}

	return c.JSON(http.StatusOK, files)
}

// @Tags storage
// @Description "详情"
// @Param dataName path string true "dataName"
// @Param filename query string true "filename"
// @Success 200 {object} models.StorageFile "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/detail [get]
func (s *storage) detail(c echo.Context) (err error) {
	data, filename, err := s.getStorageAndQueryParam(c, consts.QueryParamFilename, true)
	if err != nil {
		return
	}

	file, err := s.clientCache.StatObject(c.Request().Context(), data, filename)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageListError)
		return
	}

	return c.JSON(http.StatusOK, file)
}

// @Tags storage
// @Description "下载"
// @Param dataName path string true "dataName"
// @Param filename query string true "filename"
// @Success 200 {object} []models.StorageFile "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /storageClient/{dataName}/download [get]
func (s *storage) download(c echo.Context) (err error) {
	data, filename, err := s.getStorageAndQueryParam(c, consts.QueryParamFilename, true)
	if err != nil {
		return
	}

	reader, err := s.clientCache.GetObjectReader(c.Request().Context(), data, filename)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(s.modelName, err, i18n.StorageDownloadError)
		return
	}

	base.SetHeaderContentDisposition(c, filename)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, reader)
}

func (s *storage) getStorageAndQueryParam(c echo.Context, name string, throwEmptyError bool) (data *models.Storage, value string, err error) {
	value, err = s.baseHandler.GetQueryParam(c, name)
	if err != nil && throwEmptyError {
		return
	}

	data, err = s.baseHandler.GetOneByDataName(c)
	return
}

// @Tags storage
// @Description "getHookOptions"
// @Param dataName path string true "dataName"
// @Success 200 {object} map[string]models.HookOptions "hook配置"
// @Failure 400 {object} i18n.CustomError
// @Router /storage/hookOptions/{dataName} [get]
func (s *storage) getHookOptions(c echo.Context) error {
	data, err := s.baseHandler.GetOneByDataName(c)
	if err != nil {
		return nil
	}

	hookOptions := make(map[string]models.HookOptions)
	for profile := range data.UploadProfiles {
		hookOptions[profile] = models.GetStorageProfileHookOptions(data.Name, profile)
	}
	return c.JSON(http.StatusOK, hookOptions)
}
