// Package build
/*
 处理graphql文档
 解析指令并执行自定义逻辑
 判断文档类型Query/Mutation/Subscription
*/
package build

import (
	"bufio"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/engine/directives"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"io"
	"strings"
)

const (
	directiveResolveErrorFormat      = "directive [%s] resolve error: %v"
	directiveNotSupportedFormat      = "not support directive [%s] on [%s]"
	rootDefinitionMissFormat         = "not found rootDefinition named [%s]"
	variableTypeMustCompatibleFormat = "variable [%s] must compatible with %s"
	variableUselessFormat            = "variable [%s] useless, please make sure"
	argumentDefinitionMissFormat     = "not found argumentDefinition named [%s] on path [%s]"
	argumentRequiredFormat           = "argument [%s] is required on path [%s]"
	argumentElementRepeatFormat      = "argument element [%s] is repeat on path [%s]"
	argumentAtLeastOneFormat         = "argument of [%s] at least one on path [%s]"
	fieldDefinitionMissFormat        = "not found fieldDefinition named [%s] on path [%s]"
	fieldTouchRepeatFormat           = "field [%s] is repeat on path [%s]"
	selectionFieldMissFormat         = "not found selectionField named [%s] on path [%s]"
	nullableRequiredErrorFormat      = "definition for [null] expected [*Nullable*], but found [%s] on path [%s]"
	selectionVariableMissFormat      = "not found variable named [%s] for path [%s]"
	fieldDefinitionSupplyErrorFormat = "must supply [%s] on path [%s]"
	nullableRequiredKey              = "Nullable"
	selectionRootField               = "data"
	EnumDescriptionsKey              = "X-Enum-Descriptions"
	whereUniqueInputSuffix           = "WhereUniqueInput"
	compoundUniqueInputSuffix        = "CompoundUniqueInput"
)

type QueryDocumentItem struct {
	modelName       string
	operation       *wgpb.Operation
	definitionFetch func(string) *ast.Definition

	itemDocument            *ast.QueryDocument
	operationDefinition     *ast.OperationDefinition
	usedArgumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef]
	operationSchema         apihandler.OperationSchema
	variablesSchemas        openapi3.Schemas
	variablesRefs           []string
	variablesRefVisited     map[string]bool
	variablesExported       map[string]bool
	definitionFieldIndexes  map[*ast.Definition]*definitionFieldOverview
	fieldArgumentIndexes    map[*ast.FieldDefinition]*fieldArgumentOverview
	Errors                  []string
}

type jsonschemaObjectBuildFunc func(*ast.Definition) *openapi3.SchemaRef

func NewQueryDocumentItem(queryContent string) (item *QueryDocumentItem, err error) {
	itemDocument, err := parser.ParseQuery(&ast.Source{Input: queryContent})
	if err != nil {
		return
	}

	if size := len(itemDocument.Operations); size != 1 {
		err = fmt.Errorf("amount of operation definition expected 1, but found [%d]", size)
		return
	}

	item = &QueryDocumentItem{
		modelName:           models.OperationRoot.GetModelName(),
		itemDocument:        itemDocument,
		operationDefinition: itemDocument.Operations[0],
	}
	return
}

// ModifyOperationDirective 修改指令中参数值
// 例如修改graphql所需的角色列表
func (i *QueryDocumentItem) ModifyOperationDirective(directive *ast.Directive) {
	directiveResolve := directives.GetOperationDirectiveMapByName(directive.Name)
	if directiveResolve == nil {
		if !directives.IsBaseDirective(directive.Name) {
			i.reportError(directiveNotSupportedFormat, directive.Name, directive.Location)
		}
		return
	}

	if exist := i.operationDefinition.Directives.ForName(directive.Name); exist != nil {
		exist.Arguments = directive.Arguments
	} else {
		i.operationDefinition.Directives = append(i.operationDefinition.Directives, directive)
	}
	return
}

// PrintQueryDocument 加你个graphql文档输出到writer中
func (i *QueryDocumentItem) PrintQueryDocument(writer io.Writer) error {
	bufferWriter := bufio.NewWriter(writer)
	formatter.NewFormatter(bufferWriter).FormatQueryDocument(i.itemDocument)
	return bufferWriter.Flush()
}

// 设置解析所需的必要参数，仅当需要完成整个文档解析时需要
func (i *QueryDocumentItem) setResolveParameters(operation *wgpb.Operation, touchedArgumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef], definitionFetch func(string) *ast.Definition,
	definitionFieldIndexes map[*ast.Definition]*definitionFieldOverview, fieldArgumentIndexes map[*ast.FieldDefinition]*fieldArgumentOverview) {
	i.operation, i.usedArgumentDefinitions = operation, touchedArgumentDefinitions
	i.definitionFetch, i.definitionFieldIndexes, i.fieldArgumentIndexes = definitionFetch, definitionFieldIndexes, fieldArgumentIndexes
	i.variablesSchemas, i.variablesRefVisited, i.variablesExported = make(openapi3.Schemas), make(map[string]bool), make(map[string]bool)
}

func (i *QueryDocumentItem) resolveOperationList() {
	i.operationDefinition.Name = i.operation.Name
	i.operation.OperationType = wgpb.OperationType(wgpb.OperationType_value[strings.ToUpper(string(i.operationDefinition.Operation))])

	rootDefinitionName := utils.UppercaseFirst(string(i.operationDefinition.Operation))
	rootDefinition := i.definitionFetch(rootDefinitionName)
	if rootDefinition == nil {
		i.reportErrorWithPath(rootDefinitionMissFormat, rootDefinitionName)
		return
	}

	// 解析出返回值的schema定义
	i.operationSchema.Response, i.operationDefinition.SelectionSet = i.resolveSelectionSet("", i.operationDefinition.SelectionSet, rootDefinition, selectionRootField)

	// 解析出入参的schema定义，部分指令修饰的参数不需要传递，所以分成2个schema
	i.operationSchema.Variables = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	i.operationSchema.InternalVariables = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	i.resolveVariableDefinitions(i.operationDefinition, i.operationSchema.Variables, i.operationSchema.InternalVariables)

	i.resolveOperationDirectives()
	return
}

// 递归解析查询字段，返回schema定义
func (i *QueryDocumentItem) resolveSelectionSet(datasourceQuote string, selectionSet ast.SelectionSet, definition *ast.Definition, path ...string) (schemaRef *openapi3.SchemaRef, savedSet ast.SelectionSet) {
	schemaRef = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	setLength := len(selectionSet)
	if setLength == 0 || i.resolveErrored() {
		savedSet = selectionSet
		return
	}

	var fieldIndexes map[string]int
	if overview, ok := i.definitionFieldIndexes[definition]; ok {
		fieldIndexes = overview.indexes
	} else {
		fieldIndexes = make(map[string]int, len(definition.Fields))
		for index, item := range definition.Fields {
			fieldIndexes[item.Name] = index
		}
		i.definitionFieldIndexes[definition] = &definitionFieldOverview{indexes: fieldIndexes}
	}
	savedSet = make(ast.SelectionSet, 0, setLength)
	selectionSchema := schemaRef.Value
	for _, item := range selectionSet {
		switch field := item.(type) {
		case *ast.InlineFragment, *ast.FragmentSpread:
			savedSet = append(savedSet, item)
		case *ast.Field:
			var (
				itemName                    string
				itemPath                    []string
				itemNonNull                 bool
				itemSchemaRef               *openapi3.SchemaRef
				itemCustomizedDirectiveName string
			)
			if field.Alias != "" {
				itemName = field.Alias
			} else {
				itemName = field.Name
			}

			fieldDefIndex, ok := fieldIndexes[field.Name]
			if ok {
				fieldDefinition := definition.Fields[fieldDefIndex]
				if !datasource.ContainsRootDefinition(definition.Name, fieldDefinition.Type.Name()) &&
					(len(fieldDefinition.Arguments) > 0 || len(field.SelectionSet) == 0) &&
					slices.ContainsFunc(savedSet, func(selection ast.Selection) bool {
						savedField, saved := selection.(*ast.Field)
						return saved && savedField.Name == field.Name
					}) {
					i.reportErrorWithPath(fieldTouchRepeatFormat, field.Name, path...)
					return
				}

				var fieldOriginName string
				savedSet = append(savedSet, item)
				fieldDefDescription := fieldDefinition.Description
				// 从字段定义中匹配数据源名称并存入operation的数据源引用列表中
				if quote, cleared := matchDatasource(fieldDefDescription); quote != "" {
					datasourceQuote, fieldDefDescription = quote, cleared
					fieldOriginName = strings.TrimPrefix(field.Name, quote+"_")
					if datasource.ContainsRootDefinition(definition.Name) {
						if exist, ok := i.operation.DatasourceQuotes[quote]; ok {
							exist.Fields = append(exist.Fields, fieldOriginName)
						} else {
							i.operation.DatasourceQuotes[quote] = &wgpb.DatasourceQuote{Fields: []string{fieldOriginName}}
						}
					}
				}

				itemPath = CopyAndAppendItem(path, itemName, fieldDefinition.Type)
				itemSchemaRef = i.buildJsonschema(len(field.SelectionSet) > 0, fieldDefDescription, fieldDefinition.Type, func(definition *ast.Definition) (resultSchema *openapi3.SchemaRef) {
					// 当碰到对象schema需要继续递归处理
					resultSchema, field.SelectionSet = i.resolveSelectionSet(datasourceQuote, field.SelectionSet, definition, itemPath...)
					resultSchema.Value.Description = definition.Description
					if fieldOriginName == "queryRaw" {
						resultSchema = wrapArraySchemaRef(resultSchema)
					}
					return
				}, path...)
				itemNonNull = fieldDefinition.Type.NonNull
				i.resolveSelectionArguments(field.Name, field.Arguments, fieldDefinition, path...)
			} else {
				var itemError error
				itemPath = CopyAndAppendItem(path, itemName)
				resolver := i.makeSelectionResolver(itemPath)
				itemCustomizedDirectiveName, itemError = directives.FieldCustomizedDefinition(field.Directives, resolver)
				if itemError != nil {
					i.reportError(directiveResolveErrorFormat, itemCustomizedDirectiveName, itemError)
					return
				}
				if itemSchemaRef = resolver.Schema; itemSchemaRef == nil {
					i.reportErrorWithPath(selectionFieldMissFormat, field.Name, path...)
					return
				}
			}

			selectionSchema.Properties[itemName] = itemSchemaRef
			originNullable := itemSchemaRef.Value.Nullable
			i.resolveSelectionDirectives(field.Directives, itemPath, itemSchemaRef, itemCustomizedDirectiveName)
			if itemNonNull && !(!originNullable && itemSchemaRef.Value.Nullable) {
				selectionSchema.Required = append(selectionSchema.Required, itemName)
			}
		}
	}
	return
}

// 处理查询字段的入参
func (i *QueryDocumentItem) resolveSelectionArguments(fieldName string, arguments ast.ArgumentList, fieldDefinition *ast.FieldDefinition, parentPath ...string) {
	if len(arguments) == 0 || i.resolveErrored() {
		return
	}

	argumentOverview, ok := i.fieldArgumentIndexes[fieldDefinition]
	if !ok {
		argumentOverview = &fieldArgumentOverview{indexes: make(map[string]int, len(fieldDefinition.Arguments))}
		for index, item := range fieldDefinition.Arguments {
			argumentOverview.indexes[item.Name] = index
			if item.Type.NonNull {
				argumentOverview.required = append(argumentOverview.required, item.Name)
			}
		}
		i.fieldArgumentIndexes[fieldDefinition] = argumentOverview
	}

	var touchedArguments []string
	nextParentPath := CopyAndAppendItem(parentPath, fieldName)
	for _, argItem := range arguments {
		if slices.Contains(touchedArguments, argItem.Name) {
			i.reportErrorWithPath(argumentElementRepeatFormat, argItem.Name, nextParentPath...)
			return
		}

		argDefinitionIndex := argumentOverview.indexes[argItem.Name]
		if argDefinitionIndex == -1 {
			i.reportErrorWithPath(argumentDefinitionMissFormat, argItem.Name, nextParentPath...)
			return
		}

		argDefinition := fieldDefinition.Arguments[argumentOverview.indexes[argItem.Name]]
		if argDefinition == nil {
			i.reportErrorWithPath(argumentDefinitionMissFormat, argItem.Name, nextParentPath...)
			return
		}

		// 记录已经使用的参数定义
		touchedArguments = append(touchedArguments, argItem.Name)
		i.checkArgumentChildValueList(argItem.Name, argItem.Value, argDefinition.Description, argDefinition.Type, false, nextParentPath...)
	}

	for _, name := range argumentOverview.required {
		// 检查必填参数缺失
		if !slices.Contains(touchedArguments, name) {
			i.reportErrorWithPath(argumentRequiredFormat, name, nextParentPath...)
			return
		}
	}
}

// 根据不同类型处理参数列表
func (i *QueryDocumentItem) checkArgumentChildValueList(argName string, argValue *ast.Value, argDefDescription string, argDefType *ast.Type, parentNullable bool, parentPath ...string) {
	isVariableArg := argValue.Kind == ast.Variable
	argSchema := i.buildJsonschema(false, argDefDescription, argDefType, func(argDef *ast.Definition) *openapi3.SchemaRef {
		return i.buildSelectionArgumentSchema(isVariableArg, argDef, parentPath...)
	}, parentPath...)

	argPath := CopyAndAppendItem(parentPath, argName)
	if argValue == nil {
		return
	}

	if isVariableArg {
		variableIndex := slices.IndexFunc(i.operationDefinition.VariableDefinitions, func(item *ast.VariableDefinition) bool {
			return item.Variable == argValue.Raw
		})
		// Variable 传递参数时判断参数是否正确定义
		if variableIndex == -1 {
			i.reportError(selectionVariableMissFormat, argValue.Raw, utils.JoinStringWithDot(argPath...))
			return
		}

		variableDefinition := i.operationDefinition.VariableDefinitions[variableIndex]
		if !isCompatibleType(variableDefinition.Type, argDefType, variableDefinition.DefaultValue) {
			i.reportError(variableTypeMustCompatibleFormat, argValue.Raw, argDefType.String())
			return
		}

		i.variablesSchemas[argValue.Raw] = argSchema
	}

	// 参数值为空 判断参数定义是否允许为空
	fieldName := argDefType.Name()
	fieldIsNullValue := argValue.Kind == ast.NullValue
	// scalar类型不做处理
	if i.isScalarDefinition(fieldName) {
		if fieldIsNullValue && !parentNullable {
			i.reportError(nullableRequiredErrorFormat, fieldName, utils.JoinStringWithDot(argPath...))
			return
		}
		if argDefType.Elem != nil && argValue.Kind == ast.ListValue {
			for index, child := range argValue.Children {
				i.checkArgumentChildValueList(fmt.Sprintf(`[%d]`, index), child.Value, argDefDescription, argDefType.Elem, fieldIsNullValue, argPath...)
			}
		}
		return
	}

	fieldDefinition := i.definitionFetch(fieldName)
	if fieldDefinition == nil {
		i.reportErrorWithPath(fieldDefinitionMissFormat, fieldName, argPath...)
		return
	}
	if fieldDefinition.Kind != ast.InputObject && !(fieldDefinition.Kind == ast.Enum && argValue.Kind == ast.ListValue) {
		return
	}

	fieldNullable := strings.Contains(fieldName, nullableRequiredKey)
	if fieldIsNullValue && !fieldNullable {
		i.reportError(nullableRequiredErrorFormat, fieldName, utils.JoinStringWithDot(argPath...))
		return
	}

	// 检查是否错误使用对象定义
	if argDefType.Elem == nil && argValue.Kind == ast.ListValue {
		i.reportErrorWithPath(fieldDefinitionSupplyErrorFormat, utils.JoinString(" of ", openapi3.TypeObject, fieldName), argPath...)
		return
	}

	fieldOverview, ok := i.definitionFieldIndexes[fieldDefinition]
	if !ok {
		hasUniqueInputSuffix := strings.HasSuffix(fieldName, whereUniqueInputSuffix)
		fieldOverview = &definitionFieldOverview{indexes: make(map[string]int, len(fieldDefinition.Fields))}
		for index, item := range fieldDefinition.Fields {
			fieldOverview.indexes[item.Name] = index
			if item.Type.NonNull {
				fieldOverview.required = append(fieldOverview.required, item.Name)
			}
			if hasUniqueInputSuffix {
				itemFieldTypeName := fetchRealType(item.Type).String()
				if i.isScalarDefinition(itemFieldTypeName) || strings.HasSuffix(itemFieldTypeName, compoundUniqueInputSuffix) {
					fieldOverview.inputUniques = append(fieldOverview.inputUniques, item.Name)
				}
			}
		}
		i.definitionFieldIndexes[fieldDefinition] = fieldOverview
	}

	childFieldAllRequired := len(fieldOverview.required) == len(fieldOverview.indexes)
	// 数组类型参数判断子属性大于1时是否正确定义为数组
	if !childFieldAllRequired && len(argValue.Children) > 1 && argDefType.Elem != nil && argValue.Kind == ast.ObjectValue {
		i.reportErrorWithPath(fieldDefinitionSupplyErrorFormat, utils.JoinString(" of ", openapi3.TypeArray, fieldName), argPath...)
		return
	}

	var (
		uniqueFieldTouched bool
		touchFieldNames    []string
	)
	checkMissFieldRequired := true
	for index, child := range argValue.Children {
		if child.Name == "" {
			checkMissFieldRequired = false
			i.checkArgumentChildValueList(fmt.Sprintf(`[%d]`, index), child.Value, argDefDescription, argDefType.Elem, fieldNullable, argPath...)
			continue
		}

		if slices.Contains(touchFieldNames, child.Name) {
			i.reportErrorWithPath(argumentElementRepeatFormat, child.Name, argPath...)
			return
		}

		childFieldDefIndex, ok := fieldOverview.indexes[child.Name]
		if !ok {
			i.reportErrorWithPath(argumentDefinitionMissFormat, child.Name, argPath...)
			return
		}

		touchFieldNames = append(touchFieldNames, child.Name)
		if slices.Contains(fieldOverview.inputUniques, child.Name) {
			uniqueFieldTouched = true
		}
		childDefinition := fieldDefinition.Fields[childFieldDefIndex]
		i.checkArgumentChildValueList(child.Name, child.Value, childDefinition.Description, childDefinition.Type, fieldNullable, argPath...)
	}
	if isVariableArg || !checkMissFieldRequired {
		return
	}
	if inputUniquesLength := len(fieldOverview.inputUniques); inputUniquesLength > 0 && !uniqueFieldTouched {
		missFormat := argumentRequiredFormat
		if inputUniquesLength > 1 {
			missFormat = argumentAtLeastOneFormat
		}
		i.reportErrorWithPath(missFormat, utils.JoinString(", ", fieldOverview.inputUniques...), argPath...)
		return
	}
	for _, name := range fieldOverview.required {
		// 检查必填参数缺失
		if !slices.Contains(touchFieldNames, name) {
			i.reportErrorWithPath(argumentRequiredFormat, name, argPath...)
			return
		}
	}
	return
}

// 汇报定义缺失错误
func (i *QueryDocumentItem) reportErrorWithPath(format, argName string, path ...string) {
	args := []any{argName}
	if len(path) > 0 {
		args = append(args, utils.JoinStringWithDot(path...))
	}
	i.reportError(format, args...)
}

func (i *QueryDocumentItem) reportError(format string, args ...any) {
	i.Errors = append(i.Errors, fmt.Sprintf(format, args...))
}

func (i *QueryDocumentItem) resolveErrored() bool {
	return len(i.Errors) > 0
}

// 解析传递参数类型的定义
func (i *QueryDocumentItem) resolveVariableDefinitions(operation *ast.OperationDefinition, variableSchema, interpolationVariableSchema *openapi3.SchemaRef) {
	variableDefLength := len(operation.VariableDefinitions)
	if variableDefLength == 0 || i.resolveErrored() {
		return
	}

	var required, interpolationRequired []string
	savedVariableDefinitions := make(ast.VariableDefinitionList, 0, variableDefLength)
	schemas, interpolationSchemas := make(openapi3.Schemas), make(openapi3.Schemas)
	for _, item := range operation.VariableDefinitions {
		itemVariableName, itemVariablePath := item.Variable, []string{item.Variable}
		if i.ensureVariableDefinitionSaved(item.Directives, itemVariablePath, item.Type, item.DefaultValue) {
			savedVariableDefinitions = append(savedVariableDefinitions, item)
		}
		itemSchemaRef, ok := i.variablesSchemas[itemVariableName]
		if !ok {
			i.reportError(variableUselessFormat, itemVariableName)
			return
		}

		if defaultValue := item.DefaultValue; defaultValue != nil && itemSchemaRef.Value != nil {
			itemSchemaRef.Value.Default = defaultValue.String()
		}

		unableInput, skipped := i.resolveVariableDirectives(item.Directives, itemVariablePath, itemSchemaRef)
		if unableInput {
			continue
		}

		if item.Type.Name() == consts.ScalarBinary {
			i.operation.MultipartForms = append(i.operation.MultipartForms, &wgpb.OperationMultipartForm{
				FieldName: item.Variable,
				IsArray:   item.Type.Elem != nil,
			})
		}
		interpolationSchemas[itemVariableName] = itemSchemaRef
		if item.Type.NonNull {
			interpolationRequired = append(interpolationRequired, itemVariableName)
		}
		if skipped {
			continue
		}

		schemas[itemVariableName] = itemSchemaRef
		if item.Type.NonNull {
			required = append(required, itemVariableName)
		}
	}

	operation.VariableDefinitions = savedVariableDefinitions
	schemaValue := variableSchema.Value
	schemaValue.Properties, schemaValue.Required = schemas, required
	interpolationSchemaValue := interpolationVariableSchema.Value
	interpolationSchemaValue.Properties, interpolationSchemaValue.Required = interpolationSchemas, interpolationRequired
	return
}

// 解析operation上的指令
func (i *QueryDocumentItem) resolveOperationDirectives() {
	if len(i.operationDefinition.Directives) == 0 || i.resolveErrored() {
		return
	}

	for _, directiveItem := range i.operationDefinition.Directives {
		directiveResolve := directives.GetOperationDirectiveMapByName(directiveItem.Name)
		if directiveResolve == nil {
			if !directives.IsBaseDirective(directiveItem.Name) {
				i.reportError(directiveNotSupportedFormat, directiveItem.Name, directiveItem.Location)
			}
			continue
		}

		operationResolver := &directives.OperationResolver{
			Operation: i.operation,
			Arguments: directives.ResolveDirectiveArguments(directiveItem.Arguments),
		}
		if err := directiveResolve.Resolve(operationResolver); err != nil {
			i.reportError(directiveResolveErrorFormat, directiveItem.Name, err)
			continue
		}
	}
	return
}

// 解析查询字段上的指令
func (i *QueryDocumentItem) resolveSelectionDirectives(directiveList ast.DirectiveList, path []string, schemaRef *openapi3.SchemaRef, itemCustomizedDirectiveName string) {
	if len(directiveList) == 0 || i.resolveErrored() {
		return
	}

	for _, directiveItem := range directiveList {
		if directiveItem.Name == itemCustomizedDirectiveName {
			continue
		}
		directiveResolve := directives.GetSelectionDirectiveByName(directiveItem.Name)
		if directiveResolve == nil {
			if !directives.IsBaseDirective(directiveItem.Name) {
				i.reportError(directiveNotSupportedFormat, directiveItem.Name, directiveItem.Location)
			}
			continue
		}

		selectionResolver := i.makeSelectionResolver(path)
		selectionResolver.Schema, selectionResolver.Arguments = schemaRef, directives.ResolveDirectiveArguments(directiveItem.Arguments)
		if err := directiveResolve.Resolve(selectionResolver); err != nil {
			i.reportError(directiveResolveErrorFormat, directiveItem.Name, err)
			continue
		}
	}
	return
}

func (i *QueryDocumentItem) makeSelectionResolver(path []string) *directives.SelectionResolver {
	selectionResolver := &directives.SelectionResolver{
		Path:                path,
		VariableSchemas:     i.variablesSchemas,
		VariableExported:    i.variablesExported,
		OperationDefinition: i.operationDefinition,
	}
	selectionResolver.Operation = i.operation
	return selectionResolver
}

// 解析传递参数上的指令
func (i *QueryDocumentItem) ensureVariableDefinitionSaved(directiveList ast.DirectiveList, path []string, variableType *ast.Type, defaultValue *ast.Value) bool {
	variable := path[0]
	if _, ok := i.variablesSchemas[variable]; ok {
		return true
	}
	if !directives.VariableDefinitionRemoveRequired(directiveList) {
		return true
	}

	variableSchema := i.buildJsonschema(false, "", variableType, func(definition *ast.Definition) *openapi3.SchemaRef {
		return i.buildSelectionArgumentSchema(true, definition, path...)
	}, path...)
	i.variablesSchemas[variable] = variableSchema
	if defaultValue == nil {
		return false
	}

	if i.operation.HookVariableDefaultValues == nil {
		i.operation.HookVariableDefaultValues = []byte(`{}`)
	}
	variableDefaultValueBytes := []byte(defaultValue.String())
	i.operation.HookVariableDefaultValues, _ = jsonparser.Set(i.operation.HookVariableDefaultValues, variableDefaultValueBytes, variable)
	return false
}

// 解析传递参数上的指令
func (i *QueryDocumentItem) resolveVariableDirectives(directiveList ast.DirectiveList, path []string, schemaRef *openapi3.SchemaRef) (unableInput, skipped bool) {
	if len(directiveList) == 0 || i.resolveErrored() {
		return
	}

	for _, directiveItem := range directiveList {
		directiveResolve := directives.GetVariableDirectiveByName(directiveItem.Name)
		if directiveResolve == nil {
			if !directives.IsBaseDirective(directiveItem.Name) {
				i.reportError(directiveNotSupportedFormat, directiveItem.Name, directiveItem.Location)
			}
			continue
		}

		variableResolver := &directives.VariableResolver{
			SelectionResolver: directives.SelectionResolver{
				Path:                path,
				Schema:              schemaRef,
				VariableSchemas:     i.variablesSchemas,
				VariableExported:    i.variablesExported,
				OperationDefinition: i.operationDefinition,
			},
			ArgumentDefinitions: i.usedArgumentDefinitions,
		}
		variableResolver.Operation, variableResolver.Arguments = i.operation, directives.ResolveDirectiveArguments(directiveItem.Arguments)
		resolveUnableInput, resolveSkip, err := directiveResolve.Resolve(variableResolver)
		if err != nil {
			i.reportError(directiveResolveErrorFormat, directiveItem.Name, err)
			continue
		}

		if resolveUnableInput {
			unableInput = true
		}
		if resolveSkip {
			skipped = true
		}
	}
	return
}

func (i *QueryDocumentItem) buildSelectionArgumentSchema(isVariableArg bool, definition *ast.Definition, path ...string) (schemaRef *openapi3.SchemaRef) {
	schemaRef = &openapi3.SchemaRef{Ref: interpolate.Openapi3SchemaRefPrefix + definition.Name}
	if isVariableArg {
		if _, ok := i.variablesRefVisited[schemaRef.Ref]; !ok {
			i.variablesRefs = append(i.variablesRefs, schemaRef.Ref)
			i.variablesRefVisited[schemaRef.Ref] = true
		}
	}
	if _, ok := i.usedArgumentDefinitions.Load(definition.Name); ok {
		return
	}

	// 对象类型的入参需要递归进行处理
	objectSchema := openapi3.NewObjectSchema()
	objectSchema.Description = definition.Description
	objectSchema.Nullable = strings.Contains(definition.Name, nullableRequiredKey)
	i.usedArgumentDefinitions.Store(definition.Name, &openapi3.SchemaRef{Value: objectSchema})
	for _, field := range definition.Fields {
		fieldPath := CopyAndAppendItem(path, field.Name, field.Type)
		objectSchema.Properties[field.Name] = i.buildJsonschema(false, field.Description, field.Type, func(fieldDef *ast.Definition) *openapi3.SchemaRef {
			return i.buildSelectionArgumentSchema(isVariableArg, fieldDef, fieldPath...)
		}, fieldPath...)
		if field.Type.NonNull {
			objectSchema.Required = append(objectSchema.Required, field.Name)
		}
	}
	return
}

// 构建jsonschema
func (i *QueryDocumentItem) buildJsonschema(hasSubFields bool, fieldDescription string, fieldType *ast.Type, objectBuild jsonschemaObjectBuildFunc, path ...string) (schemaRef *openapi3.SchemaRef) {
	// graphql中scalar转换为schema中type
	fieldTypeName := fieldType.Name()
	hasSubFields = hasSubFields && datasource.IsScalarJsonName(fieldTypeName)
	format, description := datasource.MatchTypeFormat(fieldDescription)
	schemaRef, ok := directives.BuildSchemaRefForScalar(fieldTypeName, true)
	if ok && !hasSubFields {
		schemaRef.Value.Format = format
	} else {
		schemaRef = i.buildJsonschemaWithDefinition(hasSubFields, fieldTypeName, objectBuild, path...)
	}

	if fieldType.Elem != nil {
		if schemaRef.Value != nil {
			schemaRef.Value.Nullable = !fieldType.Elem.NonNull
		}
		schemaRef = wrapArraySchemaRef(schemaRef)
	}
	if schemaRef.Value != nil {
		schemaRef.Value.Nullable, schemaRef.Value.Description = !fieldType.NonNull, description
	}
	return
}

func (i *QueryDocumentItem) buildJsonschemaWithDefinition(hasSubFields bool, fieldTypeName string, objectBuild jsonschemaObjectBuildFunc, path ...string) (schemaRef *openapi3.SchemaRef) {
	schemaRef = &openapi3.SchemaRef{Value: openapi3.NewSchema()}
	fieldTypeDefinition := i.definitionFetch(fieldTypeName)
	if fieldTypeDefinition == nil {
		i.reportErrorWithPath(fieldDefinitionMissFormat, fieldTypeName, path...)
		return
	}

	// 不同的定义类型执行不同的处理逻辑
	switch fieldTypeDefinition.Kind {
	case ast.Scalar:
		if hasSubFields && datasource.IsScalarJsonName(fieldTypeName) {
			schemaRef = objectBuild(fieldTypeDefinition)
			break
		}

		emptySchema := schemaRef.Value
		if i.handleAdditionTypeOnDescription(fieldTypeDefinition.Description, emptySchema, objectBuild, path...) {
			emptySchema.Title = fieldTypeName
			break
		}

		if scalarSchema, ok := directives.BuildSchemaRefForScalar(fieldTypeName, false); ok {
			schemaRef = scalarSchema
			break
		}

		emptySchema.Type = openapi3.TypeString
		emptySchema.Format = utils.Camel2Case(fieldTypeName, "-")
	case ast.Object, ast.InputObject:
		schemaRef = objectBuild(fieldTypeDefinition)
	case ast.Enum:
		enumSchema := schemaRef.Value
		enumDescriptionMap := make(map[string]string)
		enumSchema.Title, enumSchema.Type, enumSchema.Description = fieldTypeName, openapi3.TypeString, fieldTypeDefinition.Description
		for _, value := range fieldTypeDefinition.EnumValues {
			enumSchema.Enum = append(enumSchema.Enum, value.Name)
			if len(value.Description) > 0 {
				enumDescriptionMap[value.Name] = value.Description
			}
		}
		if len(enumDescriptionMap) > 0 {
			enumSchema.Extensions = map[string]interface{}{EnumDescriptionsKey: enumDescriptionMap}
		}
	}
	return
}

func (i *QueryDocumentItem) handleAdditionTypeOnDescription(fieldTypeDefinitionDescription string, schema *openapi3.Schema, objectBuild jsonschemaObjectBuildFunc, path ...string) (matched bool) {
	additionType, clearedDescription := datasource.MatchAdditionalType(fieldTypeDefinitionDescription)
	if matched = len(additionType) > 0; matched {
		schema.Type = openapi3.TypeObject
		schema.Description = clearedDescription
		if additionType == consts.ScalarJSON {
			trueValue := true
			schema.AdditionalProperties.Has = &trueValue
			return
		}

		additionTypeOrigin := strings.Trim(additionType, utils.ArrayPath)
		additionSchemaRef, ok := directives.BuildSchemaRefForScalar(additionTypeOrigin, true)
		if !ok {
			additionSchemaRef = i.buildJsonschemaWithDefinition(false, additionTypeOrigin, objectBuild, path...)
		}
		schema.AdditionalProperties.Schema = additionSchemaRef
		if additionTypeOrigin != additionType {
			schema.AdditionalProperties.Schema = wrapArraySchemaRef(schema.AdditionalProperties.Schema)
		}
	}
	return
}

func (i *QueryDocumentItem) isScalarDefinition(name string) bool {
	if datasource.IsBaseScalarName(name) || datasource.IsScalarJsonName(name) {
		return true
	}

	definition := i.definitionFetch(name)
	return definition != nil && definition.Kind == ast.Scalar
}

// CopyAndAppendItem 拷贝并追加字符，并根据itemType判断是否添加'[]'
func CopyAndAppendItem(path []string, item string, itemType ...*ast.Type) []string {
	return utils.CopyAndAppendItem(path, item, func() bool { return len(itemType) > 0 && itemType[0].Elem != nil })
}

func wrapArraySchemaRef(schema *openapi3.SchemaRef) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: &openapi3.Schema{
		Type:  openapi3.TypeArray,
		Items: schema,
	}}
}

// 比较类型，额外判断默认值
func isCompatibleType(t, other *ast.Type, tDefault *ast.Value) bool {
	if t.NamedType != other.NamedType {
		return false
	}

	if t.Elem != nil && other.Elem == nil {
		return false
	}

	if t.Elem != nil && !isCompatibleType(t.Elem, other.Elem, nil) {
		return false
	}

	if other.NonNull {
		return t.NonNull || tDefault != nil
	}

	return true
}
