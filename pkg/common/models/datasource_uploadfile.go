// Package models
/*
 使用fileloader.ModelText管理upload下上传的文件
 依赖与父model Datasource实现多文件管理
 DatasourceUploadOas rest数据源依赖的swagger文件
 DatasourceUploadPrisma prisma数据源依赖的文件
 DatasourceUploadSqlite sqlite数据源依赖的文件
 DatasourceUploadGraphql graphql数据源依赖的文件
 提供GetDatasourceUploadFilepath函数，根据类型不同返回文件路径
 当钩子配置变更时，自动重置路径字典
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

var (
	DatasourceUploadOas      *fileloader.ModelText[Datasource]
	DatasourceUploadAsyncapi *fileloader.ModelText[Datasource]
	DatasourceUploadPrisma   *fileloader.ModelText[Datasource]
	DatasourceUploadSqlite   *fileloader.ModelText[Datasource]
	DatasourceUploadGraphql  *fileloader.ModelText[Datasource]

	datasourceUploadFileMap map[wgpb.DataSourceKind]*fileloader.ModelText[Datasource]
)

func buildDatasourceUploadFile(parent string, pathFunc func(*Datasource) string, kind wgpb.DataSourceKind) *fileloader.ModelText[Datasource] {
	item := &fileloader.ModelText[Datasource]{
		Title:                  parent,
		Root:                   utils.NormalizePath(consts.RootUpload, parent),
		ExtensionIgnored:       true,
		RelyModelActionIgnored: true,
		TextRW: &fileloader.MultipleTextRW[Datasource]{
			Enabled: func(item *Datasource, _ ...string) bool { return item.Enabled },
			Name: func(dataName string, _ int, _ ...string) (path string, extension bool) {
				data, err := DatasourceRoot.GetByDataName(dataName)
				if err != nil {
					return
				}

				path = pathFunc(data)
				return
			},
		},
	}

	datasourceUploadFileMap[kind] = item
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = DatasourceRoot
		item.Init()
		item.ResetRootDirectory()
	})
	return item
}

// GetDatasourceUploadFilepath 根据类型不同返回文件路径
func GetDatasourceUploadFilepath(data *Datasource) string {
	uploadFile, ok := datasourceUploadFileMap[data.Kind]
	if !ok {
		return ""
	}

	return uploadFile.GetPath(data.Name)
}

func init() {
	datasourceUploadFileMap = make(map[wgpb.DataSourceKind]*fileloader.ModelText[Datasource])
	databasePathFunc := func(datasource *Datasource) string {
		if datasource.CustomDatabase == nil {
			return ""
		}

		return utils.GetVariableString(datasource.CustomDatabase.DatabaseUrl)
	}
	DatasourceUploadSqlite = buildDatasourceUploadFile(consts.UploadSqliteParent, databasePathFunc, wgpb.DataSourceKind_SQLITE)
	DatasourceUploadPrisma = buildDatasourceUploadFile(consts.UploadPrismaParent, databasePathFunc, wgpb.DataSourceKind_PRISMA)
	DatasourceUploadOas = buildDatasourceUploadFile(consts.UploadOasParent, func(datasource *Datasource) string {
		if datasource.CustomRest == nil {
			return ""
		}
		return datasource.CustomRest.OasFilepath
	}, wgpb.DataSourceKind_REST)
	DatasourceUploadAsyncapi = buildDatasourceUploadFile(consts.UploadAsyncapiParent, func(datasource *Datasource) string {
		if datasource.CustomAsyncapi == nil {
			return ""
		}
		return datasource.CustomAsyncapi.OasFilepath
	}, wgpb.DataSourceKind_ASYNCAPI)
	DatasourceUploadGraphql = buildDatasourceUploadFile(consts.UploadGraphqlParent, func(datasource *Datasource) string {
		if datasource.CustomGraphql == nil {
			return ""
		}
		return datasource.CustomGraphql.SchemaFilepath
	}, wgpb.DataSourceKind_GRAPHQL)
}
