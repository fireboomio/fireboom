// Package datasource
/*
 添加对schema各个字段的访问解析
*/
package datasource

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"strings"
)

const (
	AdditionalSuffix = "Map"
	schemaOneOf      = "oneOf"
	schemaAnyOf      = "anyOf"
	schemaAllOf      = "allOf"
)

var (
	visitSchemaFuncMap      map[int32]visitSchemaFunc
	stringFormatToScalarMap map[string]string
)

type (
	enumFieldDefinition struct {
		definitionName string
		enumValues     []string
	}
	visitSchemaOutput struct {
		kind         ast.DefinitionKind
		name         string
		ofType       *valueDefinition
		defaultValue any
		typeFormat   string
	}
	visitSchemaInput struct {
		rootKind             ast.DefinitionKind
		definitionName       string
		definitionParentPath []string
		path                 []string
	}
	visitSchemaFunc func(*resolveGraphqlSchema, *visitSchemaInput, *openapi3.Schema, bool) visitSchemaOutput
)

func init() {
	visitSchemaFuncMap = make(map[int32]visitSchemaFunc)
	visitSchemaFuncMap[0] = visitSchemaAdditionalProperties
	visitSchemaFuncMap[1] = visitSchemaOneOf
	visitSchemaFuncMap[2] = visitSchemaAllOf
	visitSchemaFuncMap[3] = visitSchemaAnyOf
	visitSchemaFuncMap[4] = visitSchemaEnum
	visitSchemaFuncMap[5] = visitSchemaType

	stringFormatToScalarMap = make(map[string]string)
	stringFormatToScalarMap["byte"] = consts.ScalarBytes
	stringFormatToScalarMap["binary"] = consts.ScalarBinary
	stringFormatToScalarMap["date"] = consts.ScalarDate
	stringFormatToScalarMap["date-time"] = consts.ScalarDateTime
	stringFormatToScalarMap["uuid"] = consts.ScalarUUID
}

func visitSchemaOneOf(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, hasSchemaRefStr bool) visitSchemaOutput {
	return buildMultipleOfDefinition(r, input, schemaOneOf, schemaValue.OneOf, hasSchemaRefStr)
}

func visitSchemaAnyOf(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, hasSchemaRefStr bool) visitSchemaOutput {
	return buildMultipleOfDefinition(r, input, schemaAnyOf, schemaValue.AnyOf, hasSchemaRefStr)
}

func visitSchemaAllOf(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, hasSchemaRefStr bool) visitSchemaOutput {
	return buildMultipleOfDefinition(r, input, schemaAllOf, schemaValue.AllOf, hasSchemaRefStr)
}

func visitSchemaType(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, hasSchemaRefStr bool) (result visitSchemaOutput) {
	if schemaValue.Type == "" {
		if len(schemaValue.Properties) == 0 {
			return
		}
		schemaValue.Type = openapi3.TypeObject
	}

	switch schemaValue.Type {
	case openapi3.TypeArray:
		result.kind = kindList
		arrayVisitInput := &visitSchemaInput{
			rootKind:             input.rootKind,
			definitionParentPath: input.definitionParentPath,
			definitionName:       input.definitionName,
			path:                 utils.CopyAndAppendItem(input.path, utils.ArrayPath),
		}
		result.ofType, result.defaultValue, result.typeFormat = r.visitSchema(arrayVisitInput, schemaValue.Items, false)
		if result.defaultValue != nil {
			result.defaultValue = fmt.Sprintf(kindListFormat, cast.ToString(result.defaultValue))
		}
	case openapi3.TypeObject:
		if len(schemaValue.Properties) == 0 {
			result.kind, result.name = ast.Scalar, consts.ScalarJSON
			return
		}

		objectDef, exist := r.fetchDefinition(input.definitionName, input.rootKind, !hasSchemaRefStr && r.parentRenamed(input))
		defer r.storeCustomRestRewriterQuote(input.rootKind, objectDef.Name, input.path)
		result.kind, result.name = input.rootKind, objectDef.Name
		if exist {
			return
		}

		objectDef.Description = normalizeDescription(schemaValue.Description)
		for key, item := range schemaValue.Properties {
			itemDefinitionName := utils.JoinString("_", objectDef.Name, key)
			itemRequired := slices.Contains(schemaValue.Required, key)
			itemPath := utils.CopyAndAppendItem(input.path, key)
			itemDef := r.normalizeBaseDefinition(key, item.Value.Description, func(normalized string) {
				itemPath[len(itemPath)-1] = normalized
				r.storeCustomRestRewriter(input.rootKind, objectDef.Name, &wgpb.DataSourceRESTRewriter{
					PathComponents: []string{normalized},
					Type:           wgpb.DataSourceRESTRewriterType_fieldRewrite,
					FieldRewriteTo: key,
				})
			})
			itemVisitInput := &visitSchemaInput{
				rootKind:             input.rootKind,
				definitionParentPath: utils.CopyAndAppendItem(input.definitionParentPath, objectDef.Name),
				definitionName:       itemDefinitionName,
				path:                 itemPath,
			}
			itemType, itemDefault, itemTypeFormat := r.visitSchema(itemVisitInput, item, itemRequired)
			addTypeFormatToFieldDescription(&itemDef, itemTypeFormat)
			choiceAppendDefinitionField(objectDef, itemDef, itemType, itemDefault, input.rootKind)
		}
	case openapi3.TypeBoolean:
		result.kind, result.name = ast.Scalar, consts.ScalarBoolean
		if schemaValue.Default != nil {
			result.defaultValue = cast.ToBool(schemaValue.Default)
		}
	case openapi3.TypeInteger:
		result.kind, result.name = ast.Scalar, consts.ScalarInt
		result.typeFormat = schemaValue.Format
		if schemaValue.Default != nil {
			result.defaultValue = cast.ToInt64(schemaValue.Default)
		}
	case openapi3.TypeNumber:
		result.kind, result.name = ast.Scalar, consts.ScalarFloat
		result.typeFormat = schemaValue.Format
		if schemaValue.Default != nil {
			result.defaultValue = cast.ToFloat64(schemaValue.Default)
		}
	case openapi3.TypeString:
		result.kind = ast.Scalar
		if scalarName, ok := stringFormatToScalarMap[schemaValue.Format]; ok {
			result.name = scalarName
		} else {
			result.name = consts.ScalarString
			result.typeFormat = schemaValue.Format
		}
		if schemaValue.Default != nil {
			result.defaultValue = fmt.Sprintf(`"%s"`, strings.Trim(cast.ToString(result.defaultValue), `""`))
		}
	}
	return
}

func visitSchemaEnum(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, _ bool) (result visitSchemaOutput) {
	if schemaValue.Enum == nil {
		return
	}

	enumDef, exist := r.fetchDefinition(input.definitionName, ast.Enum)
	defer r.storeCustomRestRewriterQuote(input.rootKind, enumDef.Name, input.path)
	result.kind, result.name = ast.Enum, enumDef.Name
	if exist {
		return
	}

	valueRewrites := make(map[string]string)
	enumDef.Description = normalizeDescription(schemaValue.Description)
	for _, item := range schemaValue.Enum {
		itemValueDef := &enumValueDefinition{}
		enumItem := cast.ToString(item)
		itemValueDef.baseDefinition = r.normalizeBaseDefinition(enumItem, "", func(normalized string) {
			valueRewrites[normalized] = enumItem
		})
		enumDef.EnumValues = append(enumDef.EnumValues, itemValueDef)
	}
	if input.rootKind == ast.InputObject && len(valueRewrites) > 0 {
		r.storeCustomRestRewriter(input.rootKind, enumDef.Name, &wgpb.DataSourceRESTRewriter{
			Type:          wgpb.DataSourceRESTRewriterType_valueRewrite,
			ValueRewrites: valueRewrites,
		})
	}

	if schemaValue.Default != nil {
		result.defaultValue = utils.NormalizeName(cast.ToString(schemaValue.Default))
	}
	return
}

func visitSchemaAdditionalProperties(r *resolveGraphqlSchema, input *visitSchemaInput, schemaValue *openapi3.Schema, _ bool) (result visitSchemaOutput) {
	additional := schemaValue.AdditionalProperties
	var valueType *valueDefinition
	var valueTypeFormat string
	if additionalHas := additional.Has; additionalHas != nil && *additionalHas {
		valueName := consts.ScalarJSON
		valueType = &valueDefinition{Kind: input.rootKind, Name: &valueName}
	}
	if additionalSchema := additional.Schema; additionalSchema != nil {
		additionalVisitInput := &visitSchemaInput{
			rootKind:             input.rootKind,
			definitionParentPath: input.definitionParentPath,
			definitionName:       input.definitionName,
			path:                 utils.CopyAndAppendItem(input.path, "*"),
		}
		valueType, _, valueTypeFormat = r.visitSchema(additionalVisitInput, additionalSchema, false)
	}
	if valueType == nil {
		return
	}

	additionalTypeName := valueType.string()
	additionalDef, exist := r.fetchDefinition(additionalTypeName+AdditionalSuffix, ast.Scalar)
	result.kind, result.name, result.typeFormat = ast.Scalar, additionalDef.Name, valueTypeFormat
	if exist {
		return
	}

	additionalDef.Description = normalizeDescription(schemaValue.Description) + fmt.Sprintf(additionalTypeFormat, additionalTypeName)
	return
}

func buildSingleDefinition(r *resolveGraphqlSchema, input *visitSchemaInput, schemaRef *openapi3.SchemaRef) (result visitSchemaOutput) {
	itemType, defaultValue, itemTypeFormat := r.visitSchema(input, schemaRef, false)
	result.kind, result.ofType = itemType.Kind, itemType.OfType
	result.defaultValue, result.typeFormat = defaultValue, itemTypeFormat
	if result.ofType == nil {
		result.name = itemType.prototype().string()
	}
	return
}

func buildMultipleOfDefinition(r *resolveGraphqlSchema, input *visitSchemaInput, suffix string, schemaRefs openapi3.SchemaRefs, hasSchemaRefStr bool) (result visitSchemaOutput) {
	schemaRefsLength := len(schemaRefs)
	if schemaRefsLength == 0 {
		return
	}
	if schemaRefsLength == 1 {
		result = buildSingleDefinition(r, input, schemaRefs[0])
		return
	}
	if schemaRefsLength == 2 && matchSchemaWithType(schemaRefs[0], openapi3.TypeString) && matchSchemaWithType(schemaRefs[1], openapi3.TypeInteger) {
		result = buildSingleDefinition(r, input, schemaRefs[1])
		return
	}

	objectDef, exist := r.fetchDefinition(utils.JoinStringWithDot(input.definitionName, suffix), input.rootKind, !hasSchemaRefStr && r.parentRenamed(input))
	defer r.storeCustomRestRewriterQuote(input.rootKind, objectDef.Name, input.path)
	result.kind, result.name = input.rootKind, objectDef.Name
	if exist {
		return
	}

	var (
		itemTypes    []*valueDefinition
		itemDefaults []any
		itemBaseDefs []baseDefinition
		emptyResolve = r.emptyResolve
	)
	if emptyResolve == nil {
		emptyResolve = r
	}
	for _, item := range schemaRefs {
		itemEmptyType, _, _ := emptyResolve.visitSchema(input, item, false)
		itemEmptyKey := strings.Trim(utils.NormalizeName(buildDefinitionName(item, itemEmptyType.string())), "_")
		if slices.ContainsFunc(itemBaseDefs, func(v baseDefinition) bool { return v.Name == itemEmptyKey }) {
			itemEmptyKey += cast.ToString(itemEmptyType.depth())
		}
		multipleVisitInput := &visitSchemaInput{
			rootKind:             input.rootKind,
			definitionParentPath: utils.CopyAndAppendItem(input.definitionParentPath, objectDef.Name),
			definitionName:       itemEmptyType.prototype().string(),
			path:                 utils.CopyAndAppendItem(input.path, itemEmptyKey),
		}
		itemType, itemDefault, itemTypeFormat := r.visitSchema(multipleVisitInput, item, false)
		itemBaseDef := r.normalizeBaseDefinition(itemEmptyKey, item.Value.Description)
		addTypeFormatToFieldDescription(&itemBaseDef, itemTypeFormat)
		itemTypes = append(itemTypes, itemType)
		itemDefaults = append(itemDefaults, itemDefault)
		itemBaseDefs = append(itemBaseDefs, itemBaseDef)
	}

	var (
		enumNames             []string
		enumDef               *definition
		isNotAllOf            bool
		isNotInputObject      bool
		repeatedEnumFieldDefs map[string][]*enumFieldDefinition
		applyAllSubObjects    []*wgpb.DataSourceRESTSubObject
		applyBySubTypes       []*wgpb.DataSourceRESTSubfield
	)
	if isNotAllOf = suffix != schemaAllOf; isNotAllOf {
		enumDef, _ = r.fetchDefinition(objectDef.Name, ast.Enum)
	}
	if isNotInputObject = input.rootKind != ast.InputObject; isNotInputObject {
		repeatedEnumFieldDefs = make(map[string][]*enumFieldDefinition)
	}
	for i, itemType := range itemTypes {
		itemBaseDef, itemDefault := itemBaseDefs[i], itemDefaults[i]
		choiceAppendDefinitionField(objectDef, itemBaseDef, itemType, itemDefault, input.rootKind)
		itemEnumDefinition := r.normalizeBaseDefinition(itemBaseDef.Name, "")
		enumNames = append(enumNames, itemEnumDefinition.Name)
		if isNotInputObject {
			if itemDef, ok := r.fetchExistedDefinition(itemType, ast.Object); ok {
				var itemSubFields []*wgpb.DataSourceRESTSubfield
				for _, itemDefField := range itemDef.Fields {
					itemSubFields = append(itemSubFields, &wgpb.DataSourceRESTSubfield{
						Name: utils.FirstNotEmptyString(itemDefField.originName, itemDefField.Name),
						Type: int32(itemDefField.Type.valueType()),
					})
					if existedDef, existed := r.fetchExistedDefinition(itemDefField.Type, ast.Enum); existed {
						var enumValues []string
						for _, itemDefFieldEnumValue := range existedDef.EnumValues {
							enumValues = append(enumValues, utils.FirstNotEmptyString(itemDefFieldEnumValue.originName, itemDefFieldEnumValue.Name))
						}
						repeatedEnumFieldDefs[existedDef.Name] = append(repeatedEnumFieldDefs[existedDef.Name], &enumFieldDefinition{
							definitionName: itemEnumDefinition.Name,
							enumValues:     enumValues,
						})
					}
				}
				if len(itemSubFields) > 0 {
					applyAllSubObjects = append(applyAllSubObjects, &wgpb.DataSourceRESTSubObject{Name: itemEnumDefinition.Name, Fields: itemSubFields})
				}
			} else {
				applyBySubTypes = append(applyBySubTypes, &wgpb.DataSourceRESTSubfield{
					Name: itemEnumDefinition.Name,
					Type: int32(itemType.valueType()),
				})
			}
		}
		if isNotAllOf {
			enumDef.EnumValues = append(enumDef.EnumValues, &enumValueDefinition{baseDefinition: itemEnumDefinition})
		}
	}
	if isNotAllOf {
		extractOfType := &valueDefinition{Kind: KindNonNull, OfType: &valueDefinition{Kind: ast.Enum, Name: &enumDef.Name}}
		choiceAppendDefinitionField(objectDef, baseDefinition{Name: suffix}, extractOfType, nil, input.rootKind)
	}
	rewriter := &wgpb.DataSourceRESTRewriter{CustomObjectName: objectDef.Name}
	if isNotInputObject {
		if isNotAllOf {
			rewriter.CustomEnumField = suffix
			if len(repeatedEnumFieldDefs) > 0 {
				expectedCount := len(itemTypes)
				for key, defs := range repeatedEnumFieldDefs {
					if len(defs) == expectedCount {
						rewriter.Type = wgpb.DataSourceRESTRewriterType_applyBySubCommonFieldValue
						rewriter.ApplySubCommonField, rewriter.ApplySubCommonFieldValues = key, make(map[string]string)
						for _, def := range defs {
							for _, defEnumValue := range def.enumValues {
								rewriter.ApplySubCommonFieldValues[defEnumValue] = def.definitionName
							}
						}
						break
					}
				}
			} else if len(applyBySubTypes) > 0 {
				rewriter.Type, rewriter.ApplySubFieldTypes = wgpb.DataSourceRESTRewriterType_applyBySubfieldType, applyBySubTypes
			}
		} else {
			rewriter.Type, rewriter.ApplySubObjects = wgpb.DataSourceRESTRewriterType_applyAllSubObject, applyAllSubObjects
		}
	} else {
		if isNotAllOf {
			rewriter.Type, rewriter.CustomEnumField = wgpb.DataSourceRESTRewriterType_extractCustomEnumFieldValue, suffix
		} else {
			rewriter.Type = wgpb.DataSourceRESTRewriterType_extractAllSubfield
		}
	}
	if rewriter.Type != wgpb.DataSourceRESTRewriterType_quoteObject {
		r.storeCustomRestRewriter(input.rootKind, objectDef.Name, rewriter)
	}
	return
}

func choiceAppendDefinitionField(objectDef *definition, itemDef baseDefinition, itemType *valueDefinition, itemDefault any, rootKind ast.DefinitionKind) {
	if rootKind == ast.InputObject {
		objectDef.InputFields = append(objectDef.InputFields, &argumentDefinition{
			baseDefinition: itemDef,
			Type:           itemType,
			DefaultValue:   itemDefault,
		})
	} else {
		objectDef.Fields = append(objectDef.Fields, &fieldDefinition{
			baseDefinition: itemDef,
			Type:           itemType,
		})
	}
	return
}

func matchSchemaWithType(schemaRef *openapi3.SchemaRef, schemaType string) bool {
	return schemaRef.Value != nil && schemaRef.Value.Type == schemaType
}
