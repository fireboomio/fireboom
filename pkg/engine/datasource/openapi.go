// Package datasource
/*
 Rest类型数据源的实现
*/
package datasource

import (
	"context"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ghodss/yaml"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/customhttpclient"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"path/filepath"
	"strings"
)

func init() {
	actionMap[wgpb.DataSourceKind_REST] = func(ds *models.Datasource, _ string) Action {
		return &actionOpenapi{
			ds:                                 ds,
			customRestExistedRequestRewriters:  make(map[string]bool),
			customRestExistedResponseRewriters: make(map[string]bool),
			customRestRequestRewriterMap:       make(map[string]*wgpb.DataSourceCustom_REST_Rewriter),
			customRestResponseRewriterMap:      make(map[string]*wgpb.DataSourceCustom_REST_Rewriter),
		}
	}
}

var (
	versionKeys    = []string{"swagger", "openapi"}
	yamlExtensions = []fileloader.Extension{fileloader.ExtYaml, fileloader.ExtYml}
	mimeTypes      = []string{
		echo.MIMEApplicationJSONCharsetUTF8,
		echo.MIMEApplicationJavaScript,
		echo.MIMEApplicationJavaScriptCharsetUTF8,
		echo.MIMEApplicationXML,
		echo.MIMEApplicationXMLCharsetUTF8,
		echo.MIMETextXML,
		echo.MIMETextXMLCharsetUTF8,
		echo.MIMEApplicationForm,
		echo.MIMEApplicationProtobuf,
		echo.MIMEApplicationMsgpack,
		echo.MIMETextHTML,
		echo.MIMETextHTMLCharsetUTF8,
		echo.MIMETextPlain,
		echo.MIMETextPlainCharsetUTF8,
		echo.MIMEMultipartForm,
		echo.MIMEOctetStream,
		customhttpclient.TextEventStreamMine,
		"*/*",
	}
)

type (
	actionOpenapi struct {
		ds  *models.Datasource
		doc *openapi3.T

		customRestExistedRequestRewriters  map[string]bool
		customRestExistedResponseRewriters map[string]bool
		customRestRequestRewriterMap       map[string]*wgpb.DataSourceCustom_REST_Rewriter
		customRestResponseRewriterMap      map[string]*wgpb.DataSourceCustom_REST_Rewriter
	}
	resolveOpenapi interface {
		resolve(*resolveItem)
	}
	contentSchema struct {
		contentType string
		schema      *openapi3.SchemaRef
	}
	resolveItem struct {
		typeName        string
		operationId     string
		description     string
		path            string
		method          wgpb.HTTPMethod
		parameters      openapi3.Parameters
		requestBody     contentSchema
		responses       map[string]contentSchema
		succeedResponse contentSchema
		sseDataData     string
	}
)

// Handle 实现了FieldConfigurationAction接口，自定义处理字添加逻辑
func (a *actionOpenapi) Handle(configuration *wgpb.FieldConfiguration) {
	configuration.DisableDefaultFieldMapping = true
	configuration.ArgumentsConfiguration = nil
	configuration.Path = nil
	configuration.RequiresFields = nil
}

func (a *actionOpenapi) Introspect() (graphqlSchema string, err error) {
	resolveSchema := newResolveGraphqlSchema(a)
	resolveSchema.emptyResolve = newResolveGraphqlSchema(a)
	if err = a.resolveDocument(resolveSchema); err != nil {
		return
	}

	resolveSchema.schema.Types = maps.Values(resolveSchema.types)
	graphqlSchema = formatSchemaString(resolveSchema.schema)
	cacheGraphqlSchema(a.ds.Name, graphqlSchema)
	return
}

func (a *actionOpenapi) BuildDataSourceConfiguration(*ast.SchemaDocument) (config *wgpb.DataSourceConfiguration, err error) {
	resolveConfig := newResolveDatasourceConfiguration(a)
	if err = a.resolveDocument(resolveConfig); err != nil {
		return
	}

	config = &wgpb.DataSourceConfiguration{
		CustomRestMap:                 resolveConfig.customRestMap,
		CustomRestRequestRewriterMap:  a.customRestRequestRewriterMap,
		CustomRestResponseRewriterMap: a.customRestResponseRewriterMap,
	}
	return
}

func (a *actionOpenapi) RuntimeDataSourceConfiguration(config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration, err error) {
	customRestMap := config.CustomRestMap
	requestRewriterMap := config.CustomRestRequestRewriterMap
	responseRewriterMap := config.CustomRestResponseRewriterMap
	configs, fields = copyDatasourceWithRootNodes(config, func(rootItem *wgpb.TypeField, configItem *wgpb.DataSourceConfiguration) bool {
		existedCustomRest, ok := customRestMap[utils.JoinStringWithDot(rootItem.TypeName, rootItem.FieldNames[0])]
		if ok {
			configItem.ChildNodes = nil
			rootFieldOriginName := strings.TrimPrefix(rootItem.FieldNames[0], config.Id+"_")
			requestRewriters := a.searchRewriters(rootFieldOriginName, requestRewriterMap)
			responseRewriters := a.searchRewriters(rootFieldOriginName, responseRewriterMap)
			if len(requestRewriters) == 0 && len(responseRewriters) == 0 {
				configItem.CustomRest = existedCustomRest
			} else {
				a.sortRewriters(requestRewriters)
				a.sortRewriters(responseRewriters)
				configItem.CustomRest = &wgpb.DataSourceCustom_REST{
					Fetch:                  existedCustomRest.Fetch,
					Subscription:           existedCustomRest.Subscription,
					StatusCodeTypeMappings: existedCustomRest.StatusCodeTypeMappings,
					DefaultTypeName:        existedCustomRest.DefaultTypeName,
					RequestRewriters:       requestRewriters,
					ResponseRewriters:      responseRewriters,
				}
			}
		}
		return ok
	})
	return
}

func (a *actionOpenapi) sortRewriters(rewriters []*wgpb.DataSourceRESTRewriter) {
	slices.SortFunc(rewriters, func(a, b *wgpb.DataSourceRESTRewriter) bool {
		pathLengthA, pathLengthB := len(a.PathComponents), len(b.PathComponents)
		return pathLengthA > pathLengthB || pathLengthA == pathLengthB && a.Type < b.Type
	})
}

func (a *actionOpenapi) searchRewriters(objectName string, rewriterMap map[string]*wgpb.DataSourceCustom_REST_Rewriter, path ...string) (rewriters []*wgpb.DataSourceRESTRewriter) {
	restWriter, ok := rewriterMap[objectName]
	if !ok {
		return
	}

	for _, item := range restWriter.Rewriters {
		if item.Type != wgpb.DataSourceRESTRewriterType_quoteObject {
			if len(path) > 0 {
				originPathComponents := item.PathComponents
				item = &wgpb.DataSourceRESTRewriter{
					Type:                      item.Type,
					QuoteObjectName:           item.QuoteObjectName,
					FieldRewriteTo:            item.FieldRewriteTo,
					ValueRewrites:             item.ValueRewrites,
					CustomObjectName:          item.CustomObjectName,
					CustomEnumField:           item.CustomEnumField,
					ApplySubCommonField:       item.ApplySubCommonField,
					ApplySubCommonFieldValues: item.ApplySubCommonFieldValues,
					ApplySubObjects:           item.ApplySubObjects,
					ApplySubFieldTypes:        item.ApplySubFieldTypes,
				}
				item.PathComponents = append(item.PathComponents, path...)
				item.PathComponents = append(item.PathComponents, originPathComponents...)
			}
			rewriters = append(rewriters, item)
		} else {
			rewriters = append(rewriters, a.searchRewriters(item.QuoteObjectName, rewriterMap, item.PathComponents...)...)
		}
	}
	return
}

// 处理rest数据源依赖的文本，支持.yaml, .yml, .json等类型文件
// 添加了对openapi2和3的支持
func (a *actionOpenapi) fetchDocument() (err error) {
	oasFilepath := models.DatasourceUploadOas.GetPath(a.ds.Name)
	oasBytes, err := utils.ReadFileAsUTF8(oasFilepath)
	if err != nil {
		return
	}

	ext := fileloader.Extension(filepath.Ext(oasFilepath))
	if slices.Contains(yamlExtensions, ext) {
		if oasBytes, err = yaml.YAMLToJSON(oasBytes); err != nil {
			return
		}
	}

	versionResults := gjson.GetManyBytes(oasBytes, versionKeys...)
	var version string
	for _, result := range versionResults {
		if result.Type == gjson.Null {
			continue
		}

		version = result.String()
	}

	switch {
	case strings.HasPrefix(version, "2."):
		var openapi2Doc openapi2.T
		if err = json.Unmarshal(oasBytes, &openapi2Doc); err != nil {
			return
		}

		if a.doc, err = openapi2conv.ToV3(&openapi2Doc); err != nil {
			return
		}
	case strings.HasPrefix(version, "3."):
		if err = json.Unmarshal(oasBytes, &a.doc); err != nil {
			return
		}
	default:
		return i18n.NewCustomErrorWithMode(datasourceModelName, nil, i18n.DatabaseOasVersionError, version)
	}

	if err = openapi3.NewLoader().ResolveRefsIn(a.doc, nil); err != nil {
		return
	}

	return a.doc.Validate(context.Background(), openapi3.DisableExamplesValidation(), openapi3.DisableSchemaDefaultsValidation())
}

// 解析文档，主要处理paths，其中resolve为不同的具体处理实现
func (a *actionOpenapi) resolveDocument(resolve resolveOpenapi) (err error) {
	if a.doc == nil {
		if err = a.fetchDocument(); err != nil {
			return
		}
	}

	for path, pathItem := range a.doc.Paths {
		a.iteratorPathItem(path, pathItem.Connect, pathItem.Parameters, wgpb.HTTPMethod_CONNECT, consts.TypeQuery, resolve)
		a.iteratorPathItem(path, pathItem.Head, pathItem.Parameters, wgpb.HTTPMethod_HEAD, consts.TypeQuery, resolve)
		a.iteratorPathItem(path, pathItem.Patch, pathItem.Parameters, wgpb.HTTPMethod_PATCH, consts.TypeQuery, resolve)
		a.iteratorPathItem(path, pathItem.Trace, pathItem.Parameters, wgpb.HTTPMethod_TRACE, consts.TypeQuery, resolve)
		a.iteratorPathItem(path, pathItem.Get, pathItem.Parameters, wgpb.HTTPMethod_GET, consts.TypeQuery, resolve)

		a.iteratorPathItem(path, pathItem.Options, pathItem.Parameters, wgpb.HTTPMethod_OPTIONS, consts.TypeMutation, resolve)
		a.iteratorPathItem(path, pathItem.Post, pathItem.Parameters, wgpb.HTTPMethod_POST, consts.TypeMutation, resolve)
		a.iteratorPathItem(path, pathItem.Delete, pathItem.Parameters, wgpb.HTTPMethod_DELETE, consts.TypeMutation, resolve)
		a.iteratorPathItem(path, pathItem.Put, pathItem.Parameters, wgpb.HTTPMethod_PUT, consts.TypeMutation, resolve)
	}
	return
}

// 具体处理operation的请求定义
// 处理成引擎所需的数据源配置/graphql文档
func (a *actionOpenapi) iteratorPathItem(path string, operation *openapi3.Operation, parameters openapi3.Parameters, method wgpb.HTTPMethod, typeName string, resolve resolveOpenapi) *resolveItem {
	if operation == nil {
		return nil
	}

	resolveInfo := &resolveItem{
		typeName: typeName,
		path:     path,
		method:   method,
	}
	resolveInfo.responses = make(map[string]contentSchema)
	var (
		operationMatchMineTypes     []string
		operationOnlyEventStream    bool
		operationContainEventStream bool
	)
	operationIdSuffix := method.String()
	if typeName == consts.TypeSubscription {
		operationIdSuffix += typeName
		operationMatchMineTypes = append(operationMatchMineTypes, customhttpclient.TextEventStreamMine)
	}
	for code, resp := range operation.Responses {
		extension, respSchema, contentSize := a.fetchResponseResolveSchema(resp.Value, operationMatchMineTypes...)
		if code == "200" {
			resolveInfo.succeedResponse = respSchema
			if contentSize > 0 {
				_, operationContainEventStream = resp.Value.Content[customhttpclient.TextEventStreamMine]
			}
			operationOnlyEventStream = operationContainEventStream && contentSize == 1
			if operationContainEventStream && extension != nil {
				resolveInfo.sseDataData = cast.ToString(extension[SseDoneDataKey])
			}
		} else if strings.HasPrefix(code, "2") && resolveInfo.succeedResponse.schema == nil {
			resolveInfo.succeedResponse = respSchema
		} else {
			resolveInfo.responses[code] = respSchema
		}
	}

	operationId := operation.OperationID
	if len(operationId) == 0 {
		operationId = path
	}

	if len(operation.Description) > 0 {
		resolveInfo.description = operation.Description
	} else {
		resolveInfo.description = operation.Summary
	}
	resolveInfo.operationId = utils.NormalizeName(utils.JoinString("_", strings.TrimPrefix(operationId, "/"), strings.ToLower(operationIdSuffix)))
	resolveInfo.parameters = append(operation.Parameters, parameters...)

	if requestBody := operation.RequestBody; requestBody != nil {
		requestSchema, contentType := FetchRequestBodyResolveSchema(requestBody.Value)
		resolveInfo.requestBody = contentSchema{contentType: contentType, schema: requestSchema}
	}
	if resolveInfo.succeedResponse.schema == nil {
		resolveInfo.succeedResponse.schema = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: openapi3.TypeBoolean, Default: true}}
	}
	if typeName == consts.TypeSubscription || !operationOnlyEventStream {
		resolve.resolve(resolveInfo)
	}

	if typeName != consts.TypeSubscription && operationContainEventStream {
		subscribedInfo := a.iteratorPathItem(path, operation, parameters, method, consts.TypeSubscription, resolve)
		if _, ok := a.customRestRequestRewriterMap[subscribedInfo.operationId]; !ok {
			if resolvedRewriter, existed := a.customRestRequestRewriterMap[resolveInfo.operationId]; existed {
				a.customRestRequestRewriterMap[subscribedInfo.operationId] = resolvedRewriter
			}
		}
		if _, ok := a.customRestResponseRewriterMap[subscribedInfo.operationId]; !ok &&
			subscribedInfo.succeedResponse == resolveInfo.succeedResponse {
			if resolvedRewriter, existed := a.customRestResponseRewriterMap[resolveInfo.operationId]; existed {
				a.customRestResponseRewriterMap[subscribedInfo.operationId] = resolvedRewriter
			}
		}
	}
	return resolveInfo
}

// 获取response定义中的schemaRef
// 添加对'*/*'类型的支持
func (a *actionOpenapi) fetchResponseResolveSchema(response *openapi3.Response, matchType ...string) (extension map[string]any, schema contentSchema, contentSize int) {
	if response != nil && response.Content != nil {
		contentSize = len(response.Content)
		for _, mime := range mimeTypes {
			if result := response.Content.Get(mime); result != nil && (len(matchType) == 0 || slices.Contains(matchType, mime)) {
				extension = result.Extensions
				schema.schema, schema.contentType = result.Schema, mime
				break
			}
		}
	}
	if schema.schema == nil {
		schema.contentType = "*/*"
		schema.schema = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: openapi3.TypeBoolean, Default: true}}
	}
	return
}

// FetchRequestBodyResolveSchema 获取requestBody定义中的schemaRef
// 添加了json和form两种类型的支持
func FetchRequestBodyResolveSchema(requestBody *openapi3.RequestBody) (schema *openapi3.SchemaRef, contentType string) {
	if requestBody.Content == nil {
		return
	}

	for _, mime := range mimeTypes {
		if result := requestBody.Content.Get(mime); result != nil {
			schema, contentType = result.Schema, mime
			break
		}
	}
	return
}

func (a *actionOpenapi) fetchSchemaRef(schemaRef *openapi3.SchemaRef) *openapi3.SchemaRef {
	if schemaRef == nil || schemaRef.Value != nil || schemaRef.Ref == "" {
		return schemaRef
	}

	return a.doc.Components.Schemas[strings.TrimPrefix(schemaRef.Ref, interpolate.Openapi3SchemaRefPrefix)]
}

// 转换引号
func normalizeDescription(description string) string {
	return strings.ReplaceAll(description, `"`, `'`)
}

func buildDefinitionName(schemaRef *openapi3.SchemaRef, definitionName string) string {
	if ref := schemaRef.Ref; ref != "" {
		definitionName = strings.TrimPrefix(ref, interpolate.Openapi3SchemaRefPrefix)
	}
	return definitionName
}
