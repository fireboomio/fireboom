// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"github.com/vektah/gqlparser/v2/ast"
)

const firstRawResultName = "firstRawResult"

type firstRawResult struct{}

func (s *firstRawResult) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Name:      firstRawResultName,
		Locations: []ast.DirectiveLocation{ast.LocationField},
	}
}

func (s *firstRawResult) Definitions() ast.DefinitionList {
	return nil
}

func (s *firstRawResult) Resolve(resolver *SelectionResolver) (err error) {
	return
}

func init() {
	registerDirective(firstRawResultName, &firstRawResult{})
}
