// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义VariablesConfiguration.InjectVariables，留作后续发送graphql前填充当前时间参数
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/PaesslerAG/gval"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"strings"
)

const (
	injectRuleValueName              = "injectRuleValue"
	injectRuleValueArgExpressionName = "expression"
)

type injectRuleValue struct{}

func (v *injectRuleValue) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.InjectRuleValueDesc.String()),
		Name:        injectRuleValueName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments: ast.ArgumentDefinitionList{{
			Name:         injectRuleValueArgExpressionName,
			Type:         ast.NonNullNamedType(consts.ScalarString, nil),
			DefaultValue: &ast.Value{Kind: ast.StringValue, Raw: "arguments.name + environments.name + headers.name + user.name"},
		}},
	}
}

func (v *injectRuleValue) Definitions() ast.DefinitionList {
	return nil
}

func (v *injectRuleValue) Resolve(resolver *VariableResolver) (unableInput, skip bool, err error) {
	value, ok := resolver.Arguments[injectRuleValueArgExpressionName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, injectRuleValueArgExpressionName)
		return
	}
	value = strings.ReplaceAll(value, "'", "`")
	if _, err = apihandler.GvalFullLanguage.NewEvaluable(value); err != nil {
		return
	}

	injectRuleVariable := &wgpb.VariableInjectionConfiguration{
		VariablePathComponents: resolver.Path,
		VariableKind:           wgpb.InjectVariableKind_RULE_EXPRESSION,
		ValueTypeName:          getRealVariableTypeName(resolver.Schema),
		RuleExpression:         value,
	}
	resolver.Operation.RuleExpressionExisted = true
	resolver.Operation.VariablesConfiguration.InjectVariables = append(resolver.Operation.VariablesConfiguration.InjectVariables, injectRuleVariable)
	unableInput, skip = true, true
	return
}

func init() {
	registerDirective(injectRuleValueName, &injectRuleValue{})
	apihandler.GvalFullLanguage = gval.Full(gval.Ident(), gval.Parentheses(),
		gval.Function("isEmpty", utils.IsZeroValue),
		gval.Function("isAllEmpty", func(args ...any) bool {
			return !slices.ContainsFunc(args, func(v any) bool { return !utils.IsZeroValue(v) })
		}),
		gval.Function("isAnyEmpty", func(args ...any) bool {
			return slices.ContainsFunc(args, utils.IsZeroValue)
		}),
		gval.Function("stringContains", func(a, b string) bool {
			return strings.Contains(a, b)
		}),
		gval.Function("arrayContains", func(a []any, b any) bool {
			return slices.ContainsFunc(a, func(aa any) bool {
				return fmt.Sprintf("%v", aa) == fmt.Sprintf("%v", b)
			})
		}),
	)
	apihandler.GvalIsEmptyValue = utils.IsZeroValue
}
