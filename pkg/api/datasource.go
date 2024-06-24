// Package api
/*
 在基础路由上进行扩展
 提供数据建模相关的能力，包括dmmf, graphql, prisma, query等
*/
package api

import (
	"bytes"
	"context"
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	engineDatasource "fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	engineClient "github.com/prisma/prisma-client-go/engine"
	"github.com/prisma/prisma-client-go/generator/ast/dmmf"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"net/http"
	"strings"
	"sync"
)

func DatasourceExtraRouter(_, datasourceRouter *echo.Group, baseHandler *base.Handler[models.Datasource], modelRoot *fileloader.Model[models.Datasource]) {
	handler := &datasource{
		baseHandler: baseHandler,
		modelRoot:   modelRoot, modelName: modelRoot.GetModelName(),
		queryEngines: &sync.Map{},
	}
	base.AddRouterMetas(modelRoot,
		datasourceRouter.POST("/checkConnection", handler.checkConnection),
	)
	datasourceRouter.GET("/dmmf"+base.DataNamePath, handler.getDmmf)
	datasourceRouter.GET("/graphql"+base.DataNamePath, handler.getGraphql)
	datasourceRouter.POST("/graphqlQuery"+base.DataNamePath, handler.graphqlQuery)
	datasourceRouter.GET("/prisma"+base.DataNamePath, handler.getPrisma)
	datasourceRouter.POST("/prisma"+base.DataNamePath, handler.updatePrismaText)
	datasourceRouter.POST("/migrate"+base.DataNamePath, handler.migrate)
}

type (
	datasource struct {
		baseHandler  *base.Handler[models.Datasource]
		modelRoot    *fileloader.Model[models.Datasource]
		modelName    string
		queryEngines *sync.Map
	}
	datasourcePing struct {
		models.Datasource
		PrismaSchema string `json:"prismaSchema"`
	}
)

// @Tags datasource
// @Description "检查连接"
// @Param data body any true "#/definitions/$modelName$"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /$modelName$/checkConnection [post]
func (d *datasource) checkConnection(c echo.Context) error {
	var param datasourcePing
	if err := c.Bind(&param); err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.ParamBindError)
	}

	action, err := engineDatasource.GetDatasourceAction(&param.Datasource, param.PrismaSchema)
	if err != nil {
		return err
	}

	if _, err = action.Introspect(); err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.DatasourceConnectionError)
	}

	return c.NoContent(http.StatusOK)
}

// @Tags datasource
// @Description "迁移"
// @Param dataName path string true "model名称"
// @Param data body string true "迁移数据"
// @Success 200 {string} string "prisma文本"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/migrate/{dataName} [post]
func (d *datasource) migrate(c echo.Context) error {
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return err
	}

	migrateBytes, _, err := d.baseHandler.GetUserAndBody(c)
	if err != nil {
		return err
	}

	ctx := context.WithValue(c.Request().Context(), eventbus.ChannelDatasource, data.Name)
	err = engineDatasource.Migrate(ctx, string(migrateBytes), engineDatasource.CachePrismaSchemaText.GetPath(data.Name))
	if err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.PrismaMigrateError)
	}

	if err = utils.ReloadPrismaCache(data.Name); err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.PrismaQueryError)
	}

	if utils.GetBoolWithLockViper(consts.DevMode) {
		go utils.BuildAndStart()
	}
	return c.NoContent(http.StatusOK)
}

// @Tags datasource
// @Description "获取prisma文本"
// @Param dataName path string true "model名称"
// @Success 200 {string} string "prisma文本"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/prisma/{dataName} [get]
func (d *datasource) getPrisma(c echo.Context) error {
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return err
	}

	prismaFilepath, _, err := d.getPrismaFilepath(c, data)
	if err != nil {
		return err
	}

	base.SetHeaderCacheControlNoCache(c)
	return c.File(prismaFilepath)
}

// @Tags datasource
// @Description "获取graphql文本"
// @Param dataName path string true "model名称"
// @Success 200 {string} string "graphql文本"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/graphql/{dataName} [get]
func (d *datasource) getGraphql(c echo.Context) error {
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return err
	}

	base.SetHeaderCacheControlNoCache(c)
	return c.File(engineDatasource.CacheGraphqlSchemaText.GetPath(data.Name))
}

// @Tags datasource
// @Description "获取dmmf"
// @Param dataName path string true "model名称"
// @Param reload query bool false "重载dmmf"
// @Success 200 {object} any "dmmf"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/dmmf/{dataName} [get]
func (d *datasource) getDmmf(c echo.Context) (err error) {
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return
	}

	unmarshalFunc := func(content string) error {
		var dmmfDoc *dmmf.Document
		_ = json.Unmarshal([]byte(content), &dmmfDoc)
		base.SetHeaderCacheControlNoCache(c)
		return c.JSON(http.StatusOK, dmmfDoc)
	}

	prismaFilepath, cacheUsed, err := d.getPrismaFilepath(c, data)
	if err != nil {
		return
	}

	if cacheUsed {
		if dmmfContent, _ := engineDatasource.CacheDmmfText.Read(data.Name); dmmfContent != "" {
			return unmarshalFunc(dmmfContent)
		}
	}

	dmmfContent, err := engineDatasource.IntrospectDMMF(prismaFilepath, data.Kind == wgpb.DataSourceKind_PRISMA)
	if err != nil {
		return i18n.NewCustomErrorWithMode(d.modelName, err, i18n.PrismaQueryError)
	}

	if cacheUsed {
		go func() {
			_ = engineDatasource.CacheDmmfText.Write(data.Name, fileloader.SystemUser, []byte(dmmfContent))
		}()
	}
	return unmarshalFunc(dmmfContent)
}

// @Tags datasource
// @Description "updatePrismaText"
// @Param dataName path string true "dataName"
// @Param data body string true "文本"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/prisma/{dataName} [post]
func (d *datasource) updatePrismaText(c echo.Context) (err error) {
	dataName, err := d.baseHandler.GetPathParamDataName(c)
	if err != nil {
		return
	}

	body, user, err := d.baseHandler.GetUserAndBody(c)
	if err != nil {
		return
	}

	if err = models.DatasourceUploadPrisma.Write(dataName, user, body); err != nil {
		prismaFilepath := models.DatasourceUploadPrisma.GetPath(dataName)
		err = i18n.NewCustomErrorWithMode(d.modelName, err, i18n.FileWriteError, prismaFilepath)
		return
	}

	var (
		providerPrefix = []byte(`provider = "`)
		providerSuffix = []byte(`"`)
	)
	prefixIndex := bytes.Index(body, providerPrefix)
	startIndex := prefixIndex + len(providerPrefix)
	suffixIndex := bytes.Index(body[startIndex:], providerSuffix)
	data, _ := models.DatasourceRoot.GetByDataName(dataName)
	if prefixIndex != -1 && suffixIndex != -1 && data != nil {
		data.KindForPrisma = wgpb.DataSourceKind(wgpb.DataSourceKind_value[strings.ToUpper(string(body[startIndex:startIndex+suffixIndex]))])
		_ = models.DatasourceRoot.InsertOrUpdate(data)
	}
	return c.NoContent(http.StatusOK)
}

// @Tags datasource
// @Description "graphqlQuery"
// @Param dataName path string true "dataName"
// @Param data body string true "graphql query"
// @Success 200 "OK"
// @Failure 400 {object} i18n.CustomError
// @Router /datasource/graphqlQuery/{dataName} [post]
func (d *datasource) graphqlQuery(c echo.Context) (err error) {
	defer func() {
		if err != nil {
			err = i18n.NewCustomErrorWithMode(d.modelName, err, i18n.PrismaQueryError)
		}
	}()
	data, err := d.baseHandler.GetOneByDataName(c)
	if err != nil {
		return
	}

	var graphqlRequest engineClient.GQLRequest
	if err = c.Bind(&graphqlRequest); err != nil {
		return
	}

	if graphqlRequest.Query, err = engineClient.InlineQuery(graphqlRequest.Query, graphqlRequest.Variables); err != nil {
		return
	}

	queryEngine, ok := utils.LoadFromSyncMap[*engineClient.QueryEngine](d.queryEngines, data.Name)
	if !ok {
		var prismaSchema string
		prismaSchema, err = engineDatasource.CachePrismaSchemaText.Read(data.Name)
		if err != nil {
			return
		}

		queryEngine = engineClient.NewQueryEngine(prismaSchema, false)
		if err = queryEngine.Connect(); err != nil {
			return
		}

		queryEngine.RewriteErrorsFunc = database.RewriteErrors
		d.queryEngines.Store(data.Name, queryEngine)
		defer func() {
			_ = queryEngine.Disconnect()
			d.queryEngines.Delete(data.Name)
		}()
	}

	var graphqlResult map[string]any
	if err = queryEngine.DoManyQuery(c.Request().Context(), graphqlRequest, &graphqlResult); err != nil {
		return
	}

	return c.JSON(http.StatusOK, graphqlResult)
}

func (d *datasource) getPrismaFilepath(c echo.Context, data *models.Datasource) (prismaFilepath string, cacheUsed bool, err error) {
	cacheUsed = c.QueryParam(consts.QueryParamCrud) == "" || data.Kind != wgpb.DataSourceKind_PRISMA
	if cacheUsed {
		if prismaFilepath = engineDatasource.CachePrismaSchemaText.GetPath(data.Name); !utils.NotExistFile(prismaFilepath) {
			return
		}

		if err = utils.ReloadPrismaCache(data.Name); err != nil {
			err = i18n.NewCustomErrorWithMode(d.modelName, err, i18n.PrismaQueryError)
			return
		}
	}

	if !data.Enabled {
		err = i18n.NewCustomErrorWithMode(d.modelName, nil, i18n.DatasourceDisabledError)
		return
	}

	prismaFilepath = models.DatasourceUploadPrisma.GetPath(data.Name)
	return
}
