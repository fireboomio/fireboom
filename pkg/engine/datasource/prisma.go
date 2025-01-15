// Package datasource
/*
 prisma类型数据源的实现
*/
package datasource

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"path/filepath"
)

func init() {
	actionMap[wgpb.DataSourceKind_PRISMA] = func(ds *models.Datasource, schema string) Action {
		return &actionPrisma{ds: ds, prismaSchema: schema}
	}
}

type actionPrisma struct {
	ds           *models.Datasource
	prismaSchema string
}

func (a *actionPrisma) Introspect() (graphqlSchema string, err error) {
	if utils.InvokeFunctionLimit(consts.LicensePrismaDatasource) {
		return
	}

	return introspectForPrisma(a, a.ds.Name)
}

func (a *actionPrisma) BuildDataSourceConfiguration(*ast.SchemaDocument) (config *wgpb.DataSourceConfiguration, err error) {
	prismaSchemaFilepath := models.DatasourceUploadPrisma.GetPath(a.ds.Name)
	introspectSchema, _ := extractIntrospectSchema(prismaSchemaFilepath)
	environmentVariable, _ := extractEnvironmentVariable(introspectSchema)
	config = &wgpb.DataSourceConfiguration{CustomDatabase: &wgpb.DataSourceCustom_Database{
		DatabaseURL:         utils.MakeStaticVariable(prismaSchemaFilepath),
		JsonInputVariables:  []string{consts.ScalarJSON},
		EnvironmentVariable: environmentVariable,
	}}
	return
}

func (a *actionPrisma) RuntimeDataSourceConfiguration(config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration, err error) {
	prismaSchemaFilepath := utils.GetVariableString(config.CustomDatabase.DatabaseURL)
	configs, fields = buildRuntimeDataSourceConfigurationForPrisma(prismaSchemaFilepath, config)
	return
}

func (a *actionPrisma) ExtendDocument(document *ast.SchemaDocument) {
	extendOptionalRawField(document)
}

func (a *actionPrisma) GetFieldRealName(fieldName string) string {
	return getRawFieldOriginName(fieldName)
}

func (a *actionPrisma) fetchSchemaEngineInput() (engineInput EngineInput, skipGraphql bool, err error) {
	if skipGraphql = len(a.prismaSchema) > 0; skipGraphql {
		engineInput.PrismaSchema = a.prismaSchema
	} else {
		engineInput.PrismaSchemaFilepath = models.DatasourceUploadPrisma.GetPath(a.ds.Name)
		engineInput.PrismaSchema, err = extractIntrospectSchema(engineInput.PrismaSchemaFilepath)
	}
	engineInput.EnvironmentRequired = true
	return
}

// 构建运行时数据源配置，数据库类型数据源/prisma数据源
func buildRuntimeDataSourceConfigurationForPrisma(prismaSchema string, config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration) {
	staticData := &wgpb.DataSourceCustom_Static{Data: &wgpb.ConfigurationVariable{}}
	customDatabase := *config.CustomDatabase
	customDatabase.PrismaSchema, _ = filepath.Abs(prismaSchema)
	customDatabase.ExecuteTimeoutSeconds = utils.GetInt32WithLockViper(consts.DatabaseExecuteTimeout)
	customDatabase.CloseTimeoutSeconds = utils.GetInt32WithLockViper(consts.DatabaseCloseTimeout)
	configs, fields = copyDatasourceWithRootNodes(config, func(_ *wgpb.TypeField, configItem *wgpb.DataSourceConfiguration) bool {
		configItem.CustomDatabase = &customDatabase
		configItem.CustomStatic = staticData
		return true
	})
	return
}
