// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
)

const internalName = "internal"

type internal struct{}

func (v *internal) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.InternalDesc.String())),
		Name:        internalName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
	}
}

func (v *internal) Definitions() ast.DefinitionList {
	return nil
}

func (v *internal) Resolve(resolver *VariableResolver) (bool, bool, error) {
	_, exported := resolver.VariableExported[resolver.Path[0]]
	return exported, true, nil
}

func init() {
	registerDirective(internalName, &internal{})
}
