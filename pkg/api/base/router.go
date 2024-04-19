// Package base
/*
 与fileloader.Model, base.Handler结合使用
 提供基础的路由注册，按照fileloader.Model定义的类型，embed/single/multiple注册不同的路由
 通过范型约束及类型判断使得所有路由注册可以复用
*/
package base

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/labstack/echo/v4"
	"path"
	"strings"
)

const (
	DataNamePath     = "/:" + consts.PathParamDataName
	emptyPath        = ""
	singlePath       = "/single"
	treePath         = "/tree"
	batchPath        = "/batch"
	importPath       = "/import"
	exportPath       = "/export"
	copyPath         = "/copy"
	renamePath       = "/rename"
	renameParentPath = "/renameParent"
	deleteParentPath = "/deleteParent" + DataNamePath
	withLockUser     = "/withLockUser"
	dataWithLockUser = withLockUser + DataNamePath
)

// RegisterBaseRouter 注册基础路由
// 通过optional可以扩展路由
// 根据modelRoot.DataRW类型注册不同路由
func RegisterBaseRouter[T any](router *echo.Group, modelRoot *fileloader.Model[T], optional ...func(*echo.Group, *echo.Group, *Handler[T], *fileloader.Model[T])) {
	handler := NewBaseHandler(modelRoot)
	modelName := modelRoot.GetModelName()
	subRouter := router.Group("/" + modelName)

	var metaRequiredRoutes []*echo.Route
	switch modelRoot.DataRW.(type) {
	case *fileloader.MultipleDataRW[T]:
		metaRequiredRoutes = append(metaRequiredRoutes,
			subRouter.POST(emptyPath, handler.insert),
			subRouter.DELETE(DataNamePath, handler.deleteByDataName),
			subRouter.PUT(emptyPath, handler.updateByDataName),
			subRouter.GET(DataNamePath, handler.getByDataName),
			subRouter.GET(dataWithLockUser, handler.getWithLockUserByDataName),
			subRouter.GET(emptyPath, handler.list),
			subRouter.GET(treePath, handler.tree),

			subRouter.POST(batchPath, handler.insertBatch),
			subRouter.PUT(batchPath, handler.updateBatch),
			subRouter.DELETE(batchPath, handler.deleteBatchByDataNames),

			subRouter.POST(copyPath, handler.copyByDataName),
			subRouter.POST(renamePath, handler.renameByDataName),
			subRouter.POST(renameParentPath, handler.renameByParentDataName),
			subRouter.DELETE(deleteParentPath, handler.deleteByParentDataName),

			subRouter.POST(importPath, handler.importData),
			subRouter.GET(exportPath, handler.exportData),
		)
	case *fileloader.SingleDataRW[T]:
		metaRequiredRoutes = append(metaRequiredRoutes,
			subRouter.PUT(emptyPath, handler.updateByDataName),
			subRouter.GET(singlePath, handler.getSingleData),
			subRouter.GET(withLockUser, handler.getSingleDataWithLockUser),
		)
	case *fileloader.EmbedDataRW[T]:
		metaRequiredRoutes = append(metaRequiredRoutes,
			subRouter.GET(singlePath, handler.getSingleData),
		)
	}

	AddRouterMetas(modelRoot, metaRequiredRoutes...)

	for _, item := range optional {
		item(router, subRouter, handler, modelRoot)
	}
}

const ModelNameFlag = "$modelName$"

var RouterMetasMap map[string][]*RouterMeta

type RouterMeta struct {
	Name string
	Data any
}

func init() {
	RouterMetasMap = make(map[string][]*RouterMeta)
}

// AddRouterMetas 添加路由元数据
// 是动态构建swagger文档的基础
// 路由注册过程中将范型信息留存，便于后续反射动态构建swagger
func AddRouterMetas[T any](modelRoot *fileloader.Model[T], metaRequiredRoutes ...*echo.Route) {
	var data T
	modelName := modelRoot.GetModelName()
	prefix := path.Join(configs.ApplicationData.ContextPath, modelName)
	for _, item := range metaRequiredRoutes {
		itemPath := strings.ReplaceAll(item.Path, prefix, ModelNameFlag)
		itemPath = strings.ReplaceAll(itemPath, ":"+consts.PathParamDataName, "{"+consts.PathParamDataName+"}")
		itemPath = path.Join(item.Method, itemPath)
		RouterMetasMap[itemPath] = append(RouterMetasMap[itemPath], &RouterMeta{modelName, data})
	}
}
