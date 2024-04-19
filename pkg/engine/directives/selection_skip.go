// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"github.com/vektah/gqlparser/v2/ast"
)

const skipName = "skip"

type skip struct{}

func (s *skip) Directive() *ast.DirectiveDefinition {
	definition := includeDirective.Directive()
	definition.Name = skipName
	return definition
}

func (s *skip) Definitions() ast.DefinitionList {
	return nil
}

func (s *skip) Resolve(resolver *SelectionResolver) (err error) {
	return includeDirective.Resolve(resolver)
}

func init() {
	registerDirective(skipName, &skip{})
}
