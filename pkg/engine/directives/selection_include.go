// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)

const (
	includeName                   = "include"
	argIfName                     = "if"
	argIfRuleName                 = "ifRule"
	VariablePrefix                = "$"
	argumentMustSupplyOneOfFormat = `must supply one of arguments [%s]`
)

var includeDirective = &include{}

type include struct{}

func (i *include) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Name:      includeName,
		Locations: []ast.DirectiveLocation{ast.LocationField, ast.LocationFragmentSpread, ast.LocationInlineFragment},
		Arguments: ast.ArgumentDefinitionList{{
			Name: argIfName,
			Type: ast.NamedType(consts.ScalarBoolean, nil),
		}, {
			Name: argIfRuleName,
			Type: ast.NamedType(consts.ScalarString, nil),
		}},
	}
}

func (i *include) Definitions() ast.DefinitionList {
	return nil
}

func (i *include) Resolve(resolver *SelectionResolver) (err error) {
	argIfValue, argIfFound := resolver.Arguments[argIfName]
	argIfRuleValue, argIfRuleFound := resolver.Arguments[argIfRuleName]
	if !argIfFound && !argIfRuleFound {
		err = fmt.Errorf(argumentMustSupplyOneOfFormat, utils.JoinString(",", argIfName, argIfRuleName))
		return
	}

	if argIfFound {
		if err = i.addVariablesSchema(resolver, argIfValue, boolSchema); err != nil {
			return
		}
	}
	if argIfRuleFound {
		if err = i.addVariablesSchema(resolver, argIfRuleValue, stringSchema); err != nil {
			return
		}
		resolver.Operation.RuleExpressionExisted = true
	}
	return
}

var (
	stringSchema = openapi3.NewStringSchema()
	boolSchema   = openapi3.NewBoolSchema()
)

func (i *include) addVariablesSchema(resolver *SelectionResolver, argValue string, argSchema *openapi3.Schema) (err error) {
	argValue, ok := strings.CutPrefix(argValue, VariablePrefix)
	if !ok {
		return
	}
	if resolver.OperationDefinition.VariableDefinitions.ForName(argValue) == nil {
		err = fmt.Errorf("variable [%s] not found", argValue)
		return
	}
	resolver.VariableSchemas[argValue] = &openapi3.SchemaRef{Value: argSchema}
	return
}

func init() {
	registerDirective(includeName, &include{})
}
