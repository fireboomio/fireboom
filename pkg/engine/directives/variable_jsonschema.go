// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 改写入参的jsonschema，引擎层会作在接口调用时作参数校验
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	jsonSchemaName                     = "jsonSchema"
	jsonSchemaArgCommonPatternEnumName = "COMMON_REGEX_PATTERN"
)

type (
	jsonSchema         struct{}
	jsonSchemaArgument struct {
		description  string
		namedType    string
		updateSchema func(string, *openapi3.Schema)
	}
)

func (j *jsonSchema) Directive() *ast.DirectiveDefinition {
	var arguments ast.ArgumentDefinitionList
	for k, v := range jsonSchemaArgMap {
		arguments = append(arguments, &ast.ArgumentDefinition{
			Name:        k,
			Description: v.description,
			Type:        ast.NamedType(v.namedType, nil),
		})
	}
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.JsonschemaDesc.String()),
		Name:        jsonSchemaName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments:   arguments,
	}
}

func (j *jsonSchema) Definitions() ast.DefinitionList {
	var enumValues ast.EnumValueList
	for k := range jsonSchemaCommonPatternEnumMap {
		enumValues = append(enumValues, &ast.EnumValueDefinition{Name: k})
	}
	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       jsonSchemaArgCommonPatternEnumName,
		EnumValues: enumValues,
	}}
}

func (j *jsonSchema) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	for name, value := range resolver.Arguments {
		arg, ok := jsonSchemaArgMap[name]
		if !ok || arg.updateSchema == nil {
			continue
		}

		arg.updateSchema(value, resolver.Schema.Value)
	}
	return
}

const (
	title            = "title"
	description      = "description"
	multipleOf       = "multipleOf"
	maximum          = "maximum"
	exclusiveMaximum = "exclusiveMaximum"
	minimum          = "minimum"
	exclusiveMinimum = "exclusiveMinimum"
	maxLength        = "maxLength"
	minLength        = "minLength"
	maxItems         = "maxItems"
	minItems         = "minItems"
	uniqueItems      = "uniqueItems"
	pattern          = "pattern"
	commonPattern    = "commonPattern"
)

var (
	jsonSchemaCommonPatternEnumMap map[string]string
	jsonSchemaArgMap               map[string]*jsonSchemaArgument
)

func init() {
	registerDirective(jsonSchemaName, &jsonSchema{})

	jsonSchemaCommonPatternEnumMap = make(map[string]string)
	jsonSchemaCommonPatternEnumMap["EMAIL"] = "(?:[a-z0-9!#$%&\\'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&\\'*+/=?^_`{|}~-]+)*|\"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\\\[\x01-\x09\x0b\x0c\x0e-\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\\])"
	jsonSchemaCommonPatternEnumMap["DOMAIN"] = "^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}$"
	jsonSchemaCommonPatternEnumMap["URL"] = "/(((http|ftp|https):\\/{2})+(([0-9a-z_-]+\\.)+(aero|asia|biz|cat|com|coop|edu|gov|info|int|jobs|mil|mobi|museum|name|net|org|pro|tel|travel|ac|ad|ae|af|ag|ai|al|am|an|ao|aq|ar|as|at|au|aw|ax|az|ba|bb|bd|be|bf|bg|bh|bi|bj|bm|bn|bo|br|bs|bt|bv|bw|by|bz|ca|cc|cd|cf|cg|ch|ci|ck|cl|cm|cn|co|cr|cu|cv|cx|cy|cz|cz|de|dj|dk|dm|do|dz|ec|ee|eg|er|es|et|eu|fi|fj|fk|fm|fo|fr|ga|gb|gd|ge|gf|gg|gh|gi|gl|gm|gn|gp|gq|gr|gs|gt|gu|gw|gy|hk|hm|hn|hr|ht|hu|id|ie|il|im|in|io|iq|ir|is|it|je|jm|jo|jp|ke|kg|kh|ki|km|kn|kp|kr|kw|ky|kz|la|lb|lc|li|lk|lr|ls|lt|lu|lv|ly|ma|mc|md|me|mg|mh|mk|ml|mn|mn|mo|mp|mr|ms|mt|mu|mv|mw|mx|my|mz|na|nc|ne|nf|ng|ni|nl|no|np|nr|nu|nz|nom|pa|pe|pf|pg|ph|pk|pl|pm|pn|pr|ps|pt|pw|py|qa|re|ra|rs|ru|rw|sa|sb|sc|sd|se|sg|sh|si|sj|sj|sk|sl|sm|sn|so|sr|st|su|sv|sy|sz|tc|td|tf|tg|th|tj|tk|tl|tm|tn|to|tp|tr|tt|tv|tw|tz|ua|ug|uk|us|uy|uz|va|vc|ve|vg|vi|vn|vu|wf|ws|ye|yt|yu|za|zm|zw|arpa)(:[0-9]+)?((\\/([~0-9a-zA-Z\\#\\+\\%@\\.\\/_-]+))?(\\?[0-9a-zA-Z\\+\\%@\\/&\\[\\];=_-]+)?)?))\\b/imuS\n"

	jsonSchemaArgMap = make(map[string]*jsonSchemaArgument)
	jsonSchemaArgMap[title] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.Title = value },
		namedType:    consts.ScalarString,
	}
	jsonSchemaArgMap[description] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.Description = value },
		namedType:    consts.ScalarString,
	}
	jsonSchemaArgMap[multipleOf] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			castValue := cast.ToFloat64(value)
			schema.MultipleOf = &castValue
		},
		namedType: consts.ScalarFloat,
	}
	jsonSchemaArgMap[maximum] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			castValue := cast.ToFloat64(value)
			schema.Max = &castValue
		},
		namedType:   consts.ScalarFloat,
		description: i18n.JsonschemaArgMaximumDesc.String(),
	}
	jsonSchemaArgMap[exclusiveMaximum] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.ExclusiveMax = cast.ToBool(value) },
		namedType:    consts.ScalarBoolean,
	}
	jsonSchemaArgMap[minimum] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			castValue := cast.ToFloat64(value)
			schema.Min = &castValue
		},
		namedType:   consts.ScalarFloat,
		description: i18n.JsonschemaArgMinimumDesc.String(),
	}
	jsonSchemaArgMap[exclusiveMinimum] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.ExclusiveMin = cast.ToBool(value) },
		namedType:    consts.ScalarBoolean,
	}
	jsonSchemaArgMap[maxLength] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			castValue := cast.ToUint64(value)
			schema.MaxLength = &castValue
		},
		namedType:   consts.ScalarInt,
		description: i18n.JsonschemaArgMaxLengthDesc.String(),
	}
	jsonSchemaArgMap[minLength] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.MinLength = cast.ToUint64(value) },
		namedType:    consts.ScalarInt,
		description:  i18n.JsonschemaArgMinLengthDesc.String(),
	}
	jsonSchemaArgMap[maxItems] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			castValue := cast.ToUint64(value)
			schema.MaxItems = &castValue
		},
		namedType:   consts.ScalarInt,
		description: i18n.JsonschemaArgMaxItemsDesc.String(),
	}
	jsonSchemaArgMap[minItems] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.MinItems = cast.ToUint64(value) },
		namedType:    consts.ScalarInt,
		description:  i18n.JsonschemaArgMinItemsDesc.String(),
	}
	jsonSchemaArgMap[uniqueItems] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.UniqueItems = cast.ToBool(value) },
		namedType:    consts.ScalarBoolean,
		description:  i18n.JsonschemaArgUniqueItemsDesc.String(),
	}
	jsonSchemaArgMap[pattern] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) { schema.Pattern = value },
		namedType:    consts.ScalarString,
		description:  i18n.JsonschemaArgPatternDesc.String(),
	}
	jsonSchemaArgMap[commonPattern] = &jsonSchemaArgument{
		updateSchema: func(value string, schema *openapi3.Schema) {
			schema.Pattern = jsonSchemaCommonPatternEnumMap[value]
		},
		namedType:   jsonSchemaArgCommonPatternEnumName,
		description: i18n.JsonschemaArgCommonPatternDesc.String(),
	}
}
