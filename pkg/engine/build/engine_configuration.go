// Package build
/*
 读取store/datasource配置并转换成引擎所需的配置
 根据数据源类型不同选择不同的内省逻辑，并将数据源名称添加到graphql的命名上
*/
package build

import (
	"bufio"
	"crypto/md5"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/engine/directives"
	"fireboom-server/pkg/plugins/fileloader"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/wundergraph/wundergraph/pkg/pool"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"math"
	"os"
	"strings"
)

func init() {
	utils.RegisterInitMethod(30, func() {
		addResolve(1, func() Resolve { return &engineConfiguration{modelName: models.DatasourceRoot.GetModelName()} })
	})
}

type engineConfiguration struct {
	modelName              string
	engineConfig           *wgpb.EngineConfiguration
	typeConfigurationFlags map[string]bool
}

func (e *engineConfiguration) Resolve(builder *Builder) (err error) {
	e.engineConfig = &wgpb.EngineConfiguration{}
	e.typeConfigurationFlags = make(map[string]bool, math.MaxUint8)

	sources := models.DatasourceRoot.ListByCondition(func(item *models.Datasource) bool { return item.Enabled })
	if len(sources) == 0 {
		logger.Warn("empty datasource")
		return
	}

	directiveMap := make(map[string]*ast.DirectiveDefinition)
	rootDefinitionMap, otherDefinitionMap := make(map[string]*ast.Definition), make(map[string]*ast.Definition, math.MaxInt8)
	// 添加根类型Query/Mutation/Subscription
	for _, name := range datasource.RootObjectNames {
		rootDefinitionMap[name] = &ast.Definition{Kind: ast.Object, Name: name}
	}

	var itemAction datasource.Action
	var itemSchema string
	var itemDocument *ast.SchemaDocument
	var itemConfig *wgpb.DataSourceConfiguration
	var itemRename *dataSourceRename
	for _, ds := range sources {
		// 判断数据源数量是否触发限制
		if utils.InvokeFunctionLimit(e.modelName, len(e.engineConfig.DatasourceConfigurations)) {
			break
		}

		// 添加类型获取不同的处理函数及schema文本
		if itemAction, itemSchema, err = e.fetchDatasourceGraphqlSchema(ds); err != nil {
			e.printIntrospectError(err, ds.Name)
			continue
		}

		// 将文本统一转换成一致的文档用作后续处理
		if itemDocument, err = parser.ParseSchema(&ast.Source{Name: ds.Name, Input: itemSchema}); err != nil {
			e.printIntrospectError(err, ds.Name)
			continue
		}

		if extend, ok := itemAction.(datasource.ActionExtend); ok {
			extend.ExtendDocument(itemDocument)
		}

		// 调用不同的函数构建引擎所需配置
		if itemConfig, err = itemAction.BuildDataSourceConfiguration(itemDocument); err != nil || itemConfig == nil {
			e.printIntrospectError(err, ds.Name)
			continue
		}

		// 将数据源名称添加到graphql命名中
		itemRename = newDataSourceRename(ds.Name, itemDocument, e, itemAction)
		itemRename.resolve()

		itemConfig.Id = ds.Name
		itemConfig.Kind = ds.Kind
		itemConfig.KindForPrisma = ds.KindForPrisma
		itemConfig.RootNodes = itemRename.rootNodes
		itemConfig.ChildNodes = itemRename.childNodes
		e.engineConfig.DatasourceConfigurations = append(e.engineConfig.DatasourceConfigurations, itemConfig)

		// 合成graphql文档中的指令
		for _, item := range itemDocument.Directives {
			directiveMap[item.Name] = item
		}
		// 合成文档中的定义，根类型和普通类型作不同处理
		for _, itemDefinition := range itemDocument.Definitions {
			itemDefinitionName := itemDefinition.Name
			if itemRootOperation, ok := itemRename.rootOperationTypeNameMap[itemDefinitionName]; ok {
				itemDefinitionName = itemRootOperation
			}

			if root, ok := rootDefinitionMap[itemDefinitionName]; ok {
				root.Fields = append(root.Fields, itemDefinition.Fields...)
				continue
			}

			otherDefinitionMap[itemDefinitionName] = itemDefinition
		}
		logger.Debug("build datasource succeed", zap.String(e.modelName, ds.Name))
	}

	// 合成自定义指令到文档中
	for _, customDirective := range directives.GetDirectiveSchemas() {
		directiveItem := customDirective.Directive()
		directiveItem.Position = &ast.Position{Src: &ast.Source{}}
		directiveMap[directiveItem.Name] = directiveItem
		for _, definitionItem := range customDirective.Definitions() {
			otherDefinitionMap[definitionItem.Name] = definitionItem
		}
	}

	// 解决graphql文档根类型名称不统一的问题
	var operationTypes ast.OperationTypeDefinitionList
	definitions := make(ast.DefinitionList, 0, len(otherDefinitionMap)+len(datasource.RootObjectNames))
	for _, name := range datasource.RootObjectNames {
		rootDefinition := rootDefinitionMap[name]
		if len(rootDefinition.Fields) == 0 {
			continue
		}

		definitions = append(definitions, rootDefinition)
		operationTypes = append(operationTypes, &ast.OperationTypeDefinition{
			Type: name, Operation: ast.Operation(strings.ToLower(name)),
		})
	}
	// 根据定义类型+名称对普通的定义进行排序
	otherDefinitions := maps.Values(otherDefinitionMap)
	slices.SortFunc(otherDefinitions, func(a, b *ast.Definition) bool {
		aIndex := slices.Index(datasource.DefinitionNameSorts, a.Kind)
		bIndex := slices.Index(datasource.DefinitionNameSorts, b.Kind)
		return aIndex < bIndex || aIndex == bIndex && a.Name < b.Name
	})
	directiveDefinitions := maps.Values(directiveMap)
	slices.SortFunc(directiveDefinitions, func(a, b *ast.DirectiveDefinition) bool { return a.Name < b.Name })
	builder.Document = &ast.SchemaDocument{
		Schema:      ast.SchemaDefinitionList{{OperationTypes: operationTypes}},
		Definitions: append(definitions, otherDefinitions...),
		Directives:  directiveDefinitions,
	}

	// 输出到graphql.schema文件中
	if err = GeneratedGraphqlSchemaText.WriteCustom(GeneratedGraphqlSchemaText.Title, fileloader.SystemUser, func(file *os.File) error {
		return e.writeDocument(builder.Document, file)
	}); err != nil {
		return
	}

	maps.Clear(e.typeConfigurationFlags)
	e.calculateRootFieldHash(builder, otherDefinitionMap)
	builder.DefinedApi.EngineConfiguration = e.engineConfig
	return
}

// 根据数据源类型获取处理逻辑和schema定义
// 当数据源开启缓存或缓存文件时间在上次编译完成后时忽略读取缓存
func (e *engineConfiguration) fetchDatasourceGraphqlSchema(ds *models.Datasource) (action datasource.Action, content string, err error) {
	if action, err = datasource.GetDatasourceAction(ds); err != nil {
		return
	}

	if ds.Kind != wgpb.DataSourceKind_REST {
		var readFromCached bool
		if ds.CacheEnabled {
			readFromCached = true
		} else {
			fileInfo, _ := datasource.CacheGraphqlSchemaText.Stat(ds.Name)
			latestTime := utils.GetTimeWithLockViper(consts.EngineStartTime)
			if fileInfo != nil && !latestTime.IsZero() && latestTime.Before(fileInfo.ModTime()) {
				readFromCached = true
			}
		}

		if readFromCached {
			if content, _ = datasource.CacheGraphqlSchemaText.Read(ds.Name); content != "" {
				return
			}
		}
	}

	content, err = action.Introspect()
	return
}

func (e *engineConfiguration) writeDocument(doc *ast.SchemaDocument, file *os.File) (err error) {
	bufferWriter := bufio.NewWriter(file)
	formatter.NewFormatter(bufferWriter).FormatSchemaDocument(doc)
	err = bufferWriter.Flush()
	return
}

// 将带有入参的graphql查询定义按指定格式记录
// 后续引擎处理graphql响应时需要用到
func (e *engineConfiguration) resolveFieldConfigurations(field *ast.FieldDefinition, fieldRename, typeName string, itemAction datasource.Action) {
	var requiresFields []string
	var argsConfig []*wgpb.ArgumentConfiguration
	for _, arg := range field.Arguments {
		if arg.Type.NonNull {
			// requiresFields = append(requiresFields, arg.Name)
		}
		var renderConfig wgpb.ArgumentRenderConfiguration
		if datasource.IsScalarJsonName(fetchRealType(arg.Type).NamedType) {
			renderConfig = wgpb.ArgumentRenderConfiguration_RENDER_ARGUMENT_AS_JSON_VALUE
		}

		argsConfig = append(argsConfig, &wgpb.ArgumentConfiguration{
			Name:                arg.Name,
			SourceType:          wgpb.ArgumentSource_FIELD_ARGUMENT,
			SourcePath:          []string{},
			RenderConfiguration: renderConfig,
		})
	}
	fieldName := field.Name
	if extend, ok := itemAction.(datasource.ActionExtend); ok {
		fieldName = extend.GetFieldRealName(fieldName)
	}
	fieldConfiguration := &wgpb.FieldConfiguration{
		TypeName:               typeName,
		FieldName:              fieldRename,
		Path:                   []string{fieldName},
		ArgumentsConfiguration: argsConfig,
		RequiresFields:         requiresFields,
	}

	if customField, ok := itemAction.(datasource.FieldConfigurationAction); ok {
		customField.Handle(fieldConfiguration)
	}
	e.engineConfig.FieldConfigurations = append(e.engineConfig.FieldConfigurations, fieldConfiguration)
}

// 将添加了数据源命名的类型按指定格式记录
// 后续引擎处理graphql响应时需要用到
// typeConfigurationFlags 用作去重复判断
func (e *engineConfiguration) resolveTypeConfigurations(typeName, originName string) {
	if _, ok := e.typeConfigurationFlags[typeName]; ok {
		return
	}

	e.typeConfigurationFlags[typeName] = true
	e.engineConfig.TypeConfigurations = append(e.engineConfig.TypeConfigurations, &wgpb.TypeConfiguration{
		TypeName: typeName,
		RenameTo: originName,
	})
}

func (e *engineConfiguration) printIntrospectError(err error, dsName string) {
	logger.Error("build datasource failed", zap.Error(err), zap.String(e.modelName, dsName))
}

func (e *engineConfiguration) calculateRootFieldHash(builder *Builder, fieldDefinitions map[string]*ast.Definition) {
	if !utils.GetBoolWithLockViper(consts.DevMode) {
		return
	}

	builder.FieldHashes = make(map[string]*LazyFieldHash, math.MaxUint8)
	for dsIndex := range e.engineConfig.DatasourceConfigurations {
		dsConfig := e.engineConfig.DatasourceConfigurations[dsIndex]
		for nodeIndex := range dsConfig.RootNodes {
			rootNode := dsConfig.RootNodes[nodeIndex]
			for i := range rootNode.FieldNames {
				fieldIndex, fieldName := i, rootNode.FieldNames[i]
				builder.FieldHashes[fieldName] = &LazyFieldHash{
					lazyFunc: func() string {
						buf := pool.GetBytesBuffer()
						defer pool.PutBytesBuffer(buf)
						document := &ast.SchemaDocument{}
						format := formatter.NewFormatter(buf, formatter.WithIndent(""))
						quotes, ok := rootNode.Quotes[int32(fieldIndex)]
						if ok {
							for _, i := range quotes.Indexes {
								document.Definitions = append(document.Definitions, fieldDefinitions[dsConfig.ChildNodes[i].TypeName])
							}
						}
						format.FormatSchemaDocument(document)
						buf.WriteString(fieldName)
						return fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
					},
				}
			}
		}
	}
}
