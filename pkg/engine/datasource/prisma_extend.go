package datasource

import (
	"fireboom-server/pkg/common/consts"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	optionalRawFieldQueryRaw   = "optional_queryRaw"
	optionalRawFieldExecuteRaw = "optional_executeRaw"
)

var originRawFields = map[string]string{
	optionalRawFieldQueryRaw:   "queryRaw",
	optionalRawFieldExecuteRaw: "executeRaw",
}

func getRawFieldOriginName(fieldName string) string {
	if name, ok := originRawFields[fieldName]; ok {
		fieldName = name
	}
	return fieldName
}

func extendOptionalRawField(document *ast.SchemaDocument) {
	mutations := document.Definitions.ForName(consts.TypeMutation)
	if mutations == nil {
		return
	}
	mutations.Fields = append(mutations.Fields, makeExtendRawField(optionalRawFieldQueryRaw), makeExtendRawField(optionalRawFieldExecuteRaw))
}

func makeExtendRawField(name string) *ast.FieldDefinition {
	return &ast.FieldDefinition{
		Name: name,
		Arguments: []*ast.ArgumentDefinition{{
			Name: "query",
			Type: &ast.Type{NonNull: true, Elem: &ast.Type{NamedType: consts.ScalarString}},
		}},
		Type: &ast.Type{NamedType: consts.ScalarJSON},
	}
}
