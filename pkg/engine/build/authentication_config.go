// Package build
/*
 读取store/authentication配置并转换成引擎所需的配置
 判断认证钩子的存在、开启、路径等属性合成认证钩子的开关
*/
package build

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
)

func init() {
	utils.RegisterInitMethod(30, func() {
		addResolve(3, func() Resolve { return &authenticationConfig{modelName: models.AuthenticationRoot.GetModelName()} })
	})
}

type authenticationConfig struct {
	modelName string
}

func (a *authenticationConfig) Resolve(builder *Builder) (err error) {
	globalOperation := models.GlobalOperationRoot.FirstData()
	if globalOperation == nil {
		return
	}

	securityConfig := configs.GlobalSettingRoot.FirstData().SecurityConfig
	jwksBased := &wgpb.JwksBasedAuthentication{}
	cookieBased := &wgpb.CookieBasedAuthentication{
		AuthorizedRedirectUris:       securityConfig.AuthorizedRedirectUris,
		AuthorizedRedirectUriRegexes: securityConfig.AuthorizedRedirectUriRegexes,
		CsrfSecret:                   utils.MakeEnvironmentVariable(apihandler.WgEnvCsrfSecret, ""),
		HashKey:                      utils.MakeEnvironmentVariable(apihandler.WgEnvHashKey, ""),
		BlockKey:                     utils.MakeEnvironmentVariable(apihandler.WgEnvBlockKey, ""),
	}

	authentications := models.AuthenticationRoot.ListByCondition(func(item *models.Authentication) bool { return item.Enabled })
	for _, authItem := range authentications {
		var succeed bool
		if authItem.JwksProviderEnabled {
			succeed = true
			jwksBased.Providers = append(jwksBased.Providers, a.makeJwksProvider(authItem))
		}

		if provider := authItem.OidcConfig; authItem.OidcConfigEnabled && provider != nil {
			succeed = true
			cookieBased.Providers = append(cookieBased.Providers, a.makeCookieAuthProvider(authItem))
		}
		if succeed {
			logger.Debug("build authentication succeed", zap.String(a.modelName, authItem.Name))
		}
	}

	authenticationHooks := &wgpb.ApiAuthenticationHooks{}
	for hook, option := range models.GetAuthenticationHookOptions() {
		if !option.Enabled || !option.Existed {
			continue
		}

		switch hook {
		case consts.PostAuthentication:
			authenticationHooks.PostAuthentication = true
		case consts.MutatingPostAuthentication:
			authenticationHooks.MutatingPostAuthentication = true
		case consts.RevalidateAuthentication:
			authenticationHooks.RevalidateAuthentication = true
		case consts.PostLogout:
			authenticationHooks.PostLogout = true
		}
	}
	builder.DefinedApi.AuthenticationConfig = &wgpb.ApiAuthenticationConfig{
		JwksBased:   jwksBased,
		CookieBased: cookieBased,
		Hooks:       authenticationHooks,
	}
	return
}

func (a *authenticationConfig) makeJwksProvider(item *models.Authentication) *wgpb.JwksAuthProvider {
	jwksProvider := &wgpb.JwksAuthProvider{Issuer: item.Issuer}
	if provider := item.JwksProvider; provider != nil {
		jwksProvider.JwksJson = provider.JwksJson
		jwksProvider.UserInfoCacheTtlSeconds = provider.UserInfoCacheTtlSeconds
	}
	return jwksProvider
}

func (a *authenticationConfig) makeCookieAuthProvider(item *models.Authentication) *wgpb.AuthProvider {
	return &wgpb.AuthProvider{
		Id:   item.Name,
		Kind: wgpb.AuthProviderKind_AuthProviderOIDC,
		OidcConfig: &wgpb.OpenIDConnectAuthProviderConfig{
			Issuer:          item.Issuer,
			ClientId:        item.OidcConfig.ClientId,
			ClientSecret:    item.OidcConfig.ClientSecret,
			QueryParameters: item.OidcConfig.QueryParameters,
		},
	}
}
