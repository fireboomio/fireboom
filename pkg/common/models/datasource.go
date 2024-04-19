// Package models
/*
 使用fileloader.Model管理数据源配置
 读取store/datasource下的文件，变更后会触发引擎编译，支持逻辑删除
 使用key为kind，value为func的map来支持不同类型数据源的链接配置
 数据库类型数据源支持url和独立配置
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
)

type Datasource struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
	DeleteTime   string `json:"deleteTime"`
	CacheEnabled bool   `json:"cacheEnabled"`

	Kind           wgpb.DataSourceKind `json:"kind"`
	CustomRest     *CustomRest         `json:"customRest"`
	CustomAsyncapi *CustomRest         `json:"customAsyncapi,omitempty"`
	CustomGraphql  *CustomGraphql      `json:"customGraphql"`
	CustomDatabase *CustomDatabase     `json:"customDatabase"`
}

type CustomDatabaseKind int32

const (
	customDatabaseKindUrl   CustomDatabaseKind = 0
	customDatabaseKindAlone CustomDatabaseKind = 1
)

type (
	CustomGraphql struct {
		Customized     bool                        `json:"customized"`
		Headers        map[string]*wgpb.HTTPHeader `json:"headers"`
		Endpoint       string                      `json:"endpoint"`
		SchemaFilepath string                      `json:"schemaFilepath"`
	}
	CustomRest struct {
		OasFilepath       string                                `json:"oasFilepath"`
		BaseUrl           *wgpb.ConfigurationVariable           `json:"baseUrl"`
		Headers           map[string]*wgpb.HTTPHeader           `json:"headers"`
		ResponseExtractor *wgpb.DataSourceRESTResponseExtractor `json:"responseExtractor"`
	}
	CustomDatabase struct {
		Kind          CustomDatabaseKind          `json:"kind"`
		DatabaseUrl   *wgpb.ConfigurationVariable `json:"databaseUrl"`
		DatabaseAlone *CustomDatabaseAlone        `json:"databaseAlone"`
	}
	CustomDatabaseAlone struct {
		Host     string `json:"host"`
		Port     int32  `json:"port"`
		Database string `json:"database"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
)

func (d *Datasource) IsCustomDatabase() bool {
	return slices.Contains(databaseKinds, d.Kind)
}

func (c *CustomGraphql) GetGraphqlUrl(dsName string) (string, error) {
	if !c.Customized {
		return c.Endpoint, nil
	}

	serverOptions := configs.GlobalSettingRoot.FirstData().ServerOptions
	if serverOptions == nil || serverOptions.ServerUrl == nil {
		return "", i18n.NewCustomErrorWithMode(DatasourceRoot.GetModelName(), nil, i18n.SettingServerUrlEmptyError)
	}

	return utils.GetVariableString(serverOptions.ServerUrl) + fmt.Sprintf(`/gqls/%s/graphql`, dsName), nil
}

func (c *CustomDatabase) GetDatabaseUrl(dsKind wgpb.DataSourceKind, dsName string) (string, error) {
	var urlFunc func(*CustomDatabase, string) string
	var ok bool
	switch c.Kind {
	case customDatabaseKindUrl:
		urlFunc, ok = databaseKindUrlFuncMap[dsKind]
	case customDatabaseKindAlone:
		urlFunc, ok = databaseKindAloneFuncMap[dsKind]
	}
	if !ok {
		return "", i18n.NewCustomErrorWithMode(DatasourceRoot.GetModelName(), nil, i18n.DatasourceKindNotSupportedError, dsKind)
	}

	return urlFunc(c, dsName), nil
}

var (
	DatasourceRoot           *fileloader.Model[Datasource]
	databaseKindUrlFuncMap   map[wgpb.DataSourceKind]func(*CustomDatabase, string) string
	databaseKindAloneFuncMap map[wgpb.DataSourceKind]func(*CustomDatabase, string) string
	databaseKinds            = []wgpb.DataSourceKind{
		wgpb.DataSourceKind_PRISMA,
		wgpb.DataSourceKind_POSTGRESQL,
		wgpb.DataSourceKind_MYSQL,
		wgpb.DataSourceKind_SQLSERVER,
		wgpb.DataSourceKind_MONGODB,
		wgpb.DataSourceKind_SQLITE,
	}
)

const (
	databaseUrlFormat = `%s://%s:%s@%s:%d/%s`
	mongodbUrlFormat  = `%s+srv://%s:%s@%s:%d/%s`
	sqliteUrlFormat   = `file:%s`
)

func init() {
	DatasourceRoot = &fileloader.Model[Datasource]{
		Root:      utils.NormalizePath(consts.RootStore, consts.StoreDatasourceParent),
		Extension: fileloader.ExtJson,
		DataHook: &fileloader.DataHook[Datasource]{
			OnInsert: func(item *Datasource) error {
				item.CreateTime = utils.TimeFormatNow()
				return nil
			},
			OnUpdate: func(src, dst *Datasource, user string) (err error) {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				if !dst.CacheEnabled || src.Name != dst.Name {
					return
				}

				return utils.ReloadPrismaCache(dst.Name)
			},
			AfterInsert: func(item *Datasource, user string) bool { return item.Enabled },
		},
		DataRW: &fileloader.MultipleDataRW[Datasource]{
			GetDataName: func(item *Datasource) string { return item.Name },
			SetDataName: func(item *Datasource, name string) { item.Name = name },
			Filter:      func(item *Datasource) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Datasource) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}

	utils.RegisterInitMethod(20, func() {
		DatasourceRoot.Init()
		configs.AddFileLoaderQuestionCollector(DatasourceRoot.GetModelName(), func(dataName string) map[string]any {
			data, _ := DatasourceRoot.GetByDataName(dataName)
			if data == nil {
				return nil
			}

			result := map[string]any{fieldEnabled: data.Enabled, "kind": data.Kind}
			if data.CustomGraphql != nil {
				result["customized"] = data.CustomGraphql.Customized
			}
			return result
		})
		utils.AddBuildAndStartFuncWatcher(func(f func()) { DatasourceRoot.DataHook.AfterMutate = f })
	})

	databaseKindUrlFuncMap = make(map[wgpb.DataSourceKind]func(*CustomDatabase, string) string)
	databaseUrlFunc := func(database *CustomDatabase, _ string) string {
		return utils.GetVariableString(database.DatabaseUrl)
	}
	databaseKindUrlFuncMap[wgpb.DataSourceKind_POSTGRESQL] = databaseUrlFunc
	databaseKindUrlFuncMap[wgpb.DataSourceKind_MYSQL] = databaseUrlFunc
	databaseKindUrlFuncMap[wgpb.DataSourceKind_SQLSERVER] = databaseUrlFunc
	databaseKindUrlFuncMap[wgpb.DataSourceKind_MONGODB] = databaseUrlFunc
	databaseKindUrlFuncMap[wgpb.DataSourceKind_SQLITE] = func(database *CustomDatabase, name string) string {
		url, _ := filepath.Abs(utils.NormalizePath(DatasourceUploadSqlite.Root, utils.GetVariableString(database.DatabaseUrl)))
		return fmt.Sprintf(sqliteUrlFormat, filepath.ToSlash(url))
	}

	databaseAloneFunc := func(c *CustomDatabaseAlone, kind wgpb.DataSourceKind, format string) string {
		databaseKind := strings.ToLower(kind.String())
		return fmt.Sprintf(format, databaseKind, c.Username, c.Password, c.Host, c.Port, c.Database)
	}
	databaseKindAloneFuncMap = make(map[wgpb.DataSourceKind]func(*CustomDatabase, string) string)
	databaseKindAloneFuncMap[wgpb.DataSourceKind_POSTGRESQL] = func(alone *CustomDatabase, _ string) string {
		return databaseAloneFunc(alone.DatabaseAlone, wgpb.DataSourceKind_POSTGRESQL, databaseUrlFormat)
	}
	databaseKindAloneFuncMap[wgpb.DataSourceKind_MYSQL] = func(alone *CustomDatabase, _ string) string {
		return databaseAloneFunc(alone.DatabaseAlone, wgpb.DataSourceKind_MYSQL, databaseUrlFormat)
	}
	databaseKindAloneFuncMap[wgpb.DataSourceKind_SQLSERVER] = func(alone *CustomDatabase, _ string) string {
		return databaseAloneFunc(alone.DatabaseAlone, wgpb.DataSourceKind_SQLSERVER, databaseUrlFormat)
	}
	databaseKindAloneFuncMap[wgpb.DataSourceKind_MONGODB] = func(alone *CustomDatabase, _ string) string {
		return databaseAloneFunc(alone.DatabaseAlone, wgpb.DataSourceKind_MONGODB, mongodbUrlFormat)
	}
}
