// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve判断导出的参数是否在传递参数中定义
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	wdgast "github.com/wundergraph/graphql-go-tools/pkg/ast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"strings"
)

const (
	customizedFieldName        = "customizedField"
	customizedFieldArgType     = "type"
	customizedFieldArgDesc     = "desc"
	customizedFieldArgTypeType = "CustomizedFieldType"
)

type customizedField struct{ selectionFieldCustomized }

func (s *customizedField) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.CustomizedFieldDesc.String()),
		Name:        customizedFieldName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{
			{
				Name: customizedFieldArgType,
				Type: ast.NonNullNamedType(customizedFieldArgTypeType, nil),
			},
			{
				Name: customizedFieldArgDesc,
				Type: ast.NamedType(consts.ScalarString, nil),
			},
		},
	}
}

func (s *customizedField) Definitions() ast.DefinitionList {
	var typeEnumValues ast.EnumValueList
	for k := range scalarToSchemaMap {
		typeEnumValues = append(typeEnumValues, &ast.EnumValueDefinition{Name: k})
	}
	typeEnumValues = append(typeEnumValues, &ast.EnumValueDefinition{Name: utils.UppercaseFirst(openapi3.TypeArray)})
	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       customizedFieldArgTypeType,
		EnumValues: typeEnumValues,
	}}
}

func (s *customizedField) Resolve(resolver *SelectionResolver) (err error) {
	value, ok := resolver.Arguments[customizedFieldArgType]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, customizedFieldArgType)
		return
	}
	isArray := value == utils.UppercaseFirst(openapi3.TypeArray)
	if !isArray {
		if resolver.Schema, ok = BuildSchemaRefForScalar(value, false); !ok {
			err = fmt.Errorf(argumentValueNotSupportedFormat, value, customizedFieldArgType)
			return
		}
	}
	desc, ok := resolver.Arguments[customizedFieldArgDesc]
	if ok {
		schemaValue := *resolver.Schema.Value
		schemaValue.Description = desc
		resolver.Schema.Value = &schemaValue
	}
	if isArray && resolver.Schema.Value.Type != openapi3.TypeArray {
		schema := *resolver.Schema
		resolver.Schema.Value = &openapi3.Schema{Type: openapi3.TypeArray, Items: &schema}
	}
	return
}

var scalarToSchemaMap map[string]*openapi3.Schema

func BuildSchemaRefForScalar(scalarName string, stringFormatFilter bool) (schemaRef *openapi3.SchemaRef, ok bool) {
	schema, ok := scalarToSchemaMap[scalarName]
	if ok {
		if stringFormatFilter && schema.Format != "" {
			ok = false
			return
		}
		cloneSchema := *schema
		schemaRef = &openapi3.SchemaRef{Value: &cloneSchema}
	}
	return
}

func init() {
	registerDirective(customizedFieldName, &customizedField{})
	apihandler.AddClearFieldDirectiveName(customizedFieldName)

	scalarUnknown := utils.UppercaseFirst(jsonparser.Unknown.String())
	scalarToSchemaMap = make(map[string]*openapi3.Schema)
	scalarToSchemaMap[consts.ScalarBoolean] = openapi3.NewBoolSchema()
	scalarToSchemaMap[consts.ScalarInt] = openapi3.NewIntegerSchema()
	scalarToSchemaMap[consts.ScalarFloat] = openapi3.NewFloat64Schema()
	scalarToSchemaMap[consts.ScalarString] = openapi3.NewStringSchema()
	scalarToSchemaMap[consts.ScalarID] = openapi3.NewStringSchema()
	scalarToSchemaMap[consts.ScalarJSON] = openapi3.NewSchema()
	scalarToSchemaMap[scalarUnknown] = openapi3.NewSchema()
	scalarToSchemaMap[consts.ScalarBytes] = openapi3.NewBytesSchema()
	scalarToSchemaMap[consts.ScalarDate] = &openapi3.Schema{Type: openapi3.TypeString, Format: strings.ToLower(consts.ScalarDate)}
	scalarToSchemaMap[consts.ScalarDateTime] = openapi3.NewDateTimeSchema()
	scalarToSchemaMap[consts.ScalarUUID] = openapi3.NewUUIDSchema()
	scalarToSchemaMap[consts.ScalarBigInt] = &openapi3.Schema{Type: openapi3.TypeString, Format: strings.ToLower(consts.ScalarBigInt)}
	scalarToSchemaMap[consts.ScalarDecimal] = &openapi3.Schema{Type: openapi3.TypeString, Format: strings.ToLower(consts.ScalarDecimal)}
	scalarToSchemaMap[consts.ScalarGeometry] = &openapi3.Schema{Type: openapi3.TypeString, Format: strings.ToLower(consts.ScalarGeometry)}

	prismaTypeToFieldArgType := make(map[string]string)
	prismaTypeToFieldArgType["bool"] = consts.ScalarBoolean
	prismaTypeToFieldArgType["int"] = consts.ScalarInt
	prismaTypeToFieldArgType["float"] = consts.ScalarFloat
	prismaTypeToFieldArgType["string"] = consts.ScalarString
	prismaTypeToFieldArgType["enum"] = consts.ScalarString
	prismaTypeToFieldArgType["json"] = consts.ScalarJSON
	prismaTypeToFieldArgType["null"] = scalarUnknown
	prismaTypeToFieldArgType["date"] = consts.ScalarDate
	prismaTypeToFieldArgType["datetime"] = consts.ScalarDateTime
	prismaTypeToFieldArgType["bigint"] = consts.ScalarBigInt
	prismaTypeToFieldArgType["bytes"] = consts.ScalarBytes
	prismaTypeToFieldArgType["decimal"] = consts.ScalarDecimal
	prismaTypeToFieldArgType["uuid"] = consts.ScalarUUID
	apihandler.AddBuildFieldDirectiveFunc(func(_, prismaType string) (directiveName string, args []apihandler.DirectiveArgument) {
		directiveName = customizedFieldName
		argType, ok := prismaTypeToFieldArgType[prismaType]
		if !ok {
			argType = consts.ScalarString
			args = append(args, apihandler.DirectiveArgument{
				Name:  customizedFieldArgDesc,
				Value: fmt.Sprintf("unsupported prisma type [%s]", prismaType),
			})
		}
		args = append(args, apihandler.DirectiveArgument{
			Name:      customizedFieldArgType,
			Value:     argType,
			ValueKind: wdgast.ValueKindEnum,
		})
		return
	})
}
