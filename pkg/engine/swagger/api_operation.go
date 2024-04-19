// Package swagger
/*
 添加operation接口文档
*/
package swagger

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/wundergraph/wundergraph/pkg/apihandler"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"strings"
)

func (s *document) buildApiOperation() {
	operationsConfigData := build.GeneratedOperationsConfigRoot.FirstData()
	for _, item := range s.api.Operations {
		if item.Internal {
			continue
		}

		apiData, err := models.OperationRoot.GetByDataName(item.Path)
		if err != nil {
			continue
		}

		uri := apihandler.OperationApiPath(item.Path)
		operation := &openapi3.Operation{
			OperationID: uri,
			Summary:     item.Path,
			Tags:        s.makeApiOperationTags(item.Path),
			Security:    openapi3.NewSecurityRequirements(),
		}

		var requestSchema, responseSchema *openapi3.SchemaRef
		switch item.Engine {
		case wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
			graphqlFile, ok := operationsConfigData.GraphqlOperationFiles[item.Path]
			if !ok {
				return
			}

			requestSchema, responseSchema = graphqlFile.Variables, graphqlFile.Response
			originContent, _ := models.OperationGraphql.Read(item.Path)
			operation.Description = fmt.Sprintf("```graphql\n%s\n```", originContent)
			if remark := apiData.Remark; remark != "" {
				operation.Description = utils.JoinString("\n", remark, operation.Description)
			}
		case wgpb.OperationExecutionEngine_ENGINE_FUNCTION:
			functionFile, ok := operationsConfigData.FunctionOperationFiles[item.Path]
			if !ok {
				return
			}

			requestSchema, responseSchema = functionFile.Variables, functionFile.Response
		}
		operation.Responses = utils.MakeApiOperationResponse(responseSchema)
		if title := apiData.Title; title != "" {
			operation.Summary = title
		}

		if item.AuthenticationConfig != nil && item.AuthenticationConfig.AuthRequired {
			operation.Security = &s.doc.Security
		}
		pathItem := &openapi3.PathItem{}
		if item.OperationType == wgpb.OperationType_MUTATION {
			operation.RequestBody = utils.MakeApiOperationRequestBody(requestSchema, item.MultipartForms...)
			pathItem.Post = operation
		} else {
			operation.Parameters = s.makeApiOperationParameters(requestSchema)
			pathItem.Get = operation
		}
		s.doc.Paths[operation.OperationID] = pathItem
	}
	return
}

func (s *document) makeApiOperationTags(path string) []string {
	tag := "Others"
	if before, _, ok := strings.Cut(path, "/"); ok {
		tag = before
	}
	return []string{tag}
}

// 构建OperationType_QUERY的入参定义
func (s *document) makeApiOperationParameters(requestSchemaRef *openapi3.SchemaRef) (result openapi3.Parameters) {
	if nil == requestSchemaRef || requestSchemaRef.Value == nil {
		return
	}

	result = make(openapi3.Parameters, 0)
	for name, schemaRef := range requestSchemaRef.Value.Properties {
		param := openapi3.NewQueryParameter(name)
		param.Schema = schemaRef
		param.Required = slices.Contains(requestSchemaRef.Value.Required, name)
		result = append(result, &openapi3.ParameterRef{Value: param})
	}
	return
}
