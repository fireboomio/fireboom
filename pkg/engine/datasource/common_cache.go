// Package datasource
/*
 使用prisma引擎的数据源的缓存，其中graphql缓存适用与其他数据源的缓存
*/
package datasource

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
)

const (
	cacheSchema = "schema"
	cacheDmmf   = "dmmf"
)

var (
	CachePrismaSchemaText  *fileloader.ModelText[models.Datasource]
	CacheGraphqlSchemaText *fileloader.ModelText[models.Datasource]
	CacheDmmfText          *fileloader.ModelText[models.Datasource]
	introspectionDirname   = utils.NormalizePath(consts.RootExported, consts.ExportedIntrospectionParent)
	migrationDirname       = utils.NormalizePath(consts.RootExported, consts.ExportedMigrationParent)
)

func buildDatasourceCache(name string, extension fileloader.Extension) *fileloader.ModelText[models.Datasource] {
	itemText := &fileloader.ModelText[models.Datasource]{
		Title:     name,
		Root:      introspectionDirname,
		Extension: extension,
		TextRW: &fileloader.MultipleTextRW[models.Datasource]{
			Enabled: func(item *models.Datasource, _ ...string) bool { return item.Enabled },
			Name:    fileloader.DefaultBasenameFunc(name),
		},
	}
	utils.RegisterInitMethod(30, func() {
		itemText.RelyModel = models.DatasourceRoot
		itemText.Init()
	})
	return itemText
}

func init() {
	CachePrismaSchemaText = buildDatasourceCache(cacheSchema, fileloader.ExtPrisma)
	CacheGraphqlSchemaText = buildDatasourceCache(cacheSchema, fileloader.ExtGraphql)
	CacheDmmfText = buildDatasourceCache(cacheDmmf, fileloader.ExtJson)
}
