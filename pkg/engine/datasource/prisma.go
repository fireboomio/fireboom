// Package datasource
/*
 prisma类型数据源的实现
*/
package datasource

import (
	"context"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"path/filepath"
	"strings"
)

func init() {
	actionMap[wgpb.DataSourceKind_PRISMA] = func(ds *models.Datasource, schema string) Action {
		return &actionPrisma{ds: ds, prismaSchema: schema}
	}
}

type actionPrisma struct {
	ds                  *models.Datasource
	prismaSchema        string
	environmentVariable string
}

func (a *actionPrisma) Introspect() (graphqlSchema string, err error) {
	if utils.InvokeFunctionLimit(consts.LicensePrismaDatasource) {
		return
	}

	return introspectForPrisma(a.fetchIntrospectSchema, a.ds.Name, models.DatasourceUploadPrisma.GetPath(a.ds.Name))
}

func (a *actionPrisma) BuildDataSourceConfiguration(*ast.SchemaDocument) (config *wgpb.DataSourceConfiguration, err error) {
	prismaSchemaFilepath := models.DatasourceUploadPrisma.GetPath(a.ds.Name)
	if a.environmentVariable == "" {
		_, a.environmentVariable, _, _ = fetchEnvironmentVariable(prismaSchemaFilepath)
	}
	config = &wgpb.DataSourceConfiguration{CustomDatabase: &wgpb.DataSourceCustom_Database{
		DatabaseURL:         utils.MakeStaticVariable(prismaSchemaFilepath),
		JsonInputVariables:  []string{consts.ScalarJSON},
		EnvironmentVariable: a.environmentVariable,
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

func (a *actionPrisma) fetchIntrospectSchema() (prismaSchema, env string, skipGraphql bool, err error) {
	if skipGraphql = len(a.prismaSchema) > 0; skipGraphql {
		prismaSchema = a.prismaSchema
		return
	}

	prismaSchema, a.environmentVariable, env, err = fetchEnvironmentVariable(models.DatasourceUploadPrisma.GetPath(a.ds.Name))
	return
}

// 内省并缓存graphqlSchema，数据库类型数据源/prisma数据源
func introspectForPrisma(fetchIntrospectSchemaFunc func() (string, string, bool, error), dsName string, schemaFilepath ...string) (graphqlSchema string, err error) {
	introspectSchema, env, skipGraphql, err := fetchIntrospectSchemaFunc()
	if err != nil {
		return
	}

	rpcExt := database.JsonRPCExtension{}
	if len(env) > 0 {
		rpcExt.CmdEnvs = []string{env}
	}
	ctx := context.WithValue(context.Background(), eventbus.ChannelDatasource, dsName)
	prismaSchema, err := BuildEngine().IntrospectPrismaDatabaseSchema(ctx, introspectSchema, rpcExt)
	if err != nil {
		if !strings.Contains(err.Error(), ignoreEmptyDatabaseError) {
			return
		}

		prismaSchema, err = introspectSchema, nil
	}

	if err = CachePrismaSchemaText.Write(dsName, fileloader.SystemUser, []byte(prismaSchema)); err != nil || skipGraphql {
		return
	}

	var prismaSchemaFilepath string
	if len(schemaFilepath) > 0 {
		prismaSchemaFilepath = schemaFilepath[0]
	} else {
		prismaSchemaFilepath = CachePrismaSchemaText.GetPath(dsName)
	}
	if graphqlSchema, err = introspectGraphqlSchema(prismaSchemaFilepath, rpcExt.CmdEnvs...); err != nil {
		return
	}

	cacheGraphqlSchema(dsName, graphqlSchema)
	return
}

// 构建运行时数据源配置，数据库类型数据源/prisma数据源
func buildRuntimeDataSourceConfigurationForPrisma(prismaSchema string, config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration) {
	staticData := &wgpb.DataSourceCustom_Static{Data: &wgpb.ConfigurationVariable{}}
	customDatabase := *config.CustomDatabase
	customDatabase.PrismaSchema, _ = filepath.Abs(prismaSchema)
	configs, fields = copyDatasourceWithRootNodes(config, func(_ *wgpb.TypeField, configItem *wgpb.DataSourceConfiguration) bool {
		configItem.CustomDatabase = &customDatabase
		configItem.CustomStatic = staticData
		return true
	})
	return
}
