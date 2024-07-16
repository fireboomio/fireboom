// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 方法，根据变量类型，调用相应的解析方法
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"strings"
)

const (
	whereInputName                      = "whereInput"
	whereInputExpectedWhereInputFormat  = "expected [*WhereInput], but found [%s] on path [%s]"
	whereInputWhereInputMissTypeFormat  = "not found argument definition [%s]"
	whereInputWhereInputMissFieldFormat = "not found field [%s] argument definition [%s]"

	whereInputType                              = "WhereInput"
	whereInputFieldNotName                      = "not"
	whereInputFieldFilterName                   = "filter"
	whereInputFieldFilterType                   = "WhereInputFilter"
	whereInputFilterFieldFieldName              = "field"
	whereInputFilterFieldScalarName             = "scalar"
	whereInputFilterFieldScalarType             = "WhereInputScalarFilter"
	whereInputFilterFieldRelationName           = "relation"
	whereInputFilterFieldRelationType           = "WhereInputRelationFilter"
	whereInputFilterCommonFieldTypeName         = "type"
	whereInputFilterFieldFilterTypeType         = "WhereInputFilterType"
	whereInputFilterFieldRelationFilterTypeType = "WhereInputRelationFilterType"
	whereInputScalarFilterFieldInsensitiveName  = "insensitive"
	whereInputRelationFilterFieldWhereName      = "where"
)

type whereInput struct{}

func (v *whereInput) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.WhereInputDesc.String()),
		Name:        whereInputName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments: ast.ArgumentDefinitionList{
			{
				Description: i18n.WhereInputFieldNotDesc.String(),
				Name:        whereInputFieldNotName,
				Type:        ast.NamedType(whereInputType, nil),
			}, {
				Description: i18n.WhereInputFieldFilterDesc.String(),
				Name:        whereInputFieldFilterName,
				Type:        ast.NamedType(whereInputFieldFilterType, nil),
			},
		},
	}
}

func (v *whereInput) Definitions() ast.DefinitionList {
	var filterTypeEnumValues ast.EnumValueList
	for k := range wgpb.VariableWhereInputScalarFilterType_value {
		filterTypeEnumValues = append(filterTypeEnumValues, &ast.EnumValueDefinition{Name: k})
	}
	var relationFilterTypeEnumValues ast.EnumValueList
	for k := range wgpb.VariableWhereInputRelationFilterType_value {
		relationFilterTypeEnumValues = append(relationFilterTypeEnumValues, &ast.EnumValueDefinition{Name: k})
	}

	return ast.DefinitionList{
		{
			Kind: ast.InputObject,
			Name: whereInputType,
			Fields: ast.FieldList{
				{
					Description: i18n.WhereInputFieldNotDesc.String(),
					Name:        whereInputFieldNotName,
					Type:        ast.NamedType(whereInputType, nil),
				}, {
					Description: i18n.WhereInputFieldFilterDesc.String(),
					Name:        whereInputFieldFilterName,
					Type:        ast.NamedType(whereInputFieldFilterType, nil),
				},
			},
		},
		{
			Kind: ast.InputObject,
			Name: whereInputFieldFilterType,
			Fields: ast.FieldList{
				{
					Description: i18n.WhereInputFilterFieldFieldDesc.String(),
					Name:        whereInputFilterFieldFieldName,
					Type:        ast.NonNullNamedType(consts.ScalarString, nil),
				}, {
					Description: i18n.WhereInputFilterFieldScalarDesc.String(),
					Name:        whereInputFilterFieldScalarName,
					Type:        ast.NamedType(whereInputFilterFieldScalarType, nil),
				}, {
					Description: i18n.WhereInputFilterFieldRelationDesc.String(),
					Name:        whereInputFilterFieldRelationName,
					Type:        ast.NamedType(whereInputFilterFieldRelationType, nil),
				},
			},
		},
		{
			Kind: ast.InputObject,
			Name: whereInputFilterFieldScalarType,
			Fields: ast.FieldList{
				{
					Description: i18n.WhereInputFilterCommonFieldTypeDesc.String(),
					Name:        whereInputFilterCommonFieldTypeName,
					Type:        ast.NonNullNamedType(whereInputFilterFieldFilterTypeType, nil),
				}, {
					Description: i18n.WhereInputScalarFilterFieldInsensitiveDesc.String(),
					Name:        whereInputScalarFilterFieldInsensitiveName,
					Type:        ast.NamedType(consts.ScalarBoolean, nil),
				},
			},
		},
		{
			Kind: ast.InputObject,
			Name: whereInputFilterFieldRelationType,
			Fields: ast.FieldList{
				{
					Description: i18n.WhereInputFilterCommonFieldTypeDesc.String(),
					Name:        whereInputFilterCommonFieldTypeName,
					Type:        ast.NonNullNamedType(whereInputFilterFieldRelationFilterTypeType, nil),
				}, {
					Description: i18n.WhereInputRelationFilterFieldWhereDesc.String(),
					Name:        whereInputRelationFilterFieldWhereName,
					Type:        ast.NonNullNamedType(whereInputType, nil),
				},
			},
		},
		{
			Kind:       ast.Enum,
			Name:       whereInputFilterFieldFilterTypeType,
			EnumValues: filterTypeEnumValues,
		},
		{
			Kind:       ast.Enum,
			Name:       whereInputFilterFieldRelationFilterTypeType,
			EnumValues: relationFilterTypeEnumValues,
		},
	}
}

func (v *whereInput) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	schema := resolver.Schema
	if schema.Value != nil && schema.Value.Items != nil {
		schema = schema.Value.Items
	}
	whereInputDefinitionName := strings.TrimPrefix(schema.Ref, interpolate.Openapi3SchemaRefPrefix)
	if !strings.HasSuffix(whereInputDefinitionName, whereInputType) {
		err = fmt.Errorf(whereInputExpectedWhereInputFormat, whereInputDefinitionName, utils.JoinStringWithDot(resolver.Path...))
		return
	}

	variableWhere := &wgpb.VariableWhereInput{}
	if argValue, ok := resolver.Arguments[whereInputFieldNotName]; ok {
		skip = true
		variableWhere.Not = &wgpb.VariableWhereInput{}
		err = v.resolveWhereInput(variableWhere.Not, []byte(argValue), schema, resolver.ArgumentDefinitions)
	} else if argValue, ok = resolver.Arguments[whereInputFieldFilterName]; ok {
		skip = true
		variableWhere.Filter = &wgpb.VariableWhereInputFilter{}
		err = v.resolveWhereInputFilter(variableWhere.Filter, []byte(argValue), schema, resolver.ArgumentDefinitions)
	}
	if err != nil || !skip {
		return
	}

	resolver.Operation.VariablesConfiguration.WhereInputs = append(resolver.Operation.VariablesConfiguration.WhereInputs, &wgpb.VariableWhereInputConfiguration{
		VariablePathComponents: resolver.Path,
		WhereInput:             variableWhere,
	})
	return
}

var (
	whereInputTypeKeys               = [][]string{{whereInputFieldNotName}, {whereInputFieldFilterName}}
	whereInputFilterTypeKeys         = [][]string{{whereInputFilterFieldFieldName}, {whereInputFilterFieldScalarName}, {whereInputFilterFieldRelationName}}
	whereInputScalarFilterTypeKeys   = [][]string{{whereInputFilterCommonFieldTypeName}, {whereInputScalarFilterFieldInsensitiveName}}
	whereInputRelationFilterTypeKeys = [][]string{{whereInputFilterCommonFieldTypeName}, {whereInputRelationFilterFieldWhereName}}
)

func (v *whereInput) resolveWhereInput(variableWhere *wgpb.VariableWhereInput, data []byte, schema *openapi3.SchemaRef, argumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef]) (err error) {
	jsonparser.EachKey(data, func(i int, bytes []byte, _ jsonparser.ValueType, _ error) {
		if err != nil {
			return
		}
		switch i {
		case 0:
			variableWhere.Not = &wgpb.VariableWhereInput{}
			err = v.resolveWhereInput(variableWhere.Not, bytes, schema, argumentDefinitions)
		case 1:
			variableWhere.Filter = &wgpb.VariableWhereInputFilter{}
			err = v.resolveWhereInputFilter(variableWhere.Filter, bytes, schema, argumentDefinitions)
		}
	}, whereInputTypeKeys...)
	return
}

func (v *whereInput) resolveWhereInputFilter(variableFilter *wgpb.VariableWhereInputFilter, data []byte, schema *openapi3.SchemaRef, argumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef]) (err error) {
	jsonparser.EachKey(data, func(i int, bytes []byte, _ jsonparser.ValueType, _ error) {
		if err != nil {
			return
		}
		switch i {
		case 0:
			variableFilter.Field = string(bytes)
			schema, err = searchFieldSchemaOnDefinitionProperties(argumentDefinitions, schema, variableFilter.Field)
		case 1:
			variableFilter.Scalar = &wgpb.VariableWhereInputScalarFilter{}
			err = v.resolveWhereInputScalarFilter(variableFilter.Scalar, bytes, schema, argumentDefinitions)
		case 2:
			variableFilter.Relation = &wgpb.VariableWhereInputRelationFilter{}
			err = v.resolveWhereInputRelationFilter(variableFilter.Relation, bytes, schema, argumentDefinitions)
		}
	}, whereInputFilterTypeKeys...)
	return
}

func (v *whereInput) resolveWhereInputScalarFilter(scalarFilter *wgpb.VariableWhereInputScalarFilter, data []byte, schema *openapi3.SchemaRef, argumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef]) (err error) {
	jsonparser.EachKey(data, func(i int, bytes []byte, _ jsonparser.ValueType, _ error) {
		if err != nil {
			return
		}
		switch i {
		case 0:
			scalarType := string(bytes)
			scalarFilter.Type = wgpb.VariableWhereInputScalarFilterType(wgpb.VariableWhereInputScalarFilterType_value[scalarType])
			schema, err = searchFieldSchemaOnDefinitionProperties(argumentDefinitions, schema, scalarType)
		case 1:
			scalarFilter.Insensitive = cast.ToBool(string(bytes))
		}
	}, whereInputScalarFilterTypeKeys...)
	return
}

func (v *whereInput) resolveWhereInputRelationFilter(relationFilter *wgpb.VariableWhereInputRelationFilter, data []byte, schema *openapi3.SchemaRef, argumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef]) (err error) {
	jsonparser.EachKey(data, func(i int, bytes []byte, _ jsonparser.ValueType, _ error) {
		if err != nil {
			return
		}
		switch i {
		case 0:
			relationType := string(bytes)
			relationFilter.Type = wgpb.VariableWhereInputRelationFilterType(wgpb.VariableWhereInputRelationFilterType_value[relationType])
			schema, err = searchFieldSchemaOnDefinitionProperties(argumentDefinitions, schema, relationType)
		case 1:
			relationFilter.Where = &wgpb.VariableWhereInput{}
			err = v.resolveWhereInput(relationFilter.Where, bytes, schema, argumentDefinitions)
		}
	}, whereInputRelationFilterTypeKeys...)
	return
}

func searchFieldSchemaOnDefinitionProperties(argumentDefinitions *utils.SyncMap[string, *openapi3.SchemaRef], schemaRef *openapi3.SchemaRef, field string) (*openapi3.SchemaRef, error) {
	definitionName := strings.TrimPrefix(schemaRef.Ref, interpolate.Openapi3SchemaRefPrefix)
	schemaRef, ok := argumentDefinitions.Load(definitionName)
	if !ok || schemaRef.Value == nil {
		return nil, fmt.Errorf(whereInputWhereInputMissTypeFormat, definitionName)
	}

	if schemaRef = schemaRef.Value.Properties[field]; schemaRef == nil || schemaRef.Value == nil && schemaRef.Ref == "" {
		return nil, fmt.Errorf(whereInputWhereInputMissFieldFormat, field, definitionName)
	}
	return schemaRef, nil
}

func init() {
	registerDirective(whereInputName, &whereInput{})
}
