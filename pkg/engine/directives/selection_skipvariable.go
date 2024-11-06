// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	skipVariableName    = "skipVariable"
	skipVariableArgName = "variables"
)

type skipVariable struct{}

func (s *skipVariable) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		IsRepeatable: true,
		Description:  prependMockWorked(appendIfExistExampleGraphql(i18n.SkipVariableDesc.String())),
		Name:         skipVariableName,
		Locations:    []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{{
			Name: skipVariableArgName,
			Type: ast.NonNullListType(&ast.Type{NamedType: consts.ScalarString}, nil),
		}, {
			Name: argIfRuleName,
			Type: ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (s *skipVariable) Definitions() ast.DefinitionList {
	return nil
}

func (s *skipVariable) Resolve(resolver *SelectionResolver) (err error) {
	return includeDirective.Resolve(resolver)
}

func init() {
	registerDirective(skipVariableName, &skipVariable{})
}
