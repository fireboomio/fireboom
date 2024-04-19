// Package server
/*
 引擎启动调用功能的实现
 将生成的所有文件组装成引擎启动所需的配置文件

*/
package server

import (
	"context"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/node"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// 1. 全量内省启动[已完成]
// 2. datasource变更启动(重新内省)
// 3. operation变更启动(重新编译operation)[已完成]
// 4. storage变更启动(重新配置s3)[已完成]
// 5. sdk变更启动(重新生成模板)[已完成]
// 6. role变更启动(重新生成rbac指令枚举)
// 7. globalSetting配置变更启动
// 8. globalOperation配置变更启动

var EngineStarter *EngineStart

func init() {
	utils.RegisterInitMethod(30, func() {
		logger := zap.L()
		ctx := context.WithValue(context.Background(), consts.HeaderParamTag, utils.RandomIdentifyCode)
		EngineStarter = &EngineStart{
			logger:              logger,
			datasourceModelName: models.DatasourceRoot.GetModelName(),
			nodeServer:          node.New(ctx, node.BuildInfo{}, consts.RootExported, logger),
		}
		if !utils.GetBoolWithLockViper(consts.DevMode) {
			return
		}

		engineStarterMutex := &sync.Mutex{}
		utils.BuildAndStart = func() {
			if !engineStarterMutex.TryLock() {
				return
			}

			EngineBuilder.release()
			EngineStarter.release()
			if err := EngineBuilder.GenerateGraphqlConfig(); err != nil {
				return
			}

			EngineStarter.StartNodeServer(engineStarterMutex)
		}
		utils.CallBuildAndStartFuncWatchers()
	})
}

type EngineStart struct {
	logger     *zap.Logger
	nodeConfig *node.WunderNodeConfig
	nodeServer *node.Node

	datasourceModelName string
}

func (s *EngineStart) release() {
	s.nodeConfig = nil
	_ = s.nodeServer.Close()
}

func Shutdown() {
	_ = EngineStarter.nodeServer.Close()
}

// StartNodeServer 引擎启动
// mutex 热重启引擎是会传入，加锁防止引擎启动完成前重新调用
func (s *EngineStart) StartNodeServer(mutex ...*sync.Mutex) {
	var err error
	defer func() {
		if err != nil {
			s.logger.Error("start failed", zap.String(consts.EngineStatusField, consts.EngineStartFailed), zap.Error(err))
		}
	}()

	startingTime := time.Now()
	s.logger.Info("start begin", zap.String(consts.EngineStatusField, consts.EngineStarting))
	if err = s.fetchNodeConfig(); err != nil {
		return
	}

	setting := configs.GlobalSettingRoot.FirstData()
	if setting.BuildInfo != nil {
		s.nodeServer.UpdateBuildInfo(*setting.BuildInfo)
	}
	nodeOpts := []node.Option{
		node.WithIntrospection(true),
		node.WithStaticWunderNodeConfig(*s.nodeConfig),
		node.WithCSRFProtect(setting.EnableCSRFProtect),
		node.WithForceHttpsRedirects(setting.ForceHttpsRedirects),
		node.WithStartedHandler(func() {
			if len(mutex) > 0 {
				defer mutex[0].Unlock()
			}
			eventbus.EnsureEventBreakData(s)
			zapFields := []zap.Field{zap.String(consts.EngineStatusField, consts.EngineStartSucceed)}
			if utils.GetBoolWithLockViper(consts.DevMode) {
				// dev模式添加编译耗时统计
				prepareTime := utils.GetTimeWithLockViper(consts.EnginePrepareTime)
				zapFields = append(zapFields, zap.Duration("buildCost", startingTime.Sub(prepareTime)))
			}
			zapFields = append(zapFields, zap.Duration("startCost", time.Since(startingTime)))
			debug.FreeOSMemory()
			s.logger.Info("start finish", zapFields...)
		}),
	}
	if rateLimit := setting.GlobalRateLimit; rateLimit != nil && rateLimit.Enabled {
		nodeOpts = append(nodeOpts, node.WithGlobalRateLimit(int(rateLimit.Requests), time.Duration(rateLimit.PerSecond)*time.Second))
	}
	if utils.GetBoolWithLockViper(consts.DevMode) {
		nodeOpts = append(nodeOpts, node.WithDevMode())
		nodeOpts = append(nodeOpts, node.WithDebugMode(true))
	}
	if nodeServerUrl := s.nodeConfig.Api.Options.PublicNodeUrl; !strings.HasPrefix(nodeServerUrl, "https") {
		nodeOpts = append(nodeOpts, node.WithInsecureCookies())
	}

	eventbus.EnsureEventSubscribe(s)
	err = s.nodeServer.StartServer(nodeOpts...)
	return
}

func (s *EngineStart) fetchNodeConfig() (err error) {
	schemaText := build.GeneratedGraphqlSchemaText
	graphqlSchema, err := schemaText.Read(schemaText.Title)
	if err != nil {
		err = i18n.NewCustomError(err, i18n.EngineCreateConfigError)
		return
	}

	generateConfig := build.GeneratedGraphqlConfigRoot.FirstData()
	nodeConfig, err := node.CreateConfig(generateConfig)
	if err != nil {
		err = i18n.NewCustomError(err, i18n.EngineCreateConfigError)
		return
	}

	generateEngineConfig := generateConfig.Api.EngineConfiguration
	nodeConfig.Api.EngineConfiguration = &wgpb.EngineConfiguration{
		GraphqlSchema:        graphqlSchema,
		DefaultFlushInterval: generateEngineConfig.DefaultFlushInterval,
		TypeConfigurations:   generateEngineConfig.TypeConfigurations,
		FieldConfigurations:  make([]*wgpb.FieldConfiguration, len(generateEngineConfig.FieldConfigurations)),
	}
	copy(nodeConfig.Api.EngineConfiguration.FieldConfigurations, generateEngineConfig.FieldConfigurations)
	s.nodeConfig = &nodeConfig

	s.runtimeDataSourceConfigurations(generateEngineConfig.DatasourceConfigurations)
	s.runtimeOperations()
	s.runtimeInvalidOperations()

	s.logger.Debug("fetch node configuration succeed")
	return
}

// 构建运行时数据源配置
// 在编译过程中通过下标记录来每个查询所依赖的子字段
// 引擎所需的数据源需要将每个查询都拆成一个数据源
func (s *EngineStart) runtimeDataSourceConfigurations(datasourceConfigurations []*wgpb.DataSourceConfiguration) {
	var err error
	var itemDsAction datasource.Action
	var itemRuntimeConfigs []*wgpb.DataSourceConfiguration
	var itemRuntimeFields []*wgpb.FieldConfiguration
	runtimeEngineConfig := s.nodeConfig.Api.EngineConfiguration
	for _, dsConfig := range datasourceConfigurations {
		if itemDsAction, err = datasource.GetDatasourceAction(&models.Datasource{Name: dsConfig.Id, Kind: dsConfig.Kind}); err != nil {
			s.printSetSchemaError(err, dsConfig.Id)
			continue
		}

		if itemRuntimeConfigs, itemRuntimeFields, err = itemDsAction.RuntimeDataSourceConfiguration(dsConfig); err != nil {
			s.printSetSchemaError(err, dsConfig.Id)
			continue
		}

		runtimeEngineConfig.DatasourceConfigurations = append(runtimeEngineConfig.DatasourceConfigurations, itemRuntimeConfigs...)
		runtimeEngineConfig.FieldConfigurations = append(runtimeEngineConfig.FieldConfigurations, itemRuntimeFields...)
	}
	s.logger.Debug("build runtime datasource configurations succeed")
}

// 运行时修改内存中operation配置(未编译的)，用作前端展示和admin查询
func (s *EngineStart) setStoreOperationByRuntimeData(path string, operation *wgpb.Operation, fileItem *build.BaseOperationFile, hook ...func(*models.Operation)) {
	itemData, _ := models.OperationRoot.GetByDataName(path)
	if itemData != nil {
		models.StoreOperationResult(path, operation)
		itemData.Invalid = false
		itemData.OperationType = fileItem.OperationType
		itemData.AuthorizationConfig = fileItem.AuthorizationConfig
		for _, f := range hook {
			f(itemData)
		}
	}
}

// 编译生成的文件有大量的下标引用来缩减配置文件大小
// 启动引擎时需要将所有引用还原成真实配置
func (s *EngineStart) runtimeOperations() {
	s.nodeConfig.Api.OperationSchemas = make(map[string]*apihandler.OperationSchema, len(s.nodeConfig.Api.Operations))
	operationsConfig := build.GeneratedOperationsConfigRoot.FirstData()
	if operationsConfig == nil {
		return
	}
	for _, operation := range s.nodeConfig.Api.Operations {
		s.runtimeOperationItem(operationsConfig, operation)
	}

	s.logger.Debug("build runtime operations succeed")
}

func (s *EngineStart) runtimeOperationItem(operationsConfig *build.OperationsConfig, operation *wgpb.Operation) {
	var baseFile build.BaseOperationFile
	var optional []func(*models.Operation)
	operationPath := operation.Path
	switch operation.Engine {
	case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
		fileItem, ok := operationsConfig.FunctionOperationFiles[operationPath]
		if !ok {
			return
		}
		baseFile = fileItem.BaseOperationFile
	case wgpb.OperationExecutionEngine_ENGINE_PROXY:
		fileItem, ok := operationsConfig.ProxyOperationFiles[operationPath]
		if !ok {
			return
		}
		baseFile = fileItem.BaseOperationFile
	case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
		fileItem, ok := operationsConfig.GraphqlOperationFiles[operationPath]
		if !ok {
			return
		}
		baseFile = fileItem.BaseOperationFile
		optional = append(optional, func(itemData *models.Operation) { itemData.Internal = fileItem.Internal })
	}
	s.setStoreOperationByRuntimeData(operationPath, operation, &baseFile, optional...)

	variablesDefs := make(openapi3.Schemas)
	build.OperationsDefinitionRwMutex.Lock()
	build.SearchRefDefinitions(nil, operationsConfig.Definitions, variablesDefs, baseFile.VariablesRefs...)
	build.OperationsDefinitionRwMutex.Unlock()
	s.nodeConfig.Api.OperationSchemas[operationPath] = &apihandler.OperationSchema{
		Variables:         baseFile.Variables,
		InternalVariables: baseFile.InternalVariables,
		Response:          baseFile.Response,
		Definitions:       variablesDefs,
	}
}

// 设置失效的operation
func (s *EngineStart) runtimeInvalidOperations() {
	invalidOperationNames := build.GeneratedOperationsConfigRoot.FirstData().Invalids
	for _, itemPath := range invalidOperationNames {
		if itemData, _ := models.OperationRoot.GetByDataName(itemPath); itemData != nil {
			itemData.Invalid = true
		}
	}
	s.nodeConfig.Api.InvalidOperationNames = invalidOperationNames
	s.logger.Debug("build runtime invalid operations succeed")
}

func (s *EngineStart) printSetSchemaError(err error, dsName string) {
	s.logger.Warn("set datasource graphql schema failed", zap.Error(err), zap.String(s.datasourceModelName, dsName))
}
