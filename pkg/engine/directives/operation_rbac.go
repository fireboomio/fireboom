// Package directives
/*
 实现OperationDirective接口，只能定义在LocationQuery, LocationMutation, LocationSubscription上
 Resolve 按照引擎配置定义AuthorizationConfig.RoleConfig，引擎层在调用时会判断用户角色权限
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/plugins/i18n"
	json "github.com/json-iterator/go"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

const (
	RbacName         = "rbac"
	rbacRoleEnumName = "WG_ROLE"
)

type (
	rbac         struct{}
	rbacArgument struct {
		description      string
		updateRoleConfig func(*wgpb.OperationRoleConfig, []string)
	}
)

func (o *rbac) Directive() *ast.DirectiveDefinition {
	var arguments ast.ArgumentDefinitionList
	for k, v := range rbacArgMap {
		arguments = append(arguments, &ast.ArgumentDefinition{
			Name:        k,
			Description: v.description,
			Type:        ast.ListType(ast.NamedType(rbacRoleEnumName, nil), nil),
		})
	}
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.RbacDesc.String()),
		Name:        RbacName,
		Locations:   []ast.DirectiveLocation{ast.LocationQuery, ast.LocationMutation, ast.LocationSubscription},
		Arguments:   arguments,
	}
}

func (o *rbac) Definitions() ast.DefinitionList {
	var enumValues ast.EnumValueList
	for _, item := range models.RoleRoot.List() {
		enumValues = append(enumValues, &ast.EnumValueDefinition{Name: item.Code})
	}
	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       rbacRoleEnumName,
		EnumValues: enumValues,
	}}
}

func (o *rbac) Resolve(resolver *OperationResolver) error {
	roleConfig := &wgpb.OperationRoleConfig{}
	for name, value := range resolver.Arguments {
		updateRoleFunc := FetchUpdateRoleFunc(name)
		if updateRoleFunc == nil {
			continue
		}

		var roles []string
		if err := json.Unmarshal([]byte(value), &roles); err != nil {
			return err
		}

		updateRoleFunc(roleConfig, roles)
	}
	resolver.Operation.AuthorizationConfig.RoleConfig = roleConfig
	resolver.Operation.AuthenticationConfig = &wgpb.OperationAuthenticationConfig{AuthRequired: true}
	return nil
}

func FetchUpdateRoleFunc(name string) func(config *wgpb.OperationRoleConfig, roles []string) {
	arg, ok := rbacArgMap[name]
	if !ok {
		return nil
	}

	return arg.updateRoleConfig
}

var rbacArgMap map[string]*rbacArgument

func init() {
	registerDirective(RbacName, &rbac{})

	rbacArgMap = make(map[string]*rbacArgument)
	rbacArgMap[consts.RequireMatchAll] = &rbacArgument{
		updateRoleConfig: func(config *wgpb.OperationRoleConfig, roles []string) {
			config.RequireMatchAll = roles
		},
		description: i18n.RbacRequireMatchAllDesc.String(),
	}
	rbacArgMap[consts.RequireMatchAny] = &rbacArgument{
		updateRoleConfig: func(config *wgpb.OperationRoleConfig, roles []string) {
			config.RequireMatchAny = roles
		},
		description: i18n.RbacRequireMatchAnyDesc.String(),
	}
	rbacArgMap[consts.DenyMatchAll] = &rbacArgument{
		updateRoleConfig: func(config *wgpb.OperationRoleConfig, roles []string) {
			config.DenyMatchAll = roles
		},
		description: i18n.RbacDenyMatchAllDesc.String(),
	}
	rbacArgMap[consts.DenyMatchAny] = &rbacArgument{
		updateRoleConfig: func(config *wgpb.OperationRoleConfig, roles []string) {
			config.DenyMatchAny = roles
		},
		description: i18n.RbacDenyMatchAnyDesc.String(),
	}
}
