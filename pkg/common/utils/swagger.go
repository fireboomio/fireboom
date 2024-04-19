package utils

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"net/http"
	"strconv"
)

var (
	StatusOkText         = http.StatusText(http.StatusOK)
	StatusBadRequestText = http.StatusText(http.StatusBadRequest)
)

// MakeApiOperationRequestBody 构建OperationType_MUTATION的入参定义
func MakeApiOperationRequestBody(requestSchemaRef *openapi3.SchemaRef, multipartForms ...*wgpb.OperationMultipartForm) *openapi3.RequestBodyRef {
	var content openapi3.Content
	if len(multipartForms) > 0 {
		content = openapi3.NewContentWithFormDataSchemaRef(requestSchemaRef)
	} else {
		content = openapi3.NewContentWithJSONSchemaRef(requestSchemaRef)
	}
	return &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{Content: content},
	}
}

// MakeApiOperationResponse 构建返回参数的定义
func MakeApiOperationResponse(responseSchema *openapi3.SchemaRef) openapi3.Responses {
	return openapi3.Responses{
		strconv.Itoa(http.StatusOK): {
			Value: &openapi3.Response{
				Description: &StatusOkText,
				Content:     openapi3.NewContentWithJSONSchemaRef(responseSchema),
			},
		},
		strconv.Itoa(http.StatusBadRequest): {
			Value: &openapi3.Response{
				Description: &StatusBadRequestText,
			},
		},
	}
}
