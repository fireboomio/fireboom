// Package build
/*
 编译graphql类型的operation
*/
package build

import (
	"errors"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/pool"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap/buffer"
	"io/fs"
	"path/filepath"
	"strings"
)

type (
	fieldArgumentOverview struct {
		indexes  map[string]int
		required []string
	}
	definitionFieldOverview struct {
		indexes      map[string]int
		required     []string
		inputUniques []string
	}
)

func (o *operations) resolveGraphqlOperation(operation *models.Operation) (operationResult *wgpb.Operation, err error) {
	content, err := models.OperationGraphql.Read(operation.Path)
	// 文本为空（新建operation未创建）且未开启时直接返回
	if content == "" && !operation.Enabled {
		err = nil
		return
	}

	if err != nil {
		return
	}

	// 将片段fragments拼接到末尾并转换成graphql query文档
	content += o.graphqlFragments
	queryItem, err := NewQueryDocumentItem(content)
	if err != nil {
		return
	}

	operationResult = &wgpb.Operation{
		Name:                   normalizeOperationName(operation.Path),
		Path:                   operation.Path,
		RateLimit:              operation.RateLimit,
		Semaphore:              operation.Semaphore,
		Engine:                 wgpb.OperationExecutionEngine_ENGINE_GRAPHQL,
		AuthorizationConfig:    &wgpb.OperationAuthorizationConfig{},
		VariablesConfiguration: &wgpb.OperationVariablesConfiguration{},
		DatasourceQuotes:       make(map[string]*wgpb.DatasourceQuote),
	}

	// 处理graphql文档，修改operationResult并将入参定义保存
	argumentDefinitionsItem := make(openapi3.Schemas)
	queryItem.setResolveParameters(operationResult, argumentDefinitionsItem, o.definitionFetch,
		o.definitionFieldIndexes, o.fieldArgumentIndexes)
	queryItem.resolveOperationList()

	graphqlFile := &GraphqlOperationFile{
		BaseOperationFile: BaseOperationFile{
			VariablesRefs:       queryItem.variablesRefs,
			OperationSchema:     queryItem.operationSchema,
			OperationName:       operationResult.Name,
			FilePath:            models.OperationGraphql.GetPath(operationResult.Path),
			OperationType:       operationResult.OperationType,
			AuthorizationConfig: operationResult.AuthorizationConfig,
		},
		Internal: operationResult.Internal,
	}
	o.operationsConfigData.GraphqlOperationFiles[operationResult.Path] = graphqlFile
	if len(queryItem.Errors) > 0 {
		err = errors.New(strings.Join(queryItem.Errors, ";"))
		return
	}

	// 部分指令和操作会修改文档内容，重写输出文档到文本
	itemBuf := pool.GetBytesBuffer()
	defer pool.PutBytesBuffer(itemBuf)
	if err = queryItem.PrintQueryDocument(itemBuf); err != nil {
		return
	}

	operationResult.Content = itemBuf.String()
	if operation.Enabled {
		// 仅保留入参引用到类型定义
		SearchRefDefinitions(nil, argumentDefinitionsItem, o.operationsConfigData.Definitions, graphqlFile.VariablesRefs...)
	}

	o.mergeGlobalOperation(operation, operationResult)
	o.resolveOperationHook(operationResult)
	return
}

func (o *operations) definitionFetch(name string) *ast.Definition {
	index, ok := o.definitionIndexes[name]
	if !ok {
		return nil
	}

	return o.rootDocument.Definitions[index]
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
func (o *operations) resolveOperationHook(operationItem *wgpb.Operation) {
	hookConfigMap := make(map[string]any)
	for hook, option := range models.GetOperationHookOptions(operationItem.Path) {
		ensureEnabled := option.Enabled && option.Existed
		if hook == consts.MockResolve {
			hookConfigMap[string(hook)] = &wgpb.MockResolveHookConfiguration{Enabled: ensureEnabled}
			continue
		}

		hookConfigMap[string(hook)] = ensureEnabled
	}

	for hook, option := range models.GetHttpTransportHookOptions() {
		hookConfigMap[models.HttpTransportHookAliasMap[hook]] = option.Enabled && option.Existed
	}

	configBytes, _ := json.Marshal(hookConfigMap)
	_ = json.Unmarshal(configBytes, &operationItem.HooksConfiguration)
}

var storeFragmentDirname = utils.NormalizePath(consts.RootStore, consts.StoreFragmentParent)

// 读取fragments，拼接在每个graphql查询后
// 全局的处理片段
func (o *operations) loadFragments() error {
	if utils.NotExistFile(storeFragmentDirname) {
		return nil
	}

	var buf buffer.Buffer
	bufFormatter := formatter.NewFormatter(&buf)
	err := filepath.Walk(storeFragmentDirname, func(path string, info fs.FileInfo, _ error) error {
		if info == nil || info.IsDir() || !strings.HasSuffix(path, string(fileloader.ExtGraphql)) {
			return nil
		}

		content, err := utils.ReadFile(path)
		if err != nil {
			return err
		}

		doc, err := parser.ParseQuery(&ast.Source{Input: string(content)})
		if err != nil {
			return err
		}

		doc.Operations = nil
		bufFormatter.FormatQueryDocument(doc)
		_, _ = buf.WriteString("\n\n")
		return nil
	})
	if err != nil {
		return err
	}

	o.graphqlFragments = buf.String()
	return nil
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
