// Package sdk
/*
 钩子swagger文档，可以根据此输出自定义开发sdk服务
*/
package sdk

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func (o *reflectObjectFactory) generateHookSwagger() {
	doc := &openapi3.T{
		OpenAPI:  "3.0.1",
		Security: make(openapi3.SecurityRequirements, 0),
		Paths:    make(openapi3.Paths),
		Info: &openapi3.Info{
			Title:   "Fireboom Hook swagger3.0",
			Contact: &openapi3.Contact{URL: configs.ApplicationData.ContactAddress},
			Version: utils.GetStringWithLockViper(consts.FbVersion),
		},
		Components: &openapi3.Components{Schemas: o.definitions},
	}

	endpointKeys := maps.Keys(o.endpoints)
	slices.Sort(endpointKeys)
	for _, endpoint := range endpointKeys {
		metadata := o.endpoints[endpoint]
		var (
			requestBodySchema, responseSchema *openapi3.SchemaRef
			requestBodyOk, responseOk         bool
		)
		if metadata.requestBody != nil {
			requestBodyName := utils.GetTypeName(metadata.requestBody)
			if _, requestBodyOk = o.definitions[requestBodyName]; requestBodyOk {
				requestBodySchema = &openapi3.SchemaRef{Ref: interpolate.Openapi3SchemaRefPrefix + requestBodyName}
			}
		}
		if metadata.response != nil {
			responseName := utils.GetTypeName(metadata.response)
			if responseSchema, responseOk = o.definitions[responseName]; responseOk {
				if metadata.middlewareResponseRewrite != nil {
					rewriteName := utils.GetTypeName(metadata.middlewareResponseRewrite)
					copySchema := *responseSchema
					copySchema.Value.Properties["response"] = o.definitions[rewriteName]
					responseSchema = &copySchema
				} else {
					responseSchema = &openapi3.SchemaRef{Ref: interpolate.Openapi3SchemaRefPrefix + responseName}
				}
			}
		}
		pathItem := &openapi3.Operation{
			OperationID: string(metadata.title),
			Summary:     endpoint,
			Security:    openapi3.NewSecurityRequirements(),
		}
		if requestBodyOk {
			pathItem.RequestBody = utils.MakeApiOperationRequestBody(requestBodySchema)
		}
		if responseOk {
			pathItem.Responses = utils.MakeApiOperationResponse(responseSchema)
		}
		doc.Paths[endpoint] = &openapi3.PathItem{Post: pathItem}
	}

	docBytes, _ := json.Marshal(&doc)
	_ = build.GeneratedHookSwaggerText.Write(build.GeneratedHookSwaggerText.Title, fileloader.SystemUser, docBytes)
}
