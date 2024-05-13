// Package directives
/*
 实现OperationDirective接口，只能定义在LocationQuery, LocationMutation, LocationSubscription上
 Resolve 标识operation内部，引擎会仅注册/internal路由
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
)

const internalOperationName = "internalOperation"

type internalOperation struct{}

func (o *internalOperation) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.InternalOperationDesc.String()),
		Name:        internalOperationName,
		Locations:   []ast.DirectiveLocation{ast.LocationQuery, ast.LocationMutation, ast.LocationSubscription},
	}
}

func (o *internalOperation) Definitions() ast.DefinitionList {
	return nil
}

func (o *internalOperation) Resolve(resolver *OperationResolver) error {
	resolver.Operation.Internal = true
	return nil
}

func init() {
	registerDirective(internalOperationName, &internalOperation{})
}
