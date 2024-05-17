// Package build
/*
 读取store/operation配置并转换成引擎所需的配置
 根据executeEngine不同执行不同的编译逻辑
*/
package build

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"math"
	"strings"
	"sync"
)

func init() {
	utils.RegisterInitMethod(30, func() {
		addResolve(2, func() Resolve { return &operations{modelName: models.OperationRoot.GetModelName()} })
	})
}

const (
	buildActionResolve = "resolve"
	buildActionExtract = "extract"
)

type operations struct {
	modelName    string
	rootDocument *ast.SchemaDocument
	fieldHashes  map[string]*LazyFieldHash
	results      []*wgpb.Operation

	graphqlFragments       string
	definitionIndexes      map[string]int
	definitionFieldIndexes map[*ast.Definition]*definitionFieldOverview
	fieldArgumentIndexes   map[*ast.FieldDefinition]*fieldArgumentOverview

	operationsConfigData      *OperationsConfig
	builtOperationsConfigData *OperationsConfig
}

func (o *operations) Resolve(builder *Builder) (err error) {
	o.rootDocument = builder.Document
	o.fieldHashes = builder.FieldHashes
	o.results = o.results[:0]

	if err = o.loadFragments(); err != nil {
		logger.Warn("load fragments failed", zap.Error(err))
		return
	}

	// 将数组处理成以定义名为key的map，方便后续根据名称查询定义
	definitions := o.rootDocument.Definitions
	o.definitionIndexes = make(map[string]int, len(definitions))
	o.definitionFieldIndexes = make(map[*ast.Definition]*definitionFieldOverview)
	o.fieldArgumentIndexes = make(map[*ast.FieldDefinition]*fieldArgumentOverview)
	for i, item := range definitions {
		o.definitionIndexes[item.Name] = i
	}

	datas := models.OperationRoot.List()
	o.operationsConfigData = &OperationsConfig{
		GraphqlOperationFiles:  make(map[string]*GraphqlOperationFile),
		FunctionOperationFiles: make(map[string]*ExtensionOperationFile),
		ProxyOperationFiles:    make(map[string]*ExtensionOperationFile),
		Definitions:            make(openapi3.Schemas, math.MaxInt8),
	}
	o.builtOperationsConfigData = GeneratedOperationsConfigRoot.FirstData()
	if o.builtOperationsConfigData == nil {
		o.builtOperationsConfigData = &OperationsConfig{}
	}
	for _, item := range datas {
		// 根据类型执行operation的编译
		itemResult, succeed, invoked := o.buildOperationItem(item)
		if invoked {
			break
		}

		if succeed {
			o.results = append(o.results, itemResult)
		}
	}

	o.builtOperationsConfigData = nil
	// 将编译结果保存，留作生成swagger合成schema，运行时设置operation属性等
	if err = GeneratedOperationsConfigRoot.InsertOrUpdate(o.operationsConfigData); err != nil {
		return
	}

	builder.DefinedApi.Operations = o.results
	return
}

func (o *operations) extractOperationItem(item *models.Operation) (itemResult *wgpb.Operation, extracted bool) {
	if o.builtOperationsConfigData == nil {
		return
	}

	var defRefs []string
	switch item.Engine {
	case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
		defRefs, extracted = o.extractGraphqlOperation(item,
			o.operationsConfigData.GraphqlOperationFiles, o.builtOperationsConfigData.GraphqlOperationFiles)
	case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
		defRefs, extracted = o.extractExtensionOperation(item, models.OperationFunction,
			o.operationsConfigData.FunctionOperationFiles, o.builtOperationsConfigData.FunctionOperationFiles)
	case wgpb.OperationExecutionEngine_ENGINE_PROXY:
		defRefs, extracted = o.extractExtensionOperation(item, models.OperationProxy,
			o.operationsConfigData.ProxyOperationFiles, o.builtOperationsConfigData.ProxyOperationFiles)
	}
	if extracted && item.Enabled {
		itemResult = models.LoadOperationResult(item.Path)
		SearchRefDefinitions(nil, o.builtOperationsConfigData.Definitions, o.operationsConfigData.Definitions, defRefs...)
	}
	return
}

func (o *operations) resolveOperationItem(item *models.Operation) (itemResult *wgpb.Operation) {
	var err error
	switch item.Engine {
	case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
		itemResult, err = o.resolveGraphqlOperation(item, o.operationsConfigData.GraphqlOperationFiles)
	case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
		itemResult, err = o.resolveExtensionOperation(item, models.OperationFunction, o.operationsConfigData.FunctionOperationFiles)
	case wgpb.OperationExecutionEngine_ENGINE_PROXY:
		itemResult, err = o.resolveExtensionOperation(item, models.OperationProxy, o.operationsConfigData.ProxyOperationFiles)
	}
	if err != nil {
		itemResult = nil
		logger.Warn("build operation failed", zap.Error(err), zap.String(o.modelName, item.Path))
		o.operationsConfigData.Invalids = append(o.operationsConfigData.Invalids, item.Path)
		return
	}
	return
}

func (o *operations) buildOperationItem(item *models.Operation) (itemResult *wgpb.Operation, succeed, invoked bool) {
	// 判断编译数量是否超限制
	if invoked = utils.InvokeFunctionLimit(o.modelName, len(o.results)); invoked {
		return
	}

	buildAction := buildActionExtract
	itemResult, extracted := o.extractOperationItem(item)
	if !extracted {
		itemResult, buildAction = o.resolveOperationItem(item), buildActionResolve
	}
	if succeed = item.Enabled && itemResult != nil; succeed {
		o.mergeGlobalOperation(item, itemResult)
		o.resolveOperationHook(itemResult)
		logger.Debug("build operation succeed", zap.String(o.modelName, item.Path), zap.String("action", buildAction))
	}
	return
}

// 合成全局的配置，根据是否开启自定义配置来合成最终配置
func (o *operations) mergeGlobalOperation(operation *models.Operation, operationResult *wgpb.Operation) {
	globalOperation := models.GlobalOperationRoot.FirstData()
	if operation.ConfigCustomized {
		operationResult.CacheConfig = operation.CacheConfig
		operationResult.LiveQueryConfig = operation.LiveQueryConfig
	} else {
		operationResult.CacheConfig = globalOperation.CacheConfig
		operationResult.LiveQueryConfig = globalOperation.LiveQueryConfig
	}

	if operationResult.AuthenticationConfig != nil {
		return
	}

	if operation.ConfigCustomized {
		operationResult.AuthenticationConfig = operation.AuthenticationConfig
	} else if globalOperation.AuthenticationConfigs != nil {
		operationResult.AuthenticationConfig = globalOperation.AuthenticationConfigs[operationResult.OperationType]
	}
}

// 合成钩子配置和全局钩子配置
func (o *operations) resolveOperationHook(operationResult *wgpb.Operation) {
	hookConfigMap := make(map[string]any)
	if operationResult.Engine == wgpb.OperationExecutionEngine_ENGINE_GRAPHQL {
		for hook, option := range models.GetOperationHookOptions(operationResult.Path) {
			ensureEnabled := option.Enabled && option.Existed
			if hook == consts.MockResolve {
				hookConfigMap[string(hook)] = &wgpb.MockResolveHookConfiguration{Enabled: ensureEnabled}
				continue
			}

			hookConfigMap[string(hook)] = ensureEnabled
		}
	}

	for hook, option := range models.GetHttpTransportHookOptions() {
		hookConfigMap[models.HttpTransportHookAliasMap[hook]] = option.Enabled && option.Existed
	}

	configBytes, _ := json.Marshal(hookConfigMap)
	_ = json.Unmarshal(configBytes, &operationResult.HooksConfiguration)
}

var OperationsDefinitionRwMutex = sync.Mutex{}

type (
	OperationsConfig struct {
		GraphqlOperationFiles  map[string]*GraphqlOperationFile   `json:"graphql_operation_files"`
		FunctionOperationFiles map[string]*ExtensionOperationFile `json:"function_operation_files"`
		ProxyOperationFiles    map[string]*ExtensionOperationFile `json:"proxy_operation_files"`
		Invalids               []string                           `json:"invalids,omitempty"`
		Definitions            openapi3.Schemas                   `json:"definitions"`
	}
	BaseOperationFile struct {
		FilePath            string                             `json:"file_path"`
		OperationName       string                             `json:"operation_name"`
		OperationType       wgpb.OperationType                 `json:"operation_type"`
		AuthorizationConfig *wgpb.OperationAuthorizationConfig `json:"authorization_config"`
		VariablesRefs       []string                           `json:"variables_refs"`
		apihandler.OperationSchema
	}
	GraphqlOperationFile struct {
		BaseOperationFile
		Internal bool `json:"internal"`
	}
	ExtensionOperationFile struct {
		BaseOperationFile
		ModulePath string `json:"module_path"`
	}
)

func normalizeOperationName(path string) string {
	return strings.ReplaceAll(path, "/", "__")
}

// GetOperationsDefinitions 获取operation的出入参数schemas
func GetOperationsDefinitions() (result openapi3.Schemas) {
	return GeneratedOperationsConfigRoot.FirstData().Definitions
}

// SearchRefDefinitions 递归搜索schema定义的引用，减少schema的定义的冗余
func SearchRefDefinitions(schema *openapi3.SchemaRef, searchRefs, requireRefs openapi3.Schemas, refStr ...string) {
	if len(refStr) > 0 {
		var requireSchemas []*openapi3.SchemaRef
		for _, ref := range refStr {
			refSchemaName := strings.TrimPrefix(ref, interpolate.Openapi3SchemaRefPrefix)
			searchSchema := searchRefs[refSchemaName]
			if _, ok := requireRefs[refSchemaName]; !ok && searchSchema != nil {
				requireRefs[refSchemaName] = searchSchema
				requireSchemas = append(requireSchemas, searchSchema)
			}
		}
		if len(requireSchemas) == 0 {
			return
		}
		for _, requireSchema := range requireSchemas {
			SearchRefDefinitions(requireSchema, searchRefs, requireRefs)
		}
		return
	}

	if schema == nil {
		return
	}

	if schema.Ref != "" {
		SearchRefDefinitions(nil, searchRefs, requireRefs, schema.Ref)
		return
	}

	schemaValue := schema.Value
	if schemaValue == nil {
		return
	}

	if schemaValue.Type == openapi3.TypeArray {
		SearchRefDefinitions(schemaValue.Items, searchRefs, requireRefs)
		return
	}

	for _, itemSchema := range schemaValue.Properties {
		SearchRefDefinitions(itemSchema, searchRefs, requireRefs)
	}
}
