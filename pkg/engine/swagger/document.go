// Package swagger
/*
 生成引擎注册的接口，包括operation，认证和上传
*/
package swagger

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	build.AddAsyncGenerate(func() build.AsyncGenerate { return &document{} })
}

type document struct {
	doc *openapi3.T
	api *wgpb.UserDefinedApi
}

func (s *document) Generate(builder *build.Builder) {
	s.api = builder.DefinedApi
	s.doc = &openapi3.T{
		OpenAPI:  "3.0.1",
		Security: make(openapi3.SecurityRequirements, 0),
		Paths:    make(openapi3.Paths),
	}

	s.buildInfo()
	s.buildServers()
	s.buildSecurityRequirements()
	s.buildComponents()

	s.buildApiAuthentication()
	s.buildApiOperation()
	s.buildApiUpload()

	var err error
	defer func() {
		if err != nil {
			logger.Error("generate swagger3 failed", zap.Error(err))
		} else {
			logger.Debug("generate swagger3 succeed")
		}
	}()
	docBytes, err := json.Marshal(&s.doc)
	if err != nil {
		return
	}

	s.api = nil
	s.doc = nil
	err = build.GeneratedSwaggerText.Write(build.GeneratedSwaggerText.Title, fileloader.SystemUser, docBytes)
	return
}

func (s *document) buildInfo() {
	s.doc.Info = &openapi3.Info{
		Title:       "Fireboom swagger3.0",
		Description: "Fireboom swagger3.0",
		Contact:     &openapi3.Contact{URL: configs.ApplicationData.ContactAddress},
		Version:     utils.GetStringWithLockViper(consts.FbVersion),
	}
}

func (s *document) buildServers() {
	s.doc.Servers = openapi3.Servers{{URL: utils.GetVariableString(s.api.NodeOptions.PublicNodeUrl)}}
}

func (s *document) buildComponents() {
	s.doc.Components = &openapi3.Components{
		SecuritySchemes: openapi3.SecuritySchemes{
			"JWT": &openapi3.SecuritySchemeRef{Value: openapi3.NewJWTSecurityScheme()},
		},
		Schemas: build.GetOperationsDefinitions(),
	}
}

func (s *document) buildSecurityRequirements() {
	if s.api.AuthenticationConfig == nil {
		return
	}

	var roles []string
	for _, item := range models.RoleRoot.List() {
		roles = append(roles, item.Code)
	}

	for _, provider := range s.api.AuthenticationConfig.CookieBased.Providers {
		s.doc.Security.With(openapi3.NewSecurityRequirement().Authenticate(provider.Id, roles...))
	}
	s.doc.Security.With(openapi3.NewSecurityRequirement().Authenticate("JWT", roles...))
	return
}

func init() {
	utils.RegisterInitMethod(30, func() { logger = zap.L() })
}
