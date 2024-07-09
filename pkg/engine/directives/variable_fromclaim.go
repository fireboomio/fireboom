// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义AuthorizationConfig.Claims，留作后续发送graphql前填充用户信息参数
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
)

const (
	fromClaimName                  = "fromClaim"
	fromClaimArgNameType           = "Claim"
	fromClaimArgCustomJsonPathName = "customJsonPath"

	customClaimNotSupportedType = "customClaim [%s] not support inject type %s"
)

type fromClaim struct{}

func (v *fromClaim) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.FromClaimDesc.String()),
		Name:        fromClaimName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments: ast.ArgumentDefinitionList{
			{
				Description:  i18n.FromClaimArgNameDesc.String(),
				Name:         commonArgName,
				Type:         ast.NonNullNamedType(fromClaimArgNameType, nil),
				DefaultValue: &ast.Value{Kind: ast.EnumValue, Raw: wgpb.ClaimType_USERID.String()},
			},
			{
				Description: i18n.FromClaimArgCustomJsonPathDesc.String(),
				Name:        fromClaimArgCustomJsonPathName,
				Type:        ast.ListType(ast.NamedType(consts.ScalarString, nil), nil),
			},
		},
	}
}

func (v *fromClaim) Definitions() ast.DefinitionList {
	var nameEnumValues ast.EnumValueList
	for k, v := range wgpb.ClaimType_value {
		nameEnumValues = append(nameEnumValues, &ast.EnumValueDefinition{
			Name:        k,
			Description: claimEnumValueDescriptionMap[wgpb.ClaimType(v)],
		})
	}

	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       fromClaimArgNameType,
		EnumValues: nameEnumValues,
	}}
}

func (v *fromClaim) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	value, ok := resolver.Arguments[commonArgName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, commonArgName)
		return
	}

	claimType, ok := wgpb.ClaimType_value[value]
	if !ok {
		err = fmt.Errorf(argumentValueNotSupportedFormat, value, commonArgName)
		return
	}

	skip = true
	claimConfig := &wgpb.ClaimConfig{ClaimType: wgpb.ClaimType(claimType), VariablePathComponents: resolver.Path}
	resolver.Operation.AuthorizationConfig.Claims = append(resolver.Operation.AuthorizationConfig.Claims, claimConfig)
	resolver.Operation.AuthenticationConfig = &wgpb.OperationAuthenticationConfig{AuthRequired: true}
	if claimConfig.ClaimType != wgpb.ClaimType_CUSTOM {
		return
	}

	jsonPath, ok := resolver.Arguments[fromClaimArgCustomJsonPathName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, fromClaimArgCustomJsonPathName)
		return
	}

	var customClaim wgpb.CustomClaim
	if err = json.Unmarshal([]byte(jsonPath), &customClaim.JsonPathComponents); err != nil {
		return
	}

	customName, schemaValueType := resolver.Path[0], resolver.Schema.Value.Type
	customType, ok := customTypeMap[schemaValueType]
	if !ok {
		err = fmt.Errorf(customClaimNotSupportedType, customName, schemaValueType)
		return
	}

	customClaim.Required = slices.Contains(resolver.Schema.Value.Required, customName)
	customClaim.Name = customName
	customClaim.Type = customType
	claimConfig.Custom = &customClaim
	return
}

var (
	claimEnumValueDescriptionMap map[wgpb.ClaimType]string
	customTypeMap                map[string]wgpb.ValueType
)

func init() {
	registerDirective(fromClaimName, &fromClaim{})

	customTypeMap = make(map[string]wgpb.ValueType)
	customTypeMap[openapi3.TypeInteger] = wgpb.ValueType_INT
	customTypeMap[openapi3.TypeNumber] = wgpb.ValueType_FLOAT
	customTypeMap[openapi3.TypeArray] = wgpb.ValueType_ARRAY
	customTypeMap[openapi3.TypeString] = wgpb.ValueType_STRING
	customTypeMap[openapi3.TypeBoolean] = wgpb.ValueType_BOOLEAN

	claimEnumValueDescriptionMap = make(map[wgpb.ClaimType]string)
	claimEnumValueDescriptionMap[wgpb.ClaimType_ROLES] = `ROLES is string array, Please use in [in, notIn].`
}
