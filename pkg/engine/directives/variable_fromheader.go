// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义VariablesConfiguration.InjectVariables，留作后续发送graphql前填充请求头中参数
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

const fromHeaderName = "fromHeader"

type fromHeader struct{}

func (v *fromHeader) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.FromHeaderDesc.String()),
		Name:        fromHeaderName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments: ast.ArgumentDefinitionList{{
			Name: commonArgName,
			Type: ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (v *fromHeader) Definitions() ast.DefinitionList {
	return nil
}

func (v *fromHeader) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	value, ok := resolver.Arguments[commonArgName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, commonArgName)
		return
	}

	resolver.Operation.VariablesConfiguration.InjectVariables = append(resolver.Operation.VariablesConfiguration.InjectVariables, &wgpb.VariableInjectionConfiguration{
		VariablePathComponents: resolver.Path,
		FromHeaderName:         value,
		VariableKind:           wgpb.InjectVariableKind_FROM_HEADER,
	})
	skip = true
	return
}

func init() {
	registerDirective(fromHeaderName, &fromHeader{})
}
