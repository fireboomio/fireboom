// Package datasource
/*
 实现resolveOpenapi接口，返回graphql文档
*/
package datasource

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/graphql-go-tools/pkg/engine/datasource/httpclient"
	oas_datasource "github.com/wundergraph/wundergraph/pkg/datasources/oas"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"regexp"
	"strings"
)

func newResolveGraphqlSchema(action *actionOpenapi) *resolveGraphqlSchema {
	return &resolveGraphqlSchema{
		dsModelName:  models.DatasourceRoot.GetModelName(),
		dsName:       action.ds.Name,
		schema:       &schema{},
		types:        make(map[string]*definition),
		renamedTypes: make(map[string]bool),
		openapi:      action,
	}
}

type resolveGraphqlSchema struct {
	dsModelName, dsName string
	schema              *schema
	types               map[string]*definition
	renamedTypes        map[string]bool
	openapi             *actionOpenapi
	emptyResolve        *resolveGraphqlSchema
}

func (r *resolveGraphqlSchema) resolve(item *resolveItem) {
	rootItemField := &fieldDefinition{baseDefinition: r.normalizeBaseDefinition(item.operationId, item.description)}
	rootDef, _ := r.fetchDefinition(item.typeName, ast.Object)
	rootDef.Fields = append(rootDef.Fields, rootItemField)

	// 处理返回值
	responseVisitInput := &visitSchemaInput{
		rootKind:       ast.Object,
		definitionName: item.operationId,
		path:           []string{item.operationId},
	}
	if responseType, _, responseTypeFormat := r.visitSchema(responseVisitInput, item.succeedResponse.schema, false); responseType != nil {
		rootItemField.Type = responseType
		addTypeFormatToFieldDescription(&rootItemField.baseDefinition, responseTypeFormat)
	}
	/*for code, content := range item.responses {
		r.visitSchema(content.schema, false, utils.JoinString("_", item.operationId, code), ast.Object)
	}*/

	// 处理requestBody入参
	requestBodyVisitInput := &visitSchemaInput{
		rootKind:       ast.InputObject,
		definitionName: item.operationId,
		path:           []string{item.operationId, httpclient.BODY},
	}
	if inputType, inputDefault, inputTypeFormat := r.visitSchema(requestBodyVisitInput, item.requestBody.schema, true); inputType != nil {
		argDef := &argumentDefinition{
			baseDefinition: baseDefinition{Name: inputType.prototype().string()},
			Type:           inputType,
			DefaultValue:   inputDefault,
		}
		addTypeFormatToFieldDescription(&argDef.baseDefinition, inputTypeFormat)
		rootItemField.Args = append(rootItemField.Args, argDef)
	}

	// 处理form入参
	var parameter *openapi3.Parameter
	for _, itemParameter := range item.parameters {
		if parameter = itemParameter.Value; parameter == nil {
			continue
		}

		parameterVisitInput := &visitSchemaInput{
			rootKind:             ast.InputObject,
			definitionName:       utils.JoinString("_", item.operationId, parameter.Name),
			definitionParentPath: []string{item.operationId},
			path:                 []string{item.operationId, httpclient.QUERYPARAMS, utils.ArrayPath, oas_datasource.FieldNameEqualFlag + parameter.Name},
		}
		itemType, itemDefault, itemTypeFormat := r.visitSchema(parameterVisitInput, parameter.Schema, parameter.Required)
		itemArgDef := &argumentDefinition{
			baseDefinition: r.normalizeBaseDefinition(parameter.Name, parameter.Description),
			Type:           itemType,
			DefaultValue:   itemDefault,
		}
		addTypeFormatToFieldDescription(&itemArgDef.baseDefinition, itemTypeFormat)
		rootItemField.Args = append(rootItemField.Args, itemArgDef)
	}
}

func (r *resolveGraphqlSchema) visitSchema(input *visitSchemaInput, schemaRef *openapi3.SchemaRef, required bool) (result *valueDefinition, defaultValue any, typeFormat string) {
	if schemaRef == nil || schemaRef.Value == nil {
		return
	}

	hasSchemaRefStr := schemaRef.Ref != ""
	if hasSchemaRefStr {
		input = &visitSchemaInput{
			rootKind:             input.rootKind,
			definitionParentPath: input.definitionParentPath,
			definitionName:       buildDefinitionName(schemaRef, input.definitionName),
			path:                 input.path,
		}
	}

	var visitResult visitSchemaOutput
	// 根据index进行排序，按照不同实现执行逻辑
	visitIndexes := maps.Keys(visitSchemaFuncMap)
	slices.Sort(visitIndexes)
	for _, index := range visitIndexes {
		if visitResult = visitSchemaFuncMap[index](r, input, schemaRef.Value, hasSchemaRefStr); len(visitResult.kind) > 0 {
			break
		}
	}

	if visitResult.kind == "" {
		logger.Debug("not supported schema",
			zap.String(r.dsModelName, r.dsName),
			zap.String("definitionName", input.definitionName),
			zap.Any("rootKind", input.rootKind),
			zap.Any("schema", schemaRef.Value))
		visitResult.kind, visitResult.name, visitResult.ofType = ast.Scalar, consts.ScalarString, nil
	}
	if visitResult.kind == ast.Scalar {
		r.fetchDefinition(visitResult.name, ast.Scalar)
	}

	defaultValue, typeFormat = visitResult.defaultValue, visitResult.typeFormat
	result = &valueDefinition{Kind: visitResult.kind, OfType: visitResult.ofType}
	if len(visitResult.name) > 0 {
		result.Name = &visitResult.name
	}
	if required {
		result = &valueDefinition{Kind: KindNonNull, OfType: result}
	}
	return
}

// 格式化定义，当格式化前后名称不同时(存在特殊字符)添加源字段标识到description中
func (r *resolveGraphqlSchema) normalizeBaseDefinition(name, description string, handleIfDiffered ...func(string)) (def baseDefinition) {
	normalized := utils.NormalizeName(name)
	def = baseDefinition{Name: normalized, Description: normalizeDescription(description)}
	if normalized != name {
		def.originName = name
		for _, handle := range handleIfDiffered {
			handle(normalized)
		}
	}
	return
}

func (r *resolveGraphqlSchema) parentRenamed(input *visitSchemaInput) bool {
	return slices.ContainsFunc(input.definitionParentPath, func(name string) bool {
		_, ok := r.renamedTypes[name]
		return ok
	})
}

func (r *resolveGraphqlSchema) fetchExistedDefinition(valueDef *valueDefinition, kind ast.DefinitionKind) (*definition, bool) {
	def, ok := r.types[valueDef.prototype().string()]
	return def, ok && def.Kind == kind
}

// 构建graphql定义
// rest数据源仅存在Query/Mutation两种接口
func (r *resolveGraphqlSchema) fetchDefinition(name string, kind ast.DefinitionKind, appendKindIgnored ...bool) (*definition, bool) {
	name = utils.NormalizeName(name)
	if !ContainsRootDefinition(name) && kind != ast.Scalar && (len(appendKindIgnored) == 0 || !appendKindIgnored[0]) {
		name = utils.JoinString("_", name, strings.ToLower(string(kind)))
		r.renamedTypes[name] = true
	}

	object, ok := r.types[name]
	if !ok {
		baseDef := baseDefinition{Name: name}
		object = &definition{Kind: kind, baseDefinition: baseDef}
		r.types[name] = object
		switch name {
		case consts.TypeQuery:
			r.schema.QueryType = &baseDef
		case consts.TypeMutation:
			r.schema.MutationType = &baseDef
		case consts.TypeSubscription:
			r.schema.SubscriptionType = &baseDef
		}
	}
	return object, ok
}

func (r *resolveGraphqlSchema) storeCustomRestRewriterQuote(rootKind ast.DefinitionKind, quoteObjectName string, path []string) {
	if len(path) < 1 || r.emptyResolve == nil {
		return
	}

	var rewriterMap map[string]*wgpb.DataSourceCustom_REST_Rewriter
	if rootKind == ast.InputObject {
		rewriterMap = r.openapi.customRestRequestRewriterMap
	} else {
		rewriterMap = r.openapi.customRestResponseRewriterMap
	}

	if _, ok := rewriterMap[quoteObjectName]; !ok {
		return
	}

	rewriterQuote := &wgpb.DataSourceRESTRewriter{
		Type:            wgpb.DataSourceRESTRewriterType_quoteObject,
		QuoteObjectName: quoteObjectName,
	}
	if len(path) > 1 {
		rewriterQuote.PathComponents = path[1:]
	}
	r.storeCustomRestRewriter(rootKind, path[0], rewriterQuote)
}

func (r *resolveGraphqlSchema) storeCustomRestRewriter(rootKind ast.DefinitionKind, definitionName string, rewriter *wgpb.DataSourceRESTRewriter) {
	if r.emptyResolve == nil {
		return
	}

	var (
		rewriterMap      map[string]*wgpb.DataSourceCustom_REST_Rewriter
		existedRewriters map[string]bool
	)
	if rootKind == ast.InputObject {
		rewriterMap, existedRewriters = r.openapi.customRestRequestRewriterMap, r.openapi.customRestExistedRequestRewriters
	} else {
		rewriterMap, existedRewriters = r.openapi.customRestResponseRewriterMap, r.openapi.customRestExistedResponseRewriters
	}

	uniqueName := utils.JoinStringWithDot(rewriter.Type.String(), definitionName)
	if len(rewriter.PathComponents) > 0 {
		uniqueName = utils.JoinStringWithDot(uniqueName, utils.JoinStringWithDot(rewriter.PathComponents...))
	}
	if _, ok := existedRewriters[uniqueName]; ok {
		return
	}

	existedRewriters[uniqueName] = true
	if exist, ok := rewriterMap[definitionName]; ok {
		exist.Rewriters = append(exist.Rewriters, rewriter)
	} else {
		rewriterMap[definitionName] = &wgpb.DataSourceCustom_REST_Rewriter{Rewriters: []*wgpb.DataSourceRESTRewriter{rewriter}}
	}
}

const (
	additionalTypeFormat = `<#additionalType#>%s<#additionalType#>`
	typeFormatFormat     = `<#typeFormat#>%s<#typeFormat#>`
)

var (
	additionalTypeRegexp = regexp.MustCompile(`<#additionalType#>([^}]+)<#additionalType#>`)
	typeFormatRegexp     = regexp.MustCompile(`<#typeFormat#>([^}]+)<#typeFormat#>`)
)

func addTypeFormatToFieldDescription(field *baseDefinition, typeFormat string) {
	if len(typeFormat) == 0 {
		return
	}

	field.Description += fmt.Sprintf(typeFormatFormat, typeFormat)
}

// MatchAdditionalType 通过在description中添加的特殊标识匹配出额外字段名称
func MatchAdditionalType(description string) (string, string) {
	return utils.MatchNameWithRegexp(description, additionalTypeRegexp)
}

func MatchTypeFormat(description string) (string, string) {
	return utils.MatchNameWithRegexp(description, typeFormatRegexp)
}
