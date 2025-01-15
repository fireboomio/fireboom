package datasource

import (
	"context"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"strings"
	"time"
)

func buildDatasourceContext(ctx context.Context, dsName string) (context.Context, func()) {
	ctx = context.WithValue(ctx, eventbus.ChannelDatasource, dsName)
	return context.WithTimeout(ctx, time.Second*30)
}

func Introspect(ctx context.Context, engineInput EngineInput, dsName string) (prismaSchema string, err error) {
	introspectCmd := database.NewIntrospectCmd(zap.L())
	introspectInput := &database.IntrospectInput{Schema: engineInput.PrismaSchema, CompositeTypeDepth: -1}
	rpcExt := database.SchemaCommandRPCExtension{CmdEnvs: buildEnvironments(engineInput)}
	ctx, cancel := buildDatasourceContext(ctx, dsName)
	defer cancel()
	introspectOutput, _, err := introspectCmd.Run(ctx, introspectInput, rpcExt)
	if err != nil {
		if !strings.Contains(err.Error(), ignoreEmptyDatabaseError) {
			return
		}

		prismaSchema, err = engineInput.PrismaSchema, nil
	} else {
		prismaSchema = introspectOutput.Datamodel
	}
	return
}

// SchemaPush 推送prisma
func SchemaPush(ctx context.Context, engineInput EngineInput, dsName string) (err error) {
	schemaPushCmd := database.NewSchemaPushCmd(zap.L())
	schemaPushInput := &database.SchemaPushInput{
		Force:  true,
		Schema: engineInput.PrismaSchema,
	}
	rpcExt := database.SchemaCommandRPCExtension{
		CmdEnvs: buildEnvironments(engineInput),
		CmdArgs: []string{"--datamodel=" + engineInput.PrismaSchemaFilepath},
	}
	ctx, cancel := buildDatasourceContext(ctx, dsName)
	defer cancel()
	_, _, err = schemaPushCmd.Run(ctx, schemaPushInput, rpcExt)
	return
}

// CreateMigration 创建迁移文件
func CreateMigration(ctx context.Context, engineInput EngineInput, dsName, version string) (generatedMigrationName string, err error) {
	createMigrationCmd := database.NewCreateMigrationCmd(zap.L())
	createMigrationInput := &database.CreateMigrationInput{
		Draft:                   true,
		MigrationName:           version,
		MigrationsDirectoryPath: utils.NormalizePath(migrationDirname, dsName),
		PrismaSchema:            engineInput.PrismaSchema,
	}
	rpcExt := database.SchemaCommandRPCExtension{
		CmdEnvs: buildEnvironments(engineInput),
		CmdArgs: []string{"--datamodel=" + engineInput.PrismaSchemaFilepath},
	}
	ctx, cancel := buildDatasourceContext(ctx, dsName)
	defer cancel()
	createMigrationOutput, _, err := createMigrationCmd.Run(ctx, createMigrationInput, rpcExt)
	if err != nil {
		return
	}
	generatedMigrationName = createMigrationOutput.GeneratedMigrationName
	return
}

// ApplyMigration 创建迁移文件 TODO 暂未实验成功
func ApplyMigration(ctx context.Context, engineInput EngineInput, dsName string) (appliedMigrationNames []string, err error) {
	applyMigrationCmd := database.NewApplyMigrationsCmd(zap.L())
	applyMigrationInput := &database.ApplyMigrationsInput{
		MigrationsDirectoryPath: utils.NormalizePath(migrationDirname, dsName),
	}
	rpcExt := database.SchemaCommandRPCExtension{
		CmdEnvs: buildEnvironments(engineInput),
		CmdArgs: []string{"--datamodel=" + engineInput.PrismaSchemaFilepath},
	}
	ctx, cancel := buildDatasourceContext(ctx, dsName)
	defer cancel()
	applyMigrationOutput, printContent, err := applyMigrationCmd.Run(ctx, applyMigrationInput, rpcExt)
	if err != nil {
		return
	}
	if len(printContent) > 0 {
		zap.L().Info("applyMigrationPrint", zap.String("content", printContent))
	}
	if applyMigrationOutput != nil {
		appliedMigrationNames = applyMigrationOutput.AppliedMigrationNames
	}
	return
}

type (
	DiffCmdOption struct {
		Type        DiffCmdOptionType           `json:"type"`
		DatabaseUrl *wgpb.ConfigurationVariable `json:"databaseUrl"`
	}
	DiffCmdOptionType string
)

const (
	DiffCmdOptionMigrationsToDev DiffCmdOptionType = "migrations_to_dev"
	DiffCmdOptionDevToProd       DiffCmdOptionType = "dev_to_prod"
)

// Diff 比较不同
func Diff(ctx context.Context, engineInput EngineInput, dsName string, diffOption *DiffCmdOption) (content string, err error) {
	diffCmd := database.NewDiffCmd(zap.L())
	diffInput := &database.DiffInput{ExitCode: true, Script: true}
	schemaDataModelTarget := database.DiffInputTarget{
		SchemaDatamodel: &database.SchemaContainer{Schema: engineInput.PrismaSchemaFilepath},
	}
	switch diffOption.Type {
	case DiffCmdOptionMigrationsToDev:
		if diffOption.DatabaseUrl == nil {
			err = i18n.NewCustomError(nil, i18n.PrismaShadowDatabaseUrlEmptyError)
			return
		}
		diffInput.ShadowDatabaseUrl = utils.GetVariableString(diffOption.DatabaseUrl)
		diffInput.From = database.DiffInputTarget{
			Migrations: &database.PathContainer{Path: utils.NormalizePath(migrationDirname, dsName)},
		}
		diffInput.To = schemaDataModelTarget
	case DiffCmdOptionDevToProd:
		if diffOption.DatabaseUrl == nil {
			err = i18n.NewCustomError(nil, i18n.DatasourceDatabaseUrlEmptyError)
			return
		}
		diffInput.From = schemaDataModelTarget
		diffInput.To = database.DiffInputTarget{
			URL: &database.UrlContainer{Url: utils.GetVariableString(diffOption.DatabaseUrl)},
		}
	}
	rpcExt := database.SchemaCommandRPCExtension{CmdEnvs: buildEnvironments(engineInput)}
	ctx, cancel := buildDatasourceContext(ctx, dsName)
	defer cancel()
	_, content, err = diffCmd.Run(ctx, diffInput, rpcExt)
	return
}
