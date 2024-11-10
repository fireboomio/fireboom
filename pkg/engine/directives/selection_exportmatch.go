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
	exportMatchName    = "exportMatch"
	exportMatchArgName = "for"
)

type exportMatch struct{}

func (s *exportMatch) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.ExportMatchDesc.String())),
		Name:        exportMatchName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{{
			Name: exportMatchArgName,
			Type: ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (s *exportMatch) Definitions() ast.DefinitionList {
	return nil
}

func (s *exportMatch) Resolve(resolver *SelectionResolver) error {
	value, ok := resolver.Arguments[exportMatchArgName]
	if !ok {
		return fmt.Errorf(argumentRequiredFormat, exportMatchArgName)
	}

	if !slices.ContainsFunc(resolver.OperationDefinition.VariableDefinitions, func(variableDefinition *ast.VariableDefinition) bool {
		internalDirective := variableDefinition.Directives.ForName(internalName)
		return internalDirective != nil && variableDefinition.Variable == value
	}) {
		return fmt.Errorf(exportDirectiveRequiredFormat, value, value, utils.JoinStringWithDot(resolver.Path...))
	}
	return nil
}

func init() {
	registerDirective(exportMatchName, &exportMatch{})
}
