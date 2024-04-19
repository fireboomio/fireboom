// Package datasource
/*
 提供prisma引擎的封装
*/
package datasource

import (
	"context"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"go.uber.org/zap"
)

func BuildEngine() *database.Engine {
	return database.NewEngine(&http.Client{Timeout: time.Second * 30}, zap.L(), consts.RootExported)
}

func startQueryEngineWithAction(prismaSchemaFilepath string, action func(*database.Engine, context.Context) error, env ...string) (err error) {
	engine := BuildEngine()
	// 改造成适用绝对路径，解决文本过长出现的命令行too many arguments list报错
	prismaSchemaFilepath, _ = filepath.Abs(prismaSchemaFilepath)
	if err = engine.StartQueryEngine(prismaSchemaFilepath, env...); err != nil {
		return
	}
	defer engine.StopPrismaEngine()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if err = engine.WaitUntilReady(ctx); err != nil {
		return
	}

	err = action(engine, ctx)
	return
}

// introspectGraphqlSchema 内省graphql schema
func introspectGraphqlSchema(prismaSchemaFilepath string, env ...string) (graphqlSchema string, err error) {
	err = startQueryEngineWithAction(prismaSchemaFilepath, func(engine *database.Engine, ctx context.Context) error {
		graphqlSchema, err = engine.IntrospectGraphQLSchema(ctx)
		return err
	}, env...)
	return
}

// IntrospectDMMF 内省DMMF
func IntrospectDMMF(prismaSchemaFilepath string, environmentRequired bool) (dmmfContent string, err error) {
	var envs []string
	if environmentRequired {
		if _, _, env, _ := fetchEnvironmentVariable(prismaSchemaFilepath); env != "" {
			envs = append(envs, env)
		}
	}
	err = startQueryEngineWithAction(prismaSchemaFilepath, func(engine *database.Engine, ctx context.Context) error {
		dmmfContent, err = engine.IntrospectDMMF(ctx)
		return err
	}, envs...)
	return
}

// Migrate 迁移prisma
func Migrate(ctx context.Context, migrateSchema, dataModelFilepath string) (err error) {
	params := database.JsonRPCPayloadParams{
		"force":  true,
		"schema": migrateSchema,
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	_, err = BuildEngine().Migrate(ctx, "schemaPush", params, database.JsonRPCExtension{
		CmdArgs: []string{"--datamodel=" + dataModelFilepath},
	})
	return
}

var envUrlRegexp = regexp.MustCompile(`env\("([^"]+)"\)`)

// 匹配env("aaa")中变量aaa并且在命令行执行参数中追加
// 替换真实的环境变量且保证env变量不在缓存中泄漏
func getCmdEnvironmentVariable(str string) (key string, env string) {
	if matches := envUrlRegexp.FindStringSubmatch(str); len(matches) == 2 {
		key = matches[1]
		env = utils.JoinString("=", key, utils.GetStringWithLockViper(key))
	}
	return
}

func fetchEnvironmentVariable(prismaSchemaFilepath string) (prismaSchema, variable, env string, err error) {
	prismaSchemaFileText, err := utils.ReadWithCondition(prismaSchemaFilepath, nil,
		func(_ int, line string) bool { return strings.HasSuffix(line, "}") })
	if err != nil {
		return
	}

	prismaSchema = utils.JoinString("\n", prismaSchemaFileText...)
	variable, env = getCmdEnvironmentVariable(prismaSchema)
	return
}
