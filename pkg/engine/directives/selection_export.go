// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve判断导出的参数是否在传递参数中定义
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/exp/slices"
)

const (
	exportName                    = "export"
	exportArgName                 = "as"
	exportDirectiveRequiredFormat = `expected variable [%s] with directive @internal for @export(as: "%s") on path [%s]`
)

type export struct{}

func (e *export) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.ExportDesc.String()),
		Name:        exportName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{{
			Name: exportArgName,
			Type: ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (e *export) Definitions() ast.DefinitionList {
	return nil
}

func (e *export) Resolve(resolver *SelectionResolver) error {
	value, ok := resolver.Arguments[exportArgName]
	if !ok {
		return fmt.Errorf(argumentRequiredFormat, exportArgName)
	}

	if !slices.ContainsFunc(resolver.OperationDefinition.VariableDefinitions, func(variableDefinition *ast.VariableDefinition) bool {
		internalDirective := variableDefinition.Directives.ForName(internalName)
		return internalDirective != nil && variableDefinition.Variable == value
	}) {
		return fmt.Errorf(exportDirectiveRequiredFormat, value, value, utils.JoinStringWithDot(resolver.Path...))
	}
	resolver.VariableExported[value] = true
	return nil
}

func init() {
	registerDirective(exportName, &export{})
}
