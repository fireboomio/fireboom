// Package base
/*
 与fileloader.Model结合使用
 提供基础的数据操作方法，增删改查及批量操作，带锁变更，导入导出
 通过范型约束类型使得所有方法可以复用
*/
package base

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"golang.org/x/exp/slices"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
)

type Handler[T any] struct {
	modelRoot *fileloader.Model[T]
	modelName string
}

func NewBaseHandler[T any](modelRoot *fileloader.Model[T]) *Handler[T] {
	return &Handler[T]{modelRoot: modelRoot, modelName: modelRoot.GetModelName()}
}

// @Description "新增数据"
// @Param data body any true "#/definitions/$modelName$"
// @Success 200 "新增成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$ [post]
func (h *Handler[T]) insert(c echo.Context) error {
	bodyBytes, user, err := h.GetUserAndBody(c)
	if err != nil {
		return err
	}

	if _, err = h.modelRoot.Insert(bodyBytes, user); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataInsertError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "批量新增"
// @Param data body []any true "#/definitions/$modelName$"
// @Param overwrite query bool false "是否覆盖已存在"
// @Success 200 {object} []fileloader.DataBatchResult "批量新增结果"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/batch [post]
func (h *Handler[T]) insertBatch(c echo.Context) error {
	bodyBytes, user, err := h.GetUserAndBody(c)
	if err != nil {
		return err
	}

	batchResult, err := h.modelRoot.InsertBatch(bodyBytes, user, cast.ToBool(c.QueryParam(consts.QueryParamOverwrite)))
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataBatchInsertError)
	}

	return c.JSON(http.StatusOK, batchResult)
}

// @Description "删除"
// @Param dataName path string true "数据名称"
// @Success 200 "删除成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/{dataName} [delete]
func (h *Handler[T]) deleteByDataName(c echo.Context) error {
	dataName, user, err := h.GetPathDataNameAndUser(c)

	if err = h.modelRoot.DeleteByDataName(dataName, user); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataDeleteError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "批量删除"
// @Param dataNames query []string true "数据名称"
// @Success 200 "批量删除成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/batch [delete]
func (h *Handler[T]) deleteBatchByDataNames(c echo.Context) error {
	dataNames := h.GetQueryParamDataNames(c)
	if len(dataNames) == 0 {
		return i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.QueryParamEmptyError, consts.QueryParamDataNames)
	}

	if err := h.modelRoot.DeleteBatchByDataNames(dataNames, h.GetUser(c)); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataBatchDeleteError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "更新"
// @Param data body any true "#/definitions/$modelName$"
// @Param watchAction query string false "操作名称[监听子属性使用]"
// @Success 200 "更新成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$ [put]
func (h *Handler[T]) updateByDataName(c echo.Context) error {
	bodyBytes, user, err := h.GetUserAndBody(c)
	if err != nil {
		return err
	}

	watchAction := h.getQueryParamWatchAction(c)
	if _, err = h.modelRoot.UpdateByDataName(bodyBytes, user, watchAction...); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataUpdateError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "批量更新"
// @Param data body []any true "#/definitions/$modelName$"
// @Param watchAction query string false "操作名称[监听子属性使用]"
// @Success 200 "批量更新成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/batch [put]
func (h *Handler[T]) updateBatch(c echo.Context) (err error) {
	bodyBytes, user, err := h.GetUserAndBody(c)
	if err != nil {
		return err
	}

	watchAction := h.getQueryParamWatchAction(c)
	err = h.modelRoot.UpdateBatch(bodyBytes, user, watchAction...)
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataBatchUpdateError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "查询"
// @Param dataName path string true "model名称"
// @Success 200 {object} any "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/{dataName} [get]
func (h *Handler[T]) getByDataName(c echo.Context) error {
	data, err := h.GetOneByDataName(c)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, data)
}

// @Description "带锁查询"
// @Param dataName path string true "model名称"
// @Success 200 {object} fileloader.DataWithLockUser_data "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/withLockUser/{dataName} [get]
func (h *Handler[T]) getWithLockUserByDataName(c echo.Context) error {
	dataName, err := h.GetPathParamDataName(c)
	if err != nil {
		return err
	}

	data, err := h.modelRoot.GetWithLockUserByDataName(dataName)
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataSelectError)
	}

	return c.JSON(http.StatusOK, data)
}

// @Description "查询singleData"
// @Success 200 {object} any "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/single [get]
func (h *Handler[T]) getSingleData(c echo.Context) error {
	data := h.modelRoot.FirstData()
	if data == nil {
		return i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.DataNotExistsError)
	}

	return c.JSON(http.StatusOK, data)
}

// @Description "带锁查询singleData"
// @Success 200 {object} fileloader.DataWithLockUser_data "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/withLockUser [get]
func (h *Handler[T]) getSingleDataWithLockUser(c echo.Context) error {
	data := h.modelRoot.FirstData()
	if data == nil {
		return i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.DataNotExistsError)
	}

	result, err := h.modelRoot.GetWithLockUserByDataName(h.modelRoot.GetDataName(data))
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataSelectError)
	}

	return c.JSON(http.StatusOK, result)
}

// @Description "查询列表"
// @Param dataNames query []string false "model名称"
// @Success 200 {object} []any "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$ [get]
func (h *Handler[T]) list(c echo.Context) error {
	dataNames := h.GetQueryParamDataNames(c)
	if len(dataNames) == 0 {
		return c.JSON(http.StatusOK, h.modelRoot.List())
	}

	return c.JSON(http.StatusOK, h.modelRoot.ListByDataNames(dataNames))
}

// @Description "查询树状图"
// @Success 200 {object} []fileloader.DataTree "查询成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/tree [get]
func (h *Handler[T]) tree(c echo.Context) error {
	data, err := h.modelRoot.GetDataTrees()
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataSelectError)
	}

	return c.JSON(http.StatusOK, data)
}

// @Description "复制"
// @Param data body fileloader.DataMutation true "请求数据"
// @Success 200 "复制成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/copy [post]
func (h *Handler[T]) copyByDataName(c echo.Context) error {
	modify, err := h.BindModify(c)
	if err != nil {
		return err
	}

	if err = h.modelRoot.CopyByDataName(modify); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataCopyError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "重命名"
// @Param data body fileloader.DataMutation true "请求数据"
// @Success 200 "重命名成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/rename [post]
func (h *Handler[T]) renameByDataName(c echo.Context) error {
	modify, err := h.BindModify(c)
	if err != nil {
		return err
	}

	if err = h.modelRoot.RenameByDataName(modify); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataRenameError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "重命名[ByParentDataName]"
// @Param data body fileloader.DataMutation true "请求数据"
// @Success 200 "重命名成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/renameParent [post]
func (h *Handler[T]) renameByParentDataName(c echo.Context) error {
	modify, err := h.BindModify(c)
	if err != nil {
		return err
	}

	repeatData, err := h.modelRoot.RenameByParentDataName(modify)
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataRenameError)
	}

	return c.JSON(http.StatusOK, repeatData)
}

// @Description "删除[ByParentDataName]"
// @Param dataName path string true "数据名称"
// @Success 200 "删除成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/deleteParent/{dataName} [delete]
func (h *Handler[T]) deleteByParentDataName(c echo.Context) error {
	dataName, user, err := h.GetPathDataNameAndUser(c)
	if err != nil {
		return err
	}

	if err = h.modelRoot.DeleteByParentDataName(dataName, user); err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataDeleteError)
	}

	return c.NoContent(http.StatusOK)
}

// @Description "导入"
// @Accept multipart/form-data
// @Param file formData file true "文件"
// @Success 200 "导入成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/import [post]
func (h *Handler[T]) importData(c echo.Context) error {
	if utils.InvokeFunctionLimit(consts.LicenseImport) {
		return c.NoContent(http.StatusMethodNotAllowed)
	}

	file, err := h.GetFormParamFile(c)
	if err != nil {
		return err
	}

	filterRegexps := []*regexp.Regexp{h.modelRoot.GetPathRegexp()}
	for _, item := range h.modelRoot.GetTextItems() {
		itemRegexp := item.GetPathRegexp()
		if itemRegexp == nil || slices.ContainsFunc(filterRegexps, func(item *regexp.Regexp) bool { return item.String() == itemRegexp.String() }) {
			continue
		}

		filterRegexps = append(filterRegexps, itemRegexp)
	}

	ignored, existed := make([]string, 0), make([]string, 0)
	imported, err := utils.UnzipMultipartFile(file, func(path string) bool {
		if !slices.ContainsFunc(filterRegexps, func(item *regexp.Regexp) bool { return item.MatchString(path) }) {
			ignored = append(ignored, path)
			return false
		}

		if !utils.NotExistFile(path) {
			existed = append(existed, path)
			return false
		}

		return true
	})
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.FileUnZipError)
	}

	return c.JSON(http.StatusOK, map[string][]string{"imported": imported, "existed": existed, "ignored": ignored})
}

// @Description "导出"
// @Param dataNames query []string false "dataNames"
// @Success 200 "导出成功"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/export [get]
func (h *Handler[T]) exportData(c echo.Context) error {
	if utils.InvokeFunctionLimit(consts.LicenseImport) {
		return c.NoContent(http.StatusMethodNotAllowed)
	}

	dataNames := h.GetQueryParamDataNames(c)
	exportData := h.modelRoot.ListByCondition(func(item *T) bool {
		return dataNames == nil || slices.Contains(dataNames, h.modelRoot.GetDataName(item))
	})
	var zipFilenames []string
	for _, data := range exportData {
		dataName := h.modelRoot.GetDataName(data)
		zipFilenames = append(zipFilenames, h.modelRoot.GetPath(dataName))
		for _, text := range h.modelRoot.GetTextItems() {
			if !text.Enabled(dataName) {
				continue
			}

			textPath := text.GetPath(dataName)
			if utils.NotExistFile(textPath) {
				continue
			}

			zipFilenames = append(zipFilenames, textPath)
		}
	}
	zipBuffer, err := utils.ZipFilesWithBuffer(zipFilenames, false)
	if err != nil {
		return i18n.NewCustomErrorWithMode(h.modelName, err, i18n.FileZipError)
	}

	SetHeaderContentDisposition(c, h.modelName+utils.ExtensionZip)
	return c.Stream(http.StatusOK, "application/zip", zipBuffer)
}

func (h *Handler[T]) BindModify(c echo.Context) (modify *fileloader.DataMutation, err error) {
	if err = c.Bind(&modify); err != nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.ParamBindError)
		return
	}

	errBuildFunc := func(name ...string) {
		err = i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.BodyParamEmptyError, strings.Join(name, utils.StringComma))
	}

	if modify == nil || modify.Src == "" || modify.Dst == "" {
		errBuildFunc("src", "dst")
		return
	}

	modify.User = h.GetUser(c)
	return
}

func (h *Handler[T]) GetOneByDataName(c echo.Context) (data *T, err error) {
	dataName, err := h.GetPathParamDataName(c)
	if err != nil {
		return
	}

	data, err = h.modelRoot.GetByDataName(dataName)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.DataSelectError)
		return
	}

	return
}

func (h *Handler[T]) BindBodyParam(c echo.Context) (data *T, err error) {
	if err = c.Bind(&data); err != nil || data == nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.ParamBindError)
		return
	}

	if _, err = h.modelRoot.EmptyDataNameThrowError(data); err != nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.ParamBindError)
		return
	}
	return
}

func (h *Handler[T]) GetPathParam(c echo.Context, name string) (value string, err error) {
	if value = c.Param(name); value == "" {
		err = i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.PathParamEmptyError, name)
		return
	}

	return
}

func (h *Handler[T]) GetPathParamDataName(c echo.Context) (string, error) {
	return h.GetPathParam(c, consts.PathParamDataName)
}

func (h *Handler[T]) GetPathDataNameAndUser(c echo.Context) (dataName, user string, err error) {
	if dataName, err = h.GetPathParamDataName(c); err != nil {
		return
	}

	user = h.GetUser(c)
	return
}

func (h *Handler[T]) GetFormParamFile(c echo.Context) (file *multipart.FileHeader, err error) {
	file, err = c.FormFile(consts.FormParamFile)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.FormParamEmptyError, consts.FormParamFile)
		return
	}

	return
}

func (h *Handler[T]) GetQueryParam(c echo.Context, name string) (value string, err error) {
	if value = c.QueryParam(name); value == "" {
		err = i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.QueryParamEmptyError, name)
		return
	}

	return
}

func (h *Handler[T]) getQueryParams(c echo.Context, name string) []string {
	value := c.QueryParam(name)
	if value == "" {
		return nil
	}

	return strings.Split(value, utils.StringComma)
}

func (h *Handler[T]) GetUserAndBody(c echo.Context) (bodyBytes []byte, user string, err error) {
	if bodyBytes, err = io.ReadAll(c.Request().Body); err != nil {
		err = i18n.NewCustomErrorWithMode(h.modelName, err, i18n.RequestReadBodyError)
		return
	}

	if len(bodyBytes) == 0 {
		err = i18n.NewCustomErrorWithMode(h.modelName, nil, i18n.RequestEmptyBodyError)
		return
	}

	user = h.GetUser(c)
	return
}

func (h *Handler[T]) GetUser(c echo.Context) string {
	return c.Request().Header.Get(consts.HeaderParamUser)
}

func (h *Handler[T]) GetQueryParamDataNames(c echo.Context) []string {
	return h.getQueryParams(c, consts.QueryParamDataNames)
}

func (h *Handler[T]) getQueryParamWatchAction(c echo.Context) []string {
	return h.getQueryParams(c, consts.QueryParamWatchAction)
}

func SetHeaderContentDisposition(c echo.Context, filename string) {
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(consts.AttachmentFilenameFormat, filename))
}

func SetHeaderCacheControlNoCache(c echo.Context) {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
}
