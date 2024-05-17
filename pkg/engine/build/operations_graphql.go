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
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
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

func (o *operations) extractGraphqlOperation(operation *models.Operation,
	graphqlFiles, builtGraphqlFiles map[string]*GraphqlOperationFile) (defRefs []string, extracted bool) {
	modifiedTime, _ := models.OperationGraphql.GetModifiedTime(operation.Path)
	defer func() { operation.ContentModifiedTime = modifiedTime }()
	if !modifiedTime.Equal(operation.ContentModifiedTime) || len(operation.SelectedFieldHashes) == 0 {
		return
	}

	for field, hash := range operation.SelectedFieldHashes {
		if value, ok := o.fieldHashes[field]; !ok || value.Hash() != hash {
			return
		}
	}
	graphqlFile, ok := builtGraphqlFiles[operation.Path]
	if !ok {
		return
	}

	graphqlFiles[operation.Path] = graphqlFile
	defRefs, extracted = graphqlFile.VariablesRefs, true
	return
}

func (o *operations) resolveGraphqlOperation(operation *models.Operation, graphqlFiles map[string]*GraphqlOperationFile) (operationResult *wgpb.Operation, err error) {
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
			FilePath:            models.OperationGraphql.GetPath(operation.Path),
			OperationType:       operationResult.OperationType,
			AuthorizationConfig: operationResult.AuthorizationConfig,
		},
		Internal: operationResult.Internal,
	}
	graphqlFiles[operation.Path] = graphqlFile
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

	o.setSelectedFieldHashes(operation, operationResult.DatasourceQuotes)
	return
}

func (o *operations) setSelectedFieldHashes(operation *models.Operation, dsQuotes map[string]*wgpb.DatasourceQuote) {
	if len(o.fieldHashes) == 0 || len(dsQuotes) == 0 {
		return
	}

	operation.SelectedFieldHashes = make(map[string]string)
	for name, quote := range dsQuotes {
		for _, field := range quote.Fields {
			fieldName := utils.JoinString("_", name, field)
			if value, ok := o.fieldHashes[fieldName]; ok {
				operation.SelectedFieldHashes[fieldName] = value.Hash()
			}
		}
	}
}

func (o *operations) definitionFetch(name string) *ast.Definition {
	index, ok := o.definitionIndexes[name]
	if !ok {
		return nil
	}

	return o.rootDocument.Definitions[index]
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

		doc.Operations = doc.Operations[:0]
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
