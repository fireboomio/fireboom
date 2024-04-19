package utils

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/invopop/jsonschema"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"reflect"
	"strings"
)

const jsonschemaRefPrefix = "#/$defs/"

// ReflectStructToOpenapi3Schema 反射golang中的结构体成jsonschema并规范化输出为openapi3.Schema
func ReflectStructToOpenapi3Schema(data any, schemas openapi3.Schemas, customSchemaPrefix ...string) string {
	reflector := &jsonschema.Reflector{AllowAdditionalProperties: true, Namer: uppercaseFirstNamer}
	dataSchema := reflector.Reflect(data)
	schemaRefPrefix := interpolate.Openapi3SchemaRefPrefix
	if len(customSchemaPrefix) > 0 {
		schemaRefPrefix = customSchemaPrefix[0]
	}
	for name, schema := range dataSchema.Definitions {
		schemas[name] = parseJsonschemaToOpenapi3Schema(schema, schemaRefPrefix)
	}

	return strings.TrimPrefix(dataSchema.Ref, jsonschemaRefPrefix)
}

func GetTypeName(data any) string {
	return uppercaseFirstNamer(reflect.TypeOf(data))
}

func uppercaseFirstNamer(t reflect.Type) string {
	return UppercaseFirst(t.Name())
}

func parseJsonschemaToOpenapi3Schema(schema *jsonschema.Schema, schemaRefPrefix string) (result *openapi3.SchemaRef) {
	if schema.Ref != "" {
		result = openapi3.NewSchemaRef(strings.ReplaceAll(schema.Ref, jsonschemaRefPrefix, schemaRefPrefix), nil)
		return
	}

	result = &openapi3.SchemaRef{Value: &openapi3.Schema{
		Type:     schema.Type,
		Format:   schema.Format,
		Default:  schema.Default,
		Title:    schema.Title,
		Required: schema.Required,
	}}
	if items := schema.Items; items != nil {
		result.Value.Items = parseJsonschemaToOpenapi3Schema(items, schemaRefPrefix)
		return
	}

	if enum := schema.Enum; enum != nil {
		result.Value.Enum = enum
		return
	}

	if _, ok := schema.Extras["contentEncoding"]; ok || schema.ContentEncoding != "" {
		result.Value = openapi3.NewBytesSchema()
	}

	if oneOfLen := len(schema.OneOf); oneOfLen > 0 {
		result = parseJsonschemaToOpenapi3Schema(schema.OneOf[oneOfLen-1], schemaRefPrefix)
		return
	}

	for _, item := range schema.AnyOf {
		result.Value.AnyOf = append(result.Value.AnyOf, parseJsonschemaToOpenapi3Schema(item, schemaRefPrefix))
	}
	for _, item := range schema.AllOf {
		result.Value.AllOf = append(result.Value.AllOf, parseJsonschemaToOpenapi3Schema(item, schemaRefPrefix))
	}
	for _, item := range schema.PatternProperties {
		result.Value.AdditionalProperties.Schema = parseJsonschemaToOpenapi3Schema(item, schemaRefPrefix)
	}

	if properties := schema.Properties; properties != nil {
		result.Value.Properties = make(openapi3.Schemas)
		for _, key := range properties.Keys() {
			itemSchema, _ := properties.Get(key)
			result.Value.Properties[key] = parseJsonschemaToOpenapi3Schema(itemSchema.(*jsonschema.Schema), schemaRefPrefix)
		}
	}
	return
}
