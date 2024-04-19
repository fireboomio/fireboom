// Package sdk
/*
 将编译后引擎所需的部分配置写入到templateContext中
 提供objectInfoFactory简化jsonschema，使得编写template模板时清晰明了
*/
package sdk

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/plugins/fileloader"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"math"
	"strings"
)

const (
	operationModelInput         = "Input"
	operationModelInternalInput = "InternalInput"
	operationModelResponse      = "ResponseData"
)

var (
	logger                     *zap.Logger
	sdkModelName               string
	operationHooksFetchFuncMap map[wgpb.OperationType]func(*hooksConfiguration) map[string][]consts.MiddlewareHook
	hookGraphqlConfigText      *fileloader.ModelText[any]
)

type (
	objectInfoFactory struct {
		objectFieldMap map[string]*objectField
		enumFieldMap   map[string]*enumField
		maxLengthMap   map[string]int
		typeFormats    map[string]bool
	}
	objectFieldType struct {
		TypeName      string       // 类型名(为字段时使用)
		TypeFormat    string       // 类型格式化(为字段时使用)
		typeRef       string       // 忽略
		TypeRefObject *objectField // 类型引用(为字段时使用)
		TypeRefEnum   *enumField   // 枚举引用(为字段时使用)
		Required      bool         // 是否必须(为字段时使用)
		IsArray       bool         // 是否数组(为字段时使用)
	}
	objectField struct {
		objectFieldType
		Additional    *objectFieldType
		Name          string         // 对象/字段名
		Description   string         // 对象/字段描述
		Default       interface{}    // 字段默认值
		IsDefinition  bool           // 是否全局定义
		IsBasicType   bool           // 是否基础类型
		DocumentPath  []string       // 文档路径(建议拼接后用来做对象名/字段类型名)
		Fields        []*objectField // 字段列表(为对象时使用)
		Root          string         // 顶层归属类型(Input/InternalInput/ResponseData/Definitions)
		OperationInfo *operationInfo // operation信息
	}
	enumField struct {
		Name                string                 // 枚举名称
		ValueType           string                 // 枚举值类型
		Values              []interface{}          // 枚举值列表
		ValueAliasMap       map[string]interface{} // 枚举变量名替换
		ValueDescriptionMap map[string]string      // 枚举变量名描述
	}
	operationInfo struct {
		Engine           wgpb.OperationExecutionEngine
		Type             wgpb.OperationType
		Name             string
		Path             string
		HasInput         bool
		HasInternalInput bool
		AuthRequired     bool
		IsInternal       bool
		IsQuery          bool
		IsLiveQuery      bool
		IsMutation       bool
		IsSubscription   bool
	}
)

type (
	hooksConfiguration struct {
		Global         *globalHooksConfiguration
		Authentication []consts.MiddlewareHook
		Queries        map[string][]consts.MiddlewareHook
		Mutations      map[string][]consts.MiddlewareHook
		Subscriptions  map[string][]consts.MiddlewareHook
	}
	globalHooksConfiguration struct {
		HttpTransport map[consts.MiddlewareHook]*globalHttpTransportConfiguration
		WsTransport   []string
	}
	globalHttpTransportConfiguration struct {
		EnableForOperations    []string `json:"enableForOperations"`
		EnableForAllOperations bool     `json:"enableForAllOperations"`
	}
)

// 清除空的即未定义的全局operation钩子
func (t *templateContext) clearEmptyGlobalOperationHooks() {
	config := t.HooksConfiguration.Global
	for hook, configuration := range config.HttpTransport {
		if !configuration.EnableForAllOperations && len(configuration.EnableForOperations) == 0 {
			delete(config.HttpTransport, hook)
		}
	}
}

// 合成全局operation钩子开关配置
func (t *templateContext) buildGlobalOperationHooks() {
	config := t.HooksConfiguration.Global
	config.HttpTransport = make(map[consts.MiddlewareHook]*globalHttpTransportConfiguration)
	for hook, option := range models.GetHttpTransportHookOptions() {
		ensureEnabled := option.Enabled && option.Existed
		config.HttpTransport[hook] = &globalHttpTransportConfiguration{EnableForAllOperations: ensureEnabled}
	}
}

// 合成认证钩子开关配置
func (t *templateContext) buildAuthenticationHooks() {
	var authenticationHooks []consts.MiddlewareHook
	for hook, option := range models.GetAuthenticationHookOptions() {
		if option.Enabled && option.Existed {
			authenticationHooks = append(authenticationHooks, hook)
		}
	}
	t.HooksConfiguration.Authentication = authenticationHooks
}

// 合成operation钩子开关配置
func (t *templateContext) buildOperationHooks(operation *wgpb.Operation) {
	var enableHooks []consts.MiddlewareHook
	global := t.HooksConfiguration.Global
	for hook, option := range models.GetOperationHookOptions(operation.Path) {
		if !option.Enabled || !option.Existed {
			continue
		}

		if config, ok := global.HttpTransport[hook]; ok {
			config.EnableForOperations = append(config.EnableForOperations, operation.Path)
			continue
		}

		enableHooks = append(enableHooks, hook)
	}
	if len(enableHooks) == 0 {
		return
	}

	fetchFunc, ok := operationHooksFetchFuncMap[operation.OperationType]
	if !ok {
		return
	}

	fetchFunc(t.HooksConfiguration)[operation.Path] = enableHooks
}

// 从引擎层所需配置中解析出上传钩子元数据和operation的出入参数定义
// metadataJSONSchema 上传钩子添加profile且填写后才会编译
// Input operation公开api的入参，去除参数指令标识skip=true的参数
// InternalInput 全部的operation入参
// ResponseData operation响应定义
func (t *templateContext) buildFromDefinedApiSchema(api, sdkRequiredApi *wgpb.UserDefinedApi) {
	factory := newObjectInfoFactory(t.MaxLengthMap)
	for _, item := range api.S3UploadConfiguration {
		sdkRequiredApi.S3UploadConfiguration = append(sdkRequiredApi.S3UploadConfiguration, &wgpb.S3UploadConfiguration{
			Name:       item.Name,
			Endpoint:   item.Endpoint,
			BucketName: item.BucketName,
			UseSSL:     item.UseSSL,
		})
		for profileName, profileRow := range item.UploadProfiles {
			metadataJSONSchema := "{}"
			if profileRow.MetadataJSONSchema != "" {
				metadataJSONSchema = profileRow.MetadataJSONSchema
			}

			var profileMeta *openapi3.SchemaRef
			if err := json.Unmarshal([]byte(metadataJSONSchema), &profileMeta); err != nil {
				continue
			}

			rootName := "UploadProfile"
			metaName := fmt.Sprintf("%s_%sProfileMeta", item.Name, profileName)
			factory.buildObjectFromDataSchema(profileMeta, &objectField{Name: metaName, Root: rootName})
		}
	}

	operationsConfigData := build.GeneratedOperationsConfigRoot.FirstData()
	var maxLength int
	for _, item := range api.Operations {
		if itemLen := len(item.Name); itemLen > maxLength {
			maxLength = itemLen
		}
		t.buildOperationHooks(item)
		itemOperationInfo := &operationInfo{
			Name: item.Name, Path: item.Path, IsInternal: item.Internal,
			Engine:         item.Engine,
			Type:           item.OperationType,
			AuthRequired:   item.AuthenticationConfig != nil && item.AuthenticationConfig.AuthRequired,
			IsLiveQuery:    item.LiveQueryConfig != nil && item.LiveQueryConfig.Enabled,
			IsQuery:        item.OperationType == wgpb.OperationType_QUERY,
			IsMutation:     item.OperationType == wgpb.OperationType_MUTATION,
			IsSubscription: item.OperationType == wgpb.OperationType_SUBSCRIPTION,
		}
		t.Operations = append(t.Operations, itemOperationInfo)

		var operationSchema apihandler.OperationSchema
		switch item.Engine {
		case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
			graphqlFile, ok := operationsConfigData.GraphqlOperationFiles[item.Path]
			if !ok {
				continue
			}

			operationSchema = graphqlFile.OperationSchema
			itemOperationInfo.HasInternalInput = len(operationSchema.InternalVariables.Value.Properties) > 0
			internalInputName := item.Name + operationModelInternalInput
			factory.buildObjectFromDataSchema(operationSchema.InternalVariables, &objectField{Name: internalInputName, OperationInfo: itemOperationInfo, Root: operationModelInternalInput})
		case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
			functionFile, ok := operationsConfigData.FunctionOperationFiles[item.Path]
			if !ok {
				continue
			}

			operationSchema = functionFile.OperationSchema
		case wgpb.OperationExecutionEngine_ENGINE_PROXY:
			continue
		}

		itemOperationInfo.HasInput = len(operationSchema.Variables.Value.Properties) > 0
		inputName := item.Name + operationModelInput
		factory.buildObjectFromDataSchema(operationSchema.Variables, &objectField{Name: inputName, OperationInfo: itemOperationInfo, Root: operationModelInput})

		responseDataName := item.Name + operationModelResponse
		factory.buildObjectFromDataSchema(operationSchema.Response, &objectField{Name: responseDataName, OperationInfo: itemOperationInfo, Root: operationModelResponse})
	}

	build.OperationsDefinitionRwMutex.Lock()
	factory.buildDefinitions(build.GetOperationsDefinitions(), "Definitions")
	build.OperationsDefinitionRwMutex.Unlock()
	factory.optimizeFieldInfo()
	factory.maxLengthMap[""] = maxLength
	t.ObjectFieldArray = factory.sortObjectFieldArray()
	t.EnumFieldArray = factory.sortEnumFieldArray()
	t.TypeFormatArray = maps.Keys(factory.typeFormats)

	t.ServerObjectFieldArray = serverObjectFieldArray
	t.ServerEnumFieldArray = serverEnumFieldArray
	t.ServerTypeFormatArray = serverTypeFormatArray
}

func newObjectInfoFactory(maxLengthMap map[string]int) *objectInfoFactory {
	return &objectInfoFactory{
		objectFieldMap: make(map[string]*objectField, math.MaxInt8),
		enumFieldMap:   make(map[string]*enumField),
		maxLengthMap:   maxLengthMap,
		typeFormats:    make(map[string]bool),
	}
}

// 排序对象定义
func (o *objectInfoFactory) sortObjectFieldArray() []*objectField {
	objectFieldArray := maps.Values(o.objectFieldMap)
	slices.SortFunc(objectFieldArray, func(a *objectField, b *objectField) bool {
		return utils.JoinStringWithDot(a.DocumentPath...) < utils.JoinStringWithDot(b.DocumentPath...)
	})
	return objectFieldArray
}

// 排序枚举定义
func (o *objectInfoFactory) sortEnumFieldArray() []*enumField {
	enumFieldArray := maps.Values(o.enumFieldMap)
	slices.SortFunc(enumFieldArray, func(a *enumField, b *enumField) bool {
		return a.Name < b.Name
	})
	return enumFieldArray
}

// 遍历添加definitions且去重复
func (o *objectInfoFactory) buildDefinitions(schemas openapi3.Schemas, root string, eachItemFunc ...func(string, *openapi3.SchemaRef)) {
	for field, fieldSchema := range schemas {
		if _, ok := o.objectFieldMap[field]; ok {
			continue
		}

		o.buildObjectFromDataSchema(fieldSchema, &objectField{Name: field, IsDefinition: true, Root: root})
		for _, eachFunc := range eachItemFunc {
			eachFunc(field, fieldSchema)
		}
	}
}

// 将ref指向对象的字符串引用替换真实的对象引用
func (o *objectInfoFactory) buildObjectFieldItemRef(field *objectField) {
	if ref := field.typeRef; ref != "" {
		if refInfo, ok := o.objectFieldMap[ref]; ok {
			field.TypeRefObject = refInfo
		}
	}
	if additional := field.Additional; additional != nil && additional.typeRef != "" {
		if refInfo, ok := o.objectFieldMap[additional.typeRef]; ok {
			additional.TypeRefObject = refInfo
		}
	}

	var maxLength int
	for _, itemField := range field.Fields {
		o.buildObjectFieldItemRef(itemField)
		if itemLen := len(itemField.Name); itemLen > maxLength {
			maxLength = itemLen
		}
	}
	if maxLength > 0 {
		o.maxLengthMap[utils.JoinStringWithDot(field.DocumentPath...)] = maxLength
	}
}

// 格式化对象/枚举定义，对字段按名称进行排序
func (o *objectInfoFactory) optimizeFieldInfo() {
	for _, field := range o.objectFieldMap {
		o.buildObjectFieldItemRef(field)
		slices.SortFunc(field.Fields, func(a *objectField, b *objectField) bool {
			return a.Name < b.Name
		})
	}

	for _, enum := range o.enumFieldMap {
		slices.SortFunc(enum.Values, func(a, b any) bool {
			return cast.ToString(a) < cast.ToString(b)
		})
		o.maxLengthMap[enum.Name] = len(cast.ToString(enum.Values[len(enum.Values)-1]))
	}
}

// 将openapi3.SchemaRef转换成简化易于理解对objectField/enumField
func (o *objectInfoFactory) buildObjectFromDataSchema(schemaRef *openapi3.SchemaRef, info *objectField) {
	if info.DocumentPath == nil {
		info.DocumentPath = []string{info.Name}
	}

	if ref := schemaRef.Ref; ref != "" {
		info.typeRef = strings.TrimPrefix(ref, interpolate.Openapi3SchemaRefPrefix)
		return
	}

	schema := schemaRef.Value
	info.Description, info.Default = schema.Description, schema.Default
	if enum := schema.Enum; enum != nil {
		info.TypeRefEnum = &enumField{Name: schema.Title, ValueType: schema.Type, Values: schema.Enum}
		if extensions := schema.Extensions; extensions != nil {
			if value, ok := extensions[EnumAliasesKey]; ok {
				info.TypeRefEnum.ValueAliasMap = value.(map[string]interface{})
			}
			if value, ok := extensions[build.EnumDescriptionsKey]; ok {
				info.TypeRefEnum.ValueDescriptionMap = value.(map[string]string)
			}
		}
		o.enumFieldMap[schema.Title] = info.TypeRefEnum
		return
	}

	additionalHandle := func(additionalSchema *openapi3.SchemaRef) {
		additionalDocumentPath := utils.CopyAndAppendItem(info.DocumentPath, datasource.AdditionalSuffix)
		additionalField := &objectField{DocumentPath: additionalDocumentPath, Root: info.Root, IsDefinition: info.IsDefinition}
		o.buildObjectFromDataSchema(additionalSchema, additionalField)
		info.Additional = &additionalField.objectFieldType
	}

	switch schema.Type {
	case openapi3.TypeArray:
		if schema.Items == nil {
			break
		}

		if len(info.DocumentPath) == 1 {
			additionalHandle(schema.Items)
			info.TypeName = schema.Type
			o.objectFieldMap[info.DocumentPath[0]] = info
		} else {
			info.IsArray = true
			o.buildObjectFromDataSchema(schema.Items, info)
		}
	case openapi3.TypeObject:
		isMultipleDocumentPath := len(info.DocumentPath) > 1
		additionalProperties := schema.AdditionalProperties
		if additionalProperties.Schema != nil {
			additionalHandle(additionalProperties.Schema)
		} else if additionalProperties.Has != nil && *additionalProperties.Has {
			info.Additional = &objectFieldType{}
			if len(schema.Title) > 0 {
				info.DocumentPath = []string{schema.Title}
				info.typeRef = schema.Title
			}
		} else if len(schema.Properties) == 0 && isMultipleDocumentPath {
			info.Additional = &objectFieldType{}
		} else {
			for field, fieldSchema := range schema.Properties {
				itemDocumentPath := utils.CopyAndAppendItem(info.DocumentPath, field)
				itemField := &objectField{Name: field, DocumentPath: itemDocumentPath, Root: info.Root, IsDefinition: info.IsDefinition}
				itemField.Required = slices.Contains(schema.Required, field)
				o.buildObjectFromDataSchema(fieldSchema, itemField)
				info.Fields = append(info.Fields, itemField)
			}
		}

		info.TypeName = schema.Type
		documentName := strings.Join(info.DocumentPath, "")
		o.objectFieldMap[documentName] = info
		if isMultipleDocumentPath && len(info.typeRef) == 0 {
			info.typeRef = documentName
		}
	default:
		info.TypeName = schema.Type
		info.TypeFormat = schema.Format
		info.IsBasicType = schema.Type != ""
		if !info.IsBasicType && len(info.DocumentPath) == 1 {
			o.objectFieldMap[info.DocumentPath[0]] = info
		}
		if len(info.TypeFormat) > 0 {
			if _, ok := o.typeFormats[info.TypeFormat]; !ok {
				o.typeFormats[info.TypeFormat] = true
			}
		}
	}
}

func init() {
	operationHooksFetchFuncMap = map[wgpb.OperationType]func(*hooksConfiguration) map[string][]consts.MiddlewareHook{
		wgpb.OperationType_QUERY: func(configuration *hooksConfiguration) map[string][]consts.MiddlewareHook {
			return configuration.Queries
		},
		wgpb.OperationType_MUTATION: func(configuration *hooksConfiguration) map[string][]consts.MiddlewareHook {
			return configuration.Mutations
		},
		wgpb.OperationType_SUBSCRIPTION: func(configuration *hooksConfiguration) map[string][]consts.MiddlewareHook {
			return configuration.Subscriptions
		},
	}

	hookGraphqlConfigText = &fileloader.ModelText[any]{
		Extension: fileloader.ExtJson,
		TextRW: &fileloader.SingleTextRW[any]{
			Name: consts.ExportedGeneratedFireboomConfigFilename,
		},
	}

	utils.RegisterInitMethod(30, func() {
		logger = zap.L()
		sdkModelName = models.SdkRoot.GetModelName()
		hookGraphqlConfigText.Init()
	})

	models.AddSwitchServerSdkWatcher(func(outputPath, _, _ string, _ bool) {
		if outputPath != "" {
			hookGraphqlConfigText.Root = utils.NormalizePath(outputPath, consts.HookGeneratedParent)
		} else {
			hookGraphqlConfigText.Root = ""
		}
	})
}
