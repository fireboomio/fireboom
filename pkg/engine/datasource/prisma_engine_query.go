package datasource

import (
	"context"
	"fireboom-server/pkg/common/consts"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"go.uber.org/zap"
	"net/http"
	"path/filepath"
	"time"
)

// IntrospectGraphqlSchema 内省graphql schema
func IntrospectGraphqlSchema(engineInput EngineInput) (graphqlSchema string, err error) {
	return startQueryEngineWithAction[string](engineInput, func(ctx context.Context, engine *database.Engine) (string, error) {
		return engine.IntrospectGraphQLSchema(ctx)
	})
}

// IntrospectDMMF 内省DMMF
func IntrospectDMMF(engineInput EngineInput) (dmmfContent string, err error) {
	return startQueryEngineWithAction[string](engineInput, func(ctx context.Context, engine *database.Engine) (string, error) {
		return engine.IntrospectDMMF(ctx)
	})
}

func startQueryEngineWithAction[O any](engineInput EngineInput, action func(context.Context, *database.Engine) (O, error)) (o O, err error) {
	engine := database.NewEngine(&http.Client{Timeout: time.Second * 30}, zap.L(), consts.RootExported)
	// 改造成适用绝对路径，解决文本过长出现的命令行too many arguments list报错
	prismaSchemaFilepath, _ := filepath.Abs(engineInput.PrismaSchemaFilepath)
	if err = engine.StartQueryEngine(prismaSchemaFilepath, buildEnvironments(engineInput)...); err != nil {
		return
	}
	defer engine.StopPrismaEngine()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if err = engine.WaitUntilReady(ctx); err != nil {
		return
	}

	return action(ctx, engine)
}
