// Package swagger
/*
 添加身份认证接口文档
*/
package swagger

import (
	"fireboom-server/pkg/common/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/wundergraph/wundergraph/pkg/authentication"
)

func (s *document) buildApiAuthentication() {
	userSchemaRefName := utils.ReflectStructToOpenapi3Schema(authentication.User{}, s.doc.Components.Schemas)
	userSchema := s.doc.Components.Schemas[userSchemaRefName]
	delete(s.doc.Components.Schemas, userSchemaRefName)
	userTags := []string{"Platform-User"}
	userInfoUri := "/auth/cookie/user"
	s.doc.Paths[userInfoUri] = &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:        userTags,
			Summary:     "获取用户信息",
			OperationID: userInfoUri,
			Security:    &s.doc.Security,
			Responses:   utils.MakeApiOperationResponse(userSchema),
		},
	}

	logoutParam := openapi3.NewQueryParameter("logout_openid_connect_provider")
	var paramEnum []interface{}
	paramEnum = append(paramEnum, "true", "false")
	logoutParam.WithSchema(&openapi3.Schema{Type: openapi3.TypeString, Enum: paramEnum})
	userLogoutUri := "/auth/cookie/user/logout"
	s.doc.Paths[userLogoutUri] = &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:        userTags,
			Summary:     "用户登出",
			OperationID: userLogoutUri,
			Security:    &s.doc.Security,
			Parameters:  openapi3.Parameters{{Value: logoutParam}},
			Responses:   openapi3.Responses{},
		},
	}
}
