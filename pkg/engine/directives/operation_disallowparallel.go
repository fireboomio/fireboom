// Package directives
/*
 实现OperationDirective接口，只能定义在LocationQuery, LocationMutation, LocationSubscription上
 Resolve 标识operation禁止并行解析
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/engineconfigloader"
)

const disallowParallelName = "disallowParallel"

type disallowParallel struct{}

func (o *disallowParallel) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.DisallowParallelDesc.String())),
		Name:        disallowParallelName,
		Locations:   []ast.DirectiveLocation{ast.LocationQuery, ast.LocationMutation},
	}
}

func (o *disallowParallel) Definitions() ast.DefinitionList {
	return nil
}

func (o *disallowParallel) Resolve(*OperationResolver) error {
	return nil
}

func init() {
	registerDirective(disallowParallelName, &disallowParallel{})
	engineconfigloader.AddDisallowParallelFetchDirective(disallowParallelName)
}
