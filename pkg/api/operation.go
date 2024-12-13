// Package api
/*
 在基础路由上进行扩展
 注册graphql文本，proxy,function定义的路由
 注册开放api接口，角色绑定/解绑，角色权限查询，可以在admin项目中使用
*/
package api

import (
	"crypto/sha256"
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/engine/directives"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func OperationExtraRouter(_, operationRouter *echo.Group, baseHandler *base.Handler[models.Operation], modelRoot *fileloader.Model[models.Operation]) {
	handler := &operation{
		modelRoot.GetModelName(),
		modelRoot,
		models.RoleRoot,
		models.OperationGraphql,
		models.OperationGraphqlHistory,
		baseHandler,
		make(map[string]string),
	}
	operationRouter.GET("/graphql"+base.DataNamePath, handler.getGraphqlText)
	operationRouter.POST("/graphql"+base.DataNamePath, handler.updateGraphqlText)
	operationRouter.GET("/graphqlHistoryList"+base.DataNamePath, handler.getGraphqlHistoryList)
	operationRouter.GET("/graphqlHistory"+base.DataNamePath, handler.getGraphqlHistoryText)
	operationRouter.POST("/graphqlHistory"+base.DataNamePath, handler.backupGraphqlHistoryText)
	operationRouter.PUT("/graphqlHistory"+base.DataNamePath, handler.rollbackGraphqlHistoryText)
	operationRouter.POST("/function"+base.DataNamePath, handler.updateFunctionText)
	operationRouter.POST("/proxy"+base.DataNamePath, handler.updateProxyText)
	operationRouter.GET("/hookOptions"+base.DataNamePath, handler.getHookOptions)

	operationRouter.POST("/bindRoles", handler.bindRoles)
	base.AddRouterMetas(modelRoot,
		operationRouter.GET("/listPublic", handler.listPublic),
		operationRouter.POST("/listByRole", handler.listByRole),
	)
}

type (
	operation struct {
		modelName          string
		modelRoot          *fileloader.Model[models.Operation]
		roleRoot           *fileloader.Model[models.Role]
		graphqlText        *fileloader.ModelText[models.Operation]
		graphqlHistoryText *fileloader.ModelText[models.Operation]
		baseHandler        *base.Handler[models.Operation]
		graphqlHashMap     map[string]string
	}
	paramQueryRole struct {
		RbacType string `json:"rbacType"`
		RoleCode string `json:"roleCode"`
	}
	paramBindRole struct {
		RbacType       string   `json:"rbacType"`
		RoleCodes      []string `json:"roleCodes"`
		OperationPaths []string `json:"operationPaths"`
	}
)

// @Tags operation
// @Description "GetGraphqlText"
// @Param dataName path string true "dataName"
// @Success 200 {string} string "OK"
// @Router /operation/graphql/{dataName} [get]
func (o *operation) getGraphqlText(c echo.Context) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}

	content, _ := o.graphqlText.Read(dataName)
	return c.String(http.StatusOK, content)
}

// @Tags operation
// @Description "UpdateGraphqlText"
// @Param dataName path string true "dataName"
// @Param data body string true "文本"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/graphql/{dataName} [post]
func (o *operation) updateGraphqlText(c echo.Context) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}

	body, user, err := o.baseHandler.GetUserAndBody(c)
	if err != nil {
		return
	}

	if exist, ok := o.graphqlHashMap[dataName]; ok && exist == fmt.Sprintf("%x", sha256.Sum256(body)) {
		return c.NoContent(http.StatusOK)
	}

	if err = o.graphqlText.Write(dataName, user, body); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.FileWriteError, o.graphqlText.GetPath(dataName))
		return
	}

	o.graphqlHashMap[dataName] = fmt.Sprintf("%x", sha256.Sum256(body))
	return c.NoContent(http.StatusOK)
}

// @Tags operation
// @Description "getGraphqlHistoryList"
// @Param dataName path string true "dataName"
// @Success 200 {object} []string "OK"
// @Router /operation/graphqlHistoryList/{dataName} [get]
func (o *operation) getGraphqlHistoryList(c echo.Context) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}

	historyFileExt := string(o.graphqlHistoryText.Extension)
	historyDirname := strings.TrimSuffix(o.graphqlHistoryText.GetPath(dataName), historyFileExt)
	if utils.NotExistFile(historyDirname) {
		err = i18n.NewCustomErrorWithMode(o.modelName, nil, i18n.DirectoryReadError, historyDirname)
		return
	}

	versionNames := make([]string, 0, 8)
	_ = filepath.Walk(historyDirname, func(path string, info fs.FileInfo, _ error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		basename := filepath.Base(path)
		if strings.HasSuffix(basename, historyFileExt) {
			versionNames = append(versionNames, strings.TrimSuffix(basename, historyFileExt))
		}
		return nil
	})

	return c.JSON(http.StatusOK, versionNames)
}

// @Tags operation
// @Description "GetGraphqlHistoryText"
// @Param dataName path string true "dataName"
// @Param version query string true "version"
// @Success 200 {string} string "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/graphqlHistory/{dataName} [get]
func (o *operation) getGraphqlHistoryText(c echo.Context) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}
	version, err := o.baseHandler.GetQueryParam(c, consts.QueryParamVersion)
	if err != nil {
		return
	}

	content, err := o.graphqlHistoryText.Read(dataName, version)
	if err != nil {
		return
	}
	return c.String(http.StatusOK, content)
}

// @Tags operation
// @Description "backupGraphqlHistoryText"
// @Param dataName path string true "dataName"
// @Param version query string true "version"
// @Param data body string true "文本"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/graphqlHistory/{dataName} [post]
func (o *operation) backupGraphqlHistoryText(c echo.Context) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}
	version, err := o.baseHandler.GetQueryParam(c, consts.QueryParamVersion)
	if err != nil {
		return
	}
	body, user, err := o.baseHandler.GetUserAndBody(c)
	if err != nil {
		return
	}

	if err = o.graphqlHistoryText.Write(dataName, user, body, version); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.FileWriteError, o.graphqlHistoryText.GetPath(dataName, version))
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags operation
// @Description "rollbackGraphqlHistoryText"
// @Param dataName path string true "dataName"
// @Param version query string true "version"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/graphqlHistory/{dataName} [put]
func (o *operation) rollbackGraphqlHistoryText(c echo.Context) (err error) {
	dataName, user, err := o.baseHandler.GetPathDataNameAndUser(c)
	if err != nil {
		return
	}
	version, err := o.baseHandler.GetQueryParam(c, consts.QueryParamVersion)
	if err != nil {
		return
	}

	historyFilepath := o.graphqlHistoryText.GetPath(dataName, version)
	if utils.NotExistFile(historyFilepath) {
		err = i18n.NewCustomErrorWithMode(o.modelName, nil, i18n.FileReadError, historyFilepath)
		return
	}
	if err = o.graphqlText.WriteCustom(dataName, user, func(dstFile *os.File) error {
		srcFile, _err := os.Open(historyFilepath)
		if _err != nil {
			return _err
		}
		defer func() { _ = srcFile.Close() }()
		if _, _err = io.Copy(dstFile, srcFile); _err != nil {
			return _err
		}
		return dstFile.Sync()
	}); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.FileWriteError, o.graphqlHistoryText.GetPath(dataName, version))
		return
	}

	return c.NoContent(http.StatusOK)
}

// @Tags operation
// @Description "updateFunctionText"
// @Param dataName path string true "dataName"
// @Param data body string true "文本"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/function/{dataName} [post]
func (o *operation) updateFunctionText(c echo.Context) (err error) {
	return o.updateOperationExtensionText(c, models.OperationFunction)
}

// @Tags operation
// @Description "updateProxyText"
// @Param dataName path string true "dataName"
// @Param data body string true "文本"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/proxy/{dataName} [post]
func (o *operation) updateProxyText(c echo.Context) (err error) {
	return o.updateOperationExtensionText(c, models.OperationProxy)
}

// @Tags operation
// @Description "ListByRole"
// @Param data body paramQueryRole true "data"
// @Success 200 {object} []any "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/listByRole [post]
func (o *operation) listByRole(c echo.Context) (err error) {
	var param paramQueryRole
	if err = c.Bind(&param); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.ParamBindError)
		return
	}

	if param.RoleCode == "" {
		err = i18n.NewCustomErrorWithMode(o.modelName, nil, i18n.BodyParamEmptyError, "code")
		return
	}

	if param.RbacType == "" {
		param.RbacType = consts.RequireMatchAny
	}

	data := o.modelRoot.ListByCondition(func(item *models.Operation) bool {
		if item.AuthorizationConfig == nil || item.AuthorizationConfig.RoleConfig == nil {
			return false
		}

		var matchRoles []string
		roleConfig := item.AuthorizationConfig.RoleConfig
		switch param.RbacType {
		case consts.RequireMatchAny:
			matchRoles = roleConfig.RequireMatchAny
		case consts.RequireMatchAll:
			matchRoles = roleConfig.RequireMatchAll
		case consts.DenyMatchAny:
			matchRoles = roleConfig.DenyMatchAny
		case consts.DenyMatchAll:
			matchRoles = roleConfig.DenyMatchAll
		}
		return slices.Contains(matchRoles, param.RoleCode)
	})
	return c.JSON(http.StatusOK, data)
}

// @Tags operation
// @Description "ListPublic"
// @Success 200 {object} []any "#/definitions/$modelName$"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/listPublic [get]
func (o *operation) listPublic(c echo.Context) error {
	return c.JSON(http.StatusOK, o.modelRoot.ListByCondition(func(item *models.Operation) bool { return !item.Internal }))
}

// @Tags operation
// @Description "BindRoles"
// @Param data body paramBindRole true "data"
// @Success 200 {object} []string "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/bindRoles [post]
func (o *operation) bindRoles(c echo.Context) (err error) {
	var param paramBindRole
	if err = c.Bind(&param); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.ParamBindError)
		return
	}

	if len(param.OperationPaths) == 0 {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.BodyParamEmptyError, "code,operationPaths")
		return
	}

	if param.RbacType == "" {
		param.RbacType = consts.RequireMatchAny
	}

	updateRoleFunc := directives.FetchUpdateRoleFunc(param.RbacType)
	if updateRoleFunc == nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, nil, i18n.OperationRbacTypeError, param.RbacType)
		return
	}

	operations := o.modelRoot.ListByDataNames(param.OperationPaths)
	if len(operations) == 0 {
		err = i18n.NewCustomErrorWithMode(o.modelName, nil, i18n.DataEmptyListError)
		return
	}

	var children ast.ChildValueList
	for _, code := range param.RoleCodes {
		children = append(children, &ast.ChildValue{Value: &ast.Value{Raw: code, Kind: ast.EnumValue}})
	}
	rbacDirective := &ast.Directive{Name: directives.RbacName, Arguments: ast.ArgumentList{{
		Name: param.RbacType,
		Value: &ast.Value{
			Kind:     ast.ListValue,
			Children: children,
		},
	}}}
	var (
		succeedPaths []string
		graphqlText  string
		queryItem    *build.QueryDocumentItem
		user         = o.baseHandler.GetUser(c)
	)
	for _, item := range operations {
		dataName := o.modelRoot.GetDataName(item)
		itemResult, itemOk := models.OperationResultMap.Load(dataName)
		if !itemOk {
			continue
		}

		updateRoleFunc(itemResult.AuthorizationConfig.RoleConfig, param.RoleCodes)
		switch item.Engine {
		case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
			if graphqlText, err = o.graphqlText.Read(dataName); err != nil {
				continue
			}

			if queryItem, err = build.NewQueryDocumentItem(graphqlText); err != nil {
				continue
			}

			queryItem.ModifyOperationDirective(rbacDirective)
			if len(queryItem.Errors) > 0 {
				continue
			}

			err = o.graphqlText.WriteCustom(dataName, user, func(graphqlFile *os.File) error {
				return queryItem.PrintQueryDocument(graphqlFile)
			})
		case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
			err = o.rewriteOperationExtensionRbac(user, dataName, models.OperationFunction, itemResult)
		case wgpb.OperationExecutionEngine_ENGINE_PROXY:
			err = o.rewriteOperationExtensionRbac(user, dataName, models.OperationProxy, itemResult)
		}
		if err != nil {
			continue
		}

		succeedPaths = append(succeedPaths, dataName)
	}
	return c.JSON(http.StatusOK, succeedPaths)
}

func (o *operation) rewriteOperationExtensionRbac(user, path string, text *fileloader.ModelText[models.Operation], rewriteData *wgpb.Operation) (err error) {
	content, err := text.Read(path)
	if err != nil {
		return
	}

	var result *wgpb.Operation
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		return
	}

	result.AuthorizationConfig.RoleConfig = rewriteData.AuthorizationConfig.RoleConfig
	result.AuthenticationConfig = rewriteData.AuthenticationConfig
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return
	}

	return text.Write(path, user, resultBytes)
}

// @Tags operation
// @Description "getHookOptions"
// @Param dataName path string true "dataName"
// @Success 200 {object} models.HookOptions "hook配置"
// @Failure 400 {object} i18n.CustomError
// @Router /operation/hookOptions/{dataName} [get]
func (o *operation) getHookOptions(c echo.Context) error {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, models.GetOperationHookOptions(dataName))
}

func (o *operation) updateOperationExtensionText(c echo.Context, text *fileloader.ModelText[models.Operation]) (err error) {
	dataName, err := o.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}

	body, user, err := o.baseHandler.GetUserAndBody(c)
	if err != nil {
		return
	}

	dataName = utils.NormalizePath(text.Title, dataName)
	if err = text.Write(dataName, user, body); err != nil {
		textFilepath := text.GetPath(dataName)
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.FileWriteError, textFilepath)
		return
	}

	return c.NoContent(http.StatusOK)
}
