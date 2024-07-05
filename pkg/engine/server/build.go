// Package server
/*
 引擎编译功能的实现
 CallBuildResolves调用build包下注册的编译函数，例如数据源、接口、上传、认证等
 CallAsyncGenerates异步调用生成函数，例如swagger、sdk
*/
package server

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	_ "fireboom-server/pkg/engine/sdk"
	_ "fireboom-server/pkg/engine/swagger"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"sync"
)

var EngineBuilder *EngineBuild

func init() {
	utils.RegisterInitMethod(30, func() {
		EngineBuilder = &EngineBuild{logger: zap.L()}
	})
}

type EngineBuild struct {
	logger  *zap.Logger
	builder *build.Builder
}

func (b *EngineBuild) release() {
	build.GeneratedGraphqlConfigRoot.ClearCache()
	if b.builder != nil {
		b.builder.FieldHashes.Clear()
		b.builder = nil
	}
}

// GenerateGraphqlConfig 编译并生成引擎所需配置文件
// group 在build命令下会传入，会等待CallAsyncGenerates全部完成后结束进程
func (b *EngineBuild) GenerateGraphqlConfig(group ...*sync.WaitGroup) (err error) {
	defer func() {
		if err != nil {
			b.logger.Error("build failed", zap.String(consts.EngineStatusField, consts.EngineBuildFailed), zap.Error(err))
		}
	}()

	b.logger.Info("build begin", zap.String(consts.EngineStatusField, consts.EngineBuilding), zap.String("envEffective", configs.EnvEffectiveRoot.GetPath()))
	b.initDefinedApi()
	if err = build.CallRunResolves(b.builder); err != nil {
		return
	}

	if err = b.emitGraphqlConfigCache(); err != nil {
		return
	}

	build.CallAsyncGenerates(b.builder, group...)
	eventbus.EnsureEventSubscribe(b)
	b.logger.Info("build finish", zap.String(consts.EngineStatusField, consts.EngineBuildSucceed))
	return
}

func (b *EngineBuild) initDefinedApi() {
	setting := configs.GlobalSettingRoot.FirstData()
	b.builder = &build.Builder{DefinedApi: &wgpb.UserDefinedApi{
		NodeOptions:           setting.NodeOptions,
		ServerOptions:         setting.ServerOptions,
		CorsConfiguration:     setting.CorsConfiguration,
		EnableGraphqlEndpoint: true,
		AllowedHostNames:      setting.AllowedHostNames,
	}}
}

func (b *EngineBuild) emitGraphqlConfigCache() error {
	graphqlConfig := &wgpb.WunderGraphConfiguration{Api: b.builder.DefinedApi}
	return build.GeneratedGraphqlConfigRoot.InsertOrUpdate(graphqlConfig)
}
