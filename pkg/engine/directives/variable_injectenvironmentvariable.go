// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义VariablesConfiguration.InjectVariables，留作后续发送graphql前填充环境变量参数
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

const injectEnvironmentVariableName = "injectEnvironmentVariable"

type injectEnvironmentVariable struct{}

func (v *injectEnvironmentVariable) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.InjectEnvironmentVariableDesc.String()),
		Name:        injectEnvironmentVariableName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments: ast.ArgumentDefinitionList{{
			Name: commonArgName,
			Type: ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (v *injectEnvironmentVariable) Definitions() ast.DefinitionList {
	return nil
}

func (v *injectEnvironmentVariable) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	value, ok := resolver.Arguments[commonArgName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, commonArgName)
		return
	}

	resolver.Operation.VariablesConfiguration.InjectVariables = append(resolver.Operation.VariablesConfiguration.InjectVariables, &wgpb.VariableInjectionConfiguration{
		VariablePathComponents:  resolver.Path,
		VariableKind:            wgpb.InjectVariableKind_ENVIRONMENT_VARIABLE,
		ValueTypeName:           getRealVariableTypeName(resolver.Schema),
		EnvironmentVariableName: value,
	})
	skip = true
	return
}

func init() {
	registerDirective(injectEnvironmentVariableName, &injectEnvironmentVariable{})
}
