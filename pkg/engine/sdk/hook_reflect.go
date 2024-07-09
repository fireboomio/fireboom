// Package sdk
/*
 将钩子定义所用到的结构体、枚举定义成jsonschema
 不同语言按照简化后的定义输出对应的对象定义/结构体/枚举
*/
package sdk

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/websocket"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/uber/jaeger-client-go"
	"github.com/wundergraph/graphql-go-tools/pkg/graphql"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/authentication"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"github.com/wundergraph/wundergraph/pkg/hooks"
	"github.com/wundergraph/wundergraph/pkg/logging"
	"github.com/wundergraph/wundergraph/pkg/s3uploadclient"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

const (
	serverHook     = "hook"
	EnumAliasesKey = "X-Enum-Aliases"
)

type (
	reflectObjectFactory struct {
		*objectInfoFactory
		definitions openapi3.Schemas
		endpoints   map[string]*endpointMetadata
	}
	baseRequestBody struct {
		Wg baseRequestBodyWg `json:"__wg"`
	}
	baseRequestBodyWg struct {
		ClientRequest hooks.WunderGraphRequest `json:"clientRequest"`
		User          authentication.User      `json:"user"`
	}
	customizeHookPayload struct {
		baseRequestBody
		OperationName string         `json:"operationName"`
		Variables     map[string]any `json:"variables"`
		Query         string         `json:"query"`
	}
	customizeHookResponse struct {
		Data       json.RawMessage        `json:"data"`
		Errors     []graphql.RequestError `json:"errors"`
		Extensions map[string]interface{} `json:"extensions"`
	}
	operationHookPayload struct {
		baseRequestBody
		Op       string                `json:"op"`
		Hook     consts.MiddlewareHook `json:"hook"`
		Canceled bool                  `json:"canceled"`
		Input    any                   `json:"input"`
		Response struct {
			Data   any                    `json:"data"`
			Errors []graphql.RequestError `json:"errors"`
		} `json:"response"`
		SetClientRequestHeaders hooks.RequestHeaders `json:"setClientRequestHeaders"`
	}
	uploadHookPayload struct {
		baseRequestBody
		File  s3uploadclient.HookFile `json:"file"`
		Meta  any                     `json:"meta"`
		Error struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"error"`
	}
	onWsConnectionInitHookResponse struct {
		Payload json.RawMessage `json:"payload"`
	}
	onRequestHookPayload struct {
		baseRequestBody
		hooks.OnRequestHookPayload
	}
	onResponseHookPayload struct {
		baseRequestBody
		hooks.OnResponseHookPayload
	}
	endpointMetadata struct {
		title                     consts.MiddlewareHook
		requestBody               any
		response                  any
		middlewareResponseRewrite any
	}
)

type enumRewrite struct {
	parent   any
	property string
	schema   *openapi3.SchemaRef
}

func (o *reflectObjectFactory) buildServerStructDefinitions() {
	o.reflectHookEndpoint()
	o.reflectBaseHook()
	o.reflectCustomizeHook()
	o.reflectGlobalHook()
	o.reflectAuthenticationHook()
	o.reflectOperationHook()
	o.reflectUploadHook()
	o.reflectServerConfig()
}

// 复用引擎层proto生成的枚举map定义，构建枚举的jsonschema
func (o *reflectObjectFactory) buildInt32EnumSchema(enumMap map[string]int32, anyValue any) *openapi3.SchemaRef {
	enumSchema := openapi3.NewInt32Schema()
	enumSchema.Title = utils.GetTypeName(anyValue)
	valueAliasMap := make(map[string]interface{})
	enumSchema.Extensions = map[string]interface{}{EnumAliasesKey: valueAliasMap}
	enumKeys := maps.Keys(enumMap)
	slices.Sort(enumKeys)
	for _, name := range enumKeys {
		value := enumMap[name]
		valueStr := fmt.Sprintf("%v", value)
		if _, ok := valueAliasMap[valueStr]; !ok {
			valueAliasMap[valueStr] = name
			enumSchema.Enum = append(enumSchema.Enum, value)
		}
	}
	return &openapi3.SchemaRef{Value: enumSchema}
}

// 生成引擎的配置结构体的jsonschema，并且替换枚举jsonschema定义
// 对应的结果为wgpb.WunderGraphConfiguration，实际生成文件为exported/generated/fireboom.config.json
func (o *reflectObjectFactory) reflectServerConfig() {
	argumentSourceSchemaRef := o.buildInt32EnumSchema(wgpb.ArgumentSource_value, wgpb.ArgumentSource_OBJECT_FIELD)
	argumentRenderSchemaRef := o.buildInt32EnumSchema(wgpb.ArgumentRenderConfiguration_value, wgpb.ArgumentRenderConfiguration_RENDER_ARGUMENT_DEFAULT)
	apiCacheKindSchemaRef := o.buildInt32EnumSchema(wgpb.ApiCacheKind_value, wgpb.ApiCacheKind_NO_CACHE)
	authProviderKindSchemaRef := o.buildInt32EnumSchema(wgpb.AuthProviderKind_value, wgpb.AuthProviderKind_AuthProviderGithub)
	claimTypeSchemaRef := o.buildInt32EnumSchema(wgpb.ClaimType_value, wgpb.ClaimType_ISSUER)
	variableKindSchemaRef := o.buildInt32EnumSchema(wgpb.ConfigurationVariableKind_value, wgpb.ConfigurationVariableKind_STATIC_CONFIGURATION_VARIABLE)
	dataSourceKindSchemaRef := o.buildInt32EnumSchema(wgpb.DataSourceKind_value, wgpb.DataSourceKind_STATIC)
	dateOffsetUnitSchemaRef := o.buildInt32EnumSchema(wgpb.DateOffsetUnit_value, wgpb.DateOffsetUnit_YEAR)
	httpMethodSchemaRef := o.buildInt32EnumSchema(wgpb.HTTPMethod_value, wgpb.HTTPMethod_GET)
	injectVariableKindSchemaRef := o.buildInt32EnumSchema(wgpb.InjectVariableKind_value, wgpb.InjectVariableKind_UUID)
	logLevelSchemaRef := o.buildInt32EnumSchema(wgpb.LogLevel_value, wgpb.LogLevel_DEBUG)
	operationTypeSchemaRef := o.buildInt32EnumSchema(wgpb.OperationType_value, wgpb.OperationType_QUERY)
	executionEngineSchemaRef := o.buildInt32EnumSchema(wgpb.OperationExecutionEngine_value, wgpb.OperationExecutionEngine_ENGINE_GRAPHQL)
	transformationKindSchemaRef := o.buildInt32EnumSchema(wgpb.PostResolveTransformationKind_value, wgpb.PostResolveTransformationKind_GET_POST_RESOLVE_TRANSFORMATION)
	signingMethodSchemaRef := o.buildInt32EnumSchema(wgpb.SigningMethod_value, wgpb.SigningMethod_SigningMethodHS256)
	authenticationKindSchemaRef := o.buildInt32EnumSchema(wgpb.UpstreamAuthenticationKind_value, wgpb.UpstreamAuthenticationKind_UpstreamAuthenticationJWT)
	valueTypeSchemaRef := o.buildInt32EnumSchema(wgpb.ValueType_value, wgpb.ValueType_STRING)
	relationFilterTypeSchemaRef := o.buildInt32EnumSchema(wgpb.VariableWhereInputRelationFilterType_value, wgpb.VariableWhereInputRelationFilterType_is)
	ScalarFilterTypeSchemaRef := o.buildInt32EnumSchema(wgpb.VariableWhereInputScalarFilterType_value, wgpb.VariableWhereInputScalarFilterType_equals)
	webhookVerifierKindSchemaRef := o.buildInt32EnumSchema(wgpb.WebhookVerifierKind_value, wgpb.WebhookVerifierKind_HMAC_SHA256)

	kindProperty := "kind"
	o.reflectStructSchema(wgpb.WunderGraphConfiguration{}, consts.HookGeneratedParent,
		&enumRewrite{parent: wgpb.ArgumentConfiguration{}, property: "sourceType", schema: argumentSourceSchemaRef},
		&enumRewrite{parent: wgpb.ArgumentConfiguration{}, property: "renderConfiguration", schema: argumentRenderSchemaRef},
		&enumRewrite{parent: wgpb.ApiCacheConfig{}, property: kindProperty, schema: apiCacheKindSchemaRef},
		&enumRewrite{parent: wgpb.AuthProvider{}, property: kindProperty, schema: authProviderKindSchemaRef},
		&enumRewrite{parent: wgpb.ClaimConfig{}, property: "claimType", schema: claimTypeSchemaRef},
		&enumRewrite{parent: wgpb.VariableWhereInputScalarFilter{}, property: "type", schema: ScalarFilterTypeSchemaRef},
		&enumRewrite{parent: wgpb.VariableWhereInputRelationFilter{}, property: "type", schema: relationFilterTypeSchemaRef},
		&enumRewrite{parent: wgpb.ConfigurationVariable{}, property: kindProperty, schema: variableKindSchemaRef},
		&enumRewrite{parent: wgpb.DataSourceConfiguration{}, property: kindProperty, schema: dataSourceKindSchemaRef},
		&enumRewrite{parent: wgpb.FetchConfiguration{}, property: "method", schema: httpMethodSchemaRef},
		&enumRewrite{parent: wgpb.VariableInjectionConfiguration{}, property: "variableKind", schema: injectVariableKindSchemaRef},
		&enumRewrite{parent: wgpb.DateOffset{}, property: "unit", schema: dateOffsetUnitSchemaRef},
		&enumRewrite{parent: wgpb.Logging{}, property: "level", schema: logLevelSchemaRef},
		&enumRewrite{parent: wgpb.Operation{}, property: "operationType", schema: operationTypeSchemaRef},
		&enumRewrite{parent: wgpb.Operation{}, property: "engine", schema: executionEngineSchemaRef},
		&enumRewrite{parent: wgpb.PostResolveTransformation{}, property: kindProperty, schema: transformationKindSchemaRef},
		&enumRewrite{parent: wgpb.JwtUpstreamAuthenticationWithAccessTokenExchange{}, property: "signingMethod", schema: signingMethodSchemaRef},
		&enumRewrite{parent: wgpb.UpstreamAuthentication{}, property: kindProperty, schema: authenticationKindSchemaRef},
		&enumRewrite{parent: wgpb.CustomClaim{}, property: "type", schema: valueTypeSchemaRef},
		&enumRewrite{parent: wgpb.WebhookVerifier{}, property: kindProperty, schema: webhookVerifierKindSchemaRef})
}

func (o *reflectObjectFactory) reflectHookEndpoint() {
	endpointSchema := openapi3.NewStringSchema()
	endpointSchema.Title = "Endpoint"
	endpointValueAliasMap := map[string]interface{}{}
	endpointSchema.Extensions = map[string]interface{}{EnumAliasesKey: endpointValueAliasMap}
	for hook := range models.HttpTransportHookOptionMap {
		endpoint := fmt.Sprintf("/global/httpTransport/%s", hook)
		endpointValueAliasMap[endpoint] = hook
		metadata := &endpointMetadata{title: hook, response: hooks.MiddlewareHookResponse{}}
		o.endpoints[endpoint] = metadata
		switch hook {
		case consts.HttpTransportBeforeRequest, consts.HttpTransportOnRequest:
			metadata.requestBody, metadata.middlewareResponseRewrite = onRequestHookPayload{}, hooks.OnRequestHookResponse{}
		case consts.HttpTransportAfterResponse, consts.HttpTransportOnResponse:
			metadata.requestBody, metadata.middlewareResponseRewrite = onResponseHookPayload{}, hooks.OnResponseHookResponse{}
		}
	}
	for hook := range models.AuthenticationHookOptionMap {
		endpoint := fmt.Sprintf("/authentication/%s", hook)
		endpointValueAliasMap[endpoint] = hook
		metadata := &endpointMetadata{title: hook, requestBody: baseRequestBody{}, response: hooks.MiddlewareHookResponse{}}
		o.endpoints[endpoint] = metadata
		switch hook {
		case consts.MutatingPostAuthentication, consts.RevalidateAuthentication:
			metadata.middlewareResponseRewrite = hooks.MutatingPostAuthenticationResponse{}
		}
	}
	for hook := range models.OperationHookOptionMap {
		if _, ok := models.HttpTransportHookAliasMap[hook]; !ok {
			endpoint := fmt.Sprintf("/operation/{path}/%s", hook)
			endpointValueAliasMap[endpoint] = hook
			o.endpoints[endpoint] = &endpointMetadata{
				title:       hook,
				requestBody: operationHookPayload{},
				response:    hooks.MiddlewareHookResponse{},
			}
		}
	}
	for hook := range models.StorageProfileHookOptionMap {
		endpoint := fmt.Sprintf("/upload/{provider}/{profile}/%s", hook)
		endpointValueAliasMap[endpoint] = hook
		o.endpoints[endpoint] = &endpointMetadata{
			title:       hook,
			requestBody: uploadHookPayload{},
			response:    hooks.UploadHookResponse{},
		}
	}
	wsConnectionInitEndpoint := fmt.Sprintf("/global/wsTransport/%s", consts.WsTransportOnConnectionInit)
	endpointValueAliasMap[wsConnectionInitEndpoint] = consts.WsTransportOnConnectionInit
	o.endpoints[wsConnectionInitEndpoint] = &endpointMetadata{
		title:                     consts.WsTransportOnConnectionInit,
		requestBody:               hooks.OnWsConnectionInitHookPayload{},
		response:                  hooks.MiddlewareHookResponse{},
		middlewareResponseRewrite: onWsConnectionInitHookResponse{},
	}
	graphqlEndpoint := "/gqls/{name}/graphql"
	endpointValueAliasMap[graphqlEndpoint] = consts.HookCustomizeParent
	o.endpoints[wsConnectionInitEndpoint] = &endpointMetadata{
		title:       consts.HookCustomizeParent,
		requestBody: customizeHookPayload{},
		response:    customizeHookResponse{},
	}
	functionEndpoint := "/function/{path}"
	endpointValueAliasMap[functionEndpoint] = consts.HookFunctionParent
	o.endpoints[functionEndpoint] = &endpointMetadata{
		title:       consts.HookFunctionParent,
		requestBody: operationHookPayload{},
		response:    hooks.MiddlewareHookResponse{},
	}
	proxyEndpoint := "/proxy/{path}"
	endpointValueAliasMap[proxyEndpoint] = consts.HookProxyParent
	o.endpoints[proxyEndpoint] = &endpointMetadata{
		title:       consts.HookProxyParent,
		requestBody: onRequestHookPayload{},
		response:    hooks.MiddlewareHookResponse{},
	}
	healthEndpoint, healthTitle := "/health", "health"
	endpointValueAliasMap[healthEndpoint] = healthTitle
	o.endpoints[healthEndpoint] = &endpointMetadata{title: consts.MiddlewareHook(healthTitle), response: websocket.Health{}}
	for name := range endpointValueAliasMap {
		endpointSchema.Enum = append(endpointSchema.Enum, name)
	}
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: endpointSchema}, &objectField{})

	internalEndpointSchema := openapi3.NewStringSchema()
	internalEndpointSchema.Title = "InternalEndpoint"
	internalEndpointValueAliasMap := map[string]interface{}{}
	internalEndpointSchema.Extensions = map[string]interface{}{EnumAliasesKey: internalEndpointValueAliasMap}
	internalEndpointValueAliasMap["/internal/operations/{path}"] = "internalRequest"
	internalEndpointValueAliasMap["/internal/notifyTransactionFinish"] = "internalTransaction"
	internalEndpointValueAliasMap["/s3/{provider}/upload"] = "s3upload"
	for name := range internalEndpointValueAliasMap {
		internalEndpointSchema.Enum = append(internalEndpointSchema.Enum, name)
	}
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: internalEndpointSchema}, &objectField{})
}

// 生成基础钩子相关的jsonschema，包括请求定义、响应定义、健康检查、实现端点等
func (o *reflectObjectFactory) reflectBaseHook() {
	o.reflectStructSchema(baseRequestBody{}, serverHook)
	o.reflectStructSchema(websocket.Health{}, serverHook)

	hookParentSchema := openapi3.NewStringSchema()
	hookParentSchema.Title = "HookParent"
	hookParentSchema.Enum = append(hookParentSchema.Enum,
		consts.HookGeneratedParent, consts.HookGlobalParent, consts.HookAuthenticationParent,
		consts.HookOperationParent, consts.HookStorageParent, consts.HookCustomizeParent,
		consts.HookProxyParent, consts.HookFunctionParent, consts.HookFragmentsParent)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: hookParentSchema}, &objectField{})

	rateLimitHeaderSchema := openapi3.NewStringSchema()
	rateLimitHeaderSchema.Title = "RateLimitHeader"
	rateLimitHeaderSchema.Enum = append(rateLimitHeaderSchema.Enum, apihandler.HeaderParamRateLimitUniqueKey,
		apihandler.HeaderParamRateLimitPerSecond, apihandler.HeaderParamRateLimitRequests)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: rateLimitHeaderSchema}, &objectField{})

	rbacHeaderSchema := openapi3.NewStringSchema()
	rbacHeaderSchema.Title = "RbacHeader"
	rbacHeaderSchema.Enum = append(rbacHeaderSchema.Enum,
		authentication.HeaderRbacRequireMatchAll, authentication.HeaderRbacRequireMatchAny,
		authentication.HeaderRbacDenyMatchAll, authentication.HeaderRbacDenyMatchAny)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: rbacHeaderSchema}, &objectField{})

	transactionHeaderSchema := openapi3.NewStringSchema()
	transactionHeaderSchema.Title = "TransactionHeader"
	transactionHeaderSchema.Enum = append(transactionHeaderSchema.Enum,
		database.HeaderTransactionId, database.HeaderTransactionManually)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: transactionHeaderSchema}, &objectField{})

	requestHeaderSchema := openapi3.NewStringSchema()
	requestHeaderSchema.Title = "InternalHeader"
	requestHeaderSchema.Enum = append(requestHeaderSchema.Enum, logging.RequestIDHeader, jaeger.TraceContextHeaderName,
		s3uploadclient.HeaderMetadata, s3uploadclient.HeaderUploadProfile)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: requestHeaderSchema}, &objectField{})
}

// 生成自定义数据源相关的jsonschema
func (o *reflectObjectFactory) reflectCustomizeHook() {
	o.reflectStructSchema(graphql.RequestError{}, consts.HookCustomizeParent, &enumRewrite{
		property: "path", schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: openapi3.TypeArray, Items: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}},
	})
	o.reflectStructSchema(customizeHookPayload{}, consts.HookCustomizeParent)
	o.reflectStructSchema(customizeHookResponse{}, consts.HookCustomizeParent)

	graphqlEndpointFlag := "${graphqlEndpoint}"
	customizeFlagSchema := openapi3.NewStringSchema()
	customizeFlagSchema.Title = "CustomizeFlag"
	customizeFlagSchema.Extensions = map[string]interface{}{EnumAliasesKey: map[string]interface{}{graphqlEndpointFlag: "graphqlEndpoint"}}
	customizeFlagSchema.Enum = append(customizeFlagSchema.Enum, "subscription", "__schema", graphqlEndpointFlag)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: customizeFlagSchema}, &objectField{})
}

// 生成全局钩子相关的jsonschema
func (o *reflectObjectFactory) reflectGlobalHook() {
	operationTypeStringSchema := openapi3.NewStringSchema()
	operationTypeStringSchema.Title = "OperationTypeString"
	for name := range wgpb.OperationType_value {
		operationTypeStringSchema.Enum = append(operationTypeStringSchema.Enum, strings.ToLower(name))
	}
	operationTypeRewrite := &enumRewrite{property: "operationType", schema: &openapi3.SchemaRef{Value: operationTypeStringSchema}}
	o.reflectStructSchema(onRequestHookPayload{}, consts.HookGlobalParent, operationTypeRewrite)
	o.reflectStructSchema(hooks.OnRequestHookResponse{}, consts.HookGlobalParent)
	o.reflectStructSchema(onResponseHookPayload{}, consts.HookGlobalParent, operationTypeRewrite)
	o.reflectStructSchema(hooks.OnResponseHookResponse{}, consts.HookGlobalParent)
	o.reflectStructSchema(hooks.OnWsConnectionInitHookPayload{}, consts.HookGlobalParent)
	o.reflectStructSchema(onWsConnectionInitHookResponse{}, consts.HookGlobalParent)
}

// 生成认证钩子相关的jsonschema
func (o *reflectObjectFactory) reflectAuthenticationHook() {
	o.reflectStructSchema(hooks.MutatingPostAuthenticationResponse{}, consts.HookAuthenticationParent)
}

// 生成operation钩子相关的jsonschema
func (o *reflectObjectFactory) reflectOperationHook() {
	middlewareHookSchema := openapi3.NewStringSchema()
	middlewareHookSchema.Enum = append(middlewareHookSchema.Enum,
		consts.PreResolve, consts.MutatingPreResolve, consts.MockResolve, consts.CustomResolve, consts.PostResolve, consts.MutatingPostResolve,
		consts.PostAuthentication, consts.MutatingPostAuthentication, consts.RevalidateAuthentication, consts.PostLogout,
		consts.HttpTransportBeforeRequest, consts.HttpTransportAfterResponse,
		consts.HttpTransportOnRequest, consts.HttpTransportOnResponse, consts.WsTransportOnConnectionInit)
	middlewareHookRewrite := &enumRewrite{
		property: "hook",
		schema:   &openapi3.SchemaRef{Value: middlewareHookSchema},
	}
	o.reflectStructSchema(operationHookPayload{}, consts.HookOperationParent, middlewareHookRewrite)
	o.reflectStructSchema(hooks.MiddlewareHookResponse{}, serverHook, middlewareHookRewrite)

	operationFieldSchema := openapi3.NewStringSchema()
	operationFieldSchema.Title = "OperationField"
	operationFieldSchema.Enum = append(operationFieldSchema.Enum, "path", "operationType", "variablesSchema", "responseSchema")
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: operationFieldSchema}, &objectField{})
}

// 生成上传钩子相关的jsonschema
func (o *reflectObjectFactory) reflectUploadHook() {
	o.reflectStructSchema(uploadHookPayload{}, consts.HookStorageParent)
	o.reflectStructSchema(hooks.UploadHookResponse{}, consts.HookStorageParent)
	o.reflectStructSchema(s3uploadclient.UploadedFiles{}, consts.HookStorageParent)

	uploadHookSchema := openapi3.NewStringSchema()
	uploadHookSchema.Title = utils.GetTypeName(consts.PreUpload)
	uploadHookSchema.Enum = append(uploadHookSchema.Enum, consts.PreUpload, consts.PostUpload)
	o.buildObjectFromDataSchema(&openapi3.SchemaRef{Value: uploadHookSchema}, &objectField{})
}

// 通过反射结构体生成jsonschema，并且支持改写其中枚举schema
func (o *reflectObjectFactory) reflectStructSchema(data any, rootName string, rewrites ...*enumRewrite) {
	schemas := make(openapi3.Schemas)
	refName := utils.ReflectStructToOpenapi3Schema(data, schemas)
	for _, rewrite := range rewrites {
		rewriteObject := data
		if rewrite.parent != nil {
			rewriteObject = rewrite.parent
		}
		parent, ok := schemas[utils.GetTypeName(rewriteObject)]
		if !ok || parent.Value == nil || len(parent.Value.Properties) == 0 {
			continue
		}

		properties := parent.Value.Properties
		_, ok = properties[rewrite.property]
		if !ok {
			continue
		}

		schemaValue := rewrite.schema.Value
		if schemaValue.Title == "" && len(schemaValue.Enum) > 0 {
			schemaValue.Title = utils.GetTypeName(schemaValue.Enum[0])
		}
		properties[rewrite.property] = rewrite.schema
	}
	refSchema := schemas[refName]
	o.buildObjectFromDataSchema(refSchema, &objectField{Name: refName, Root: rootName})

	delete(schemas, refName)
	if slices.Contains(endpointIgnoreNames, refName) {
		o.buildDefinitions(schemas, rootName)
	} else {
		o.definitions[refName] = refSchema
		o.buildDefinitions(schemas, rootName, func(field string, fieldSchema *openapi3.SchemaRef) {
			o.definitions[field] = fieldSchema
		})
	}
}

var (
	serverObjectFieldArray []*objectField
	serverEnumFieldArray   []*enumField
	serverTypeFormatArray  []string
	endpointIgnoreNames    = []string{utils.GetTypeName(s3uploadclient.UploadedFiles{}), utils.GetTypeName(wgpb.WunderGraphConfiguration{})}
)

func init() {
	serverReflectFactory := reflectObjectFactory{
		objectInfoFactory: newObjectInfoFactory(make(map[string]int)),
		definitions:       make(openapi3.Schemas),
		endpoints:         make(map[string]*endpointMetadata),
	}
	serverReflectFactory.buildServerStructDefinitions()
	serverReflectFactory.optimizeFieldInfo()
	serverObjectFieldArray = serverReflectFactory.sortObjectFieldArray()
	serverEnumFieldArray = serverReflectFactory.sortEnumFieldArray()
	serverTypeFormatArray = maps.Keys(serverReflectFactory.typeFormats)

	utils.RegisterInitMethod(30, func() {
		hookSwaggerText := build.GeneratedHookSwaggerText
		if _, err := hookSwaggerText.Stat(hookSwaggerText.Title); os.IsNotExist(err) {
			go serverReflectFactory.generateHookSwagger()
		}
	})
}
