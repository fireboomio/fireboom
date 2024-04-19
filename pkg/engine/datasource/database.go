// Package datasource
/*
 数据库类型数据源的实现
*/
package datasource

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"strings"
)

func init() {
	generateDatabaseFunc := func(ds *models.Datasource, _ string) Action { return &actionDatabase{ds: ds} }
	actionMap[wgpb.DataSourceKind_POSTGRESQL] = generateDatabaseFunc
	actionMap[wgpb.DataSourceKind_MYSQL] = generateDatabaseFunc
	actionMap[wgpb.DataSourceKind_SQLSERVER] = generateDatabaseFunc
	actionMap[wgpb.DataSourceKind_MONGODB] = generateDatabaseFunc
	actionMap[wgpb.DataSourceKind_SQLITE] = generateDatabaseFunc
}

const (
	ignoreEmptyDatabaseError = "The introspected database was empty."
	introspectSchemaFormat   = `datasource db {
  provider = "%s"
  url      = "%s"
}`
)

type actionDatabase struct {
	ds *models.Datasource
}

func (a *actionDatabase) Introspect() (graphqlSchema string, err error) {
	return introspectForPrisma(a.fetchIntrospectSchema, a.ds.Name)
}

func (a *actionDatabase) BuildDataSourceConfiguration(*ast.SchemaDocument) (config *wgpb.DataSourceConfiguration, err error) {
	databaseUrl, _ := a.ds.CustomDatabase.GetDatabaseUrl(a.ds.Kind, a.ds.Name)
	config = &wgpb.DataSourceConfiguration{CustomDatabase: &wgpb.DataSourceCustom_Database{
		DatabaseURL: utils.MakeStaticVariable(databaseUrl),
	}}
	return
}

func (a *actionDatabase) RuntimeDataSourceConfiguration(config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration, err error) {
	prismaSchemaFilepath := CachePrismaSchemaText.GetPath(config.Id)
	configs, fields = buildRuntimeDataSourceConfigurationForPrisma(prismaSchemaFilepath, config)
	return
}

// 根据数据库类型组装introspectSchema
func (a *actionDatabase) fetchIntrospectSchema() (introspectSchema, _ string, skipGraphql bool, err error) {
	databaseConfig := a.ds.CustomDatabase
	if databaseConfig == nil {
		err = i18n.NewCustomErrorWithMode(datasourceModelName, nil, i18n.StructParamEmtpyError, "customDatabase")
		return
	}

	databaseURL, err := databaseConfig.GetDatabaseUrl(a.ds.Kind, a.ds.Name)
	if err != nil {
		return
	}

	if databaseURL == "" {
		err = i18n.NewCustomErrorWithMode(datasourceModelName, nil, i18n.DatasourceDatabaseUrlEmptyError)
		return
	}

	databaseKind := strings.ToLower(a.ds.Kind.String())
	introspectSchema = fmt.Sprintf(introspectSchemaFormat, databaseKind, databaseURL)
	return
}
