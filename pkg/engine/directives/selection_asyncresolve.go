// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
)

const asyncResolveName = "asyncResolve"

type asyncResolve struct{}

func (s *asyncResolve) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.AsyncResolveDesc.String())),
		Name:        asyncResolveName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
	}
}

func (s *asyncResolve) Definitions() ast.DefinitionList {
	return nil
}

func (s *asyncResolve) Resolve(resolver *SelectionResolver) (err error) {
	if resolver.Schema.Value.Type != openapi3.TypeArray {
		err = fmt.Errorf("@%s directive only support on array type", asyncResolveName)
		return
	}
	return
}

func init() {
	registerDirective(asyncResolveName, &asyncResolve{})
}
