// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 目前空置为提供支持（主要未解决不存在参数的判断问题）
*/
package directives

import (
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
)

const hookVariableName = "hookVariable"

type hookVariable struct{ variableDirectiveRemoved }

func (h *hookVariable) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Name:        hookVariableName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Description: appendIfExistExampleGraphql(""),
	}
}

func (h *hookVariable) Definitions() ast.DefinitionList {
	return nil
}

func (h *hookVariable) Resolve(*VariableResolver) (bool, bool, error) {
	return false, false, nil
}

func init() {
	registerDirective(hookVariableName, &hookVariable{})
	apihandler.AddClearVariableDirectiveName(hookVariableName)
}
