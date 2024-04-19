// Package build
/*
 读取store/operation配置并转换成引擎所需的配置
 根据executeEngine不同执行不同的编译逻辑
*/
package build

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
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

type operations struct {
	modelName    string
	rootDocument *ast.SchemaDocument
	results      []*wgpb.Operation

	graphqlFragments       string
	definitionIndexes      map[string]int
	definitionFieldIndexes map[*ast.Definition]*definitionFieldOverview
	fieldArgumentIndexes   map[*ast.FieldDefinition]*fieldArgumentOverview
	operationsConfigData   *OperationsConfig
}

func (o *operations) Resolve(builder *Builder) (err error) {
	o.rootDocument = builder.Document
	o.results = nil

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

	// 将编译结果保存，留作生成swagger合成schema，运行时设置operation属性等
	if err = GeneratedOperationsConfigRoot.InsertOrUpdate(o.operationsConfigData); err != nil {
		return
	}

	builder.DefinedApi.Operations = o.results
	return
}

func (o *operations) buildOperationItem(item *models.Operation) (itemResult *wgpb.Operation, succeed, invoked bool) {
	// 判断编译数量是否超限制
	if invoked = utils.InvokeFunctionLimit(o.modelName, len(o.results)); invoked {
		return
	}

	var err error
	switch item.Engine {
	case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
		itemResult, err = o.resolveGraphqlOperation(item)
	case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
		itemResult, err = o.resolveExtensionOperation(func(path string, file *ExtensionOperationFile) {
			o.operationsConfigData.FunctionOperationFiles[path] = file
		}, item, models.OperationFunction)
	case wgpb.OperationExecutionEngine_ENGINE_PROXY:
		itemResult, err = o.resolveExtensionOperation(func(path string, file *ExtensionOperationFile) {
			o.operationsConfigData.ProxyOperationFiles[path] = file
		}, item, models.OperationProxy)
	}
	if err != nil {
		itemResult = nil
		logger.Warn("build operation failed", zap.Error(err), zap.String(o.modelName, item.Path))
		o.operationsConfigData.Invalids = append(o.operationsConfigData.Invalids, item.Path)
		return
	}

	if succeed = item.Enabled && itemResult != nil; succeed {
		logger.Debug("build operation succeed", zap.String(o.modelName, item.Path))
	}
	return
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
