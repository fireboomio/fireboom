// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义VariablesConfiguration.InjectVariables，留作后续发送graphql前填充UUID参数
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

const injectGeneratedUUIDName = "injectGeneratedUUID"

type injectGeneratedUUID struct{}

func (v *injectGeneratedUUID) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.InjectGeneratedUUIDDesc.String()),
		Name:        injectGeneratedUUIDName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
	}
}

func (v *injectGeneratedUUID) Definitions() ast.DefinitionList {
	return nil
}

func (v *injectGeneratedUUID) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	resolver.Operation.VariablesConfiguration.InjectVariables = append(resolver.Operation.VariablesConfiguration.InjectVariables, &wgpb.VariableInjectionConfiguration{
		VariablePathComponents: resolver.Path,
		VariableKind:           wgpb.InjectVariableKind_UUID,
		ValueTypeName:          resolver.Schema.Value.Type,
	})
	skip = true
	return
}

func init() {
	registerDirective(injectGeneratedUUIDName, &injectGeneratedUUID{})
}
