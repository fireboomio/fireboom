// Package swagger
/*
 添加上传接口文档
*/
package swagger

import (
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/s3uploadclient"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"net/http"
	"strconv"
)

func (s *document) buildApiUpload() {
	for _, upload := range s.api.S3UploadConfiguration {
		uri := fmt.Sprintf("/s3/%s/upload", upload.Name)
		operation := &openapi3.Operation{
			Tags:        []string{"FileUpload"},
			Summary:     fmt.Sprintf("upload file to %s", upload.Name),
			OperationID: uri,
			Responses:   s.makeApiUploadResponse(),
			Security:    openapi3.NewSecurityRequirements(),
		}

		operation.Parameters = s.makeApiUploadParameters(upload)
		operation.RequestBody = s.makeApiUploadRequestBody()
		s.doc.Paths[uri] = &openapi3.PathItem{
			Post: operation,
		}
	}
}

// 构建上传的普通参数定义
func (s *document) makeApiUploadParameters(upload *wgpb.S3UploadConfiguration) (result openapi3.Parameters) {
	directoryParam := openapi3.NewQueryParameter("directory")
	directoryParam.WithSchema(&openapi3.Schema{Type: openapi3.TypeString, Description: "上传文件目录"})
	result = openapi3.Parameters{{Value: directoryParam}}
	if len(upload.UploadProfiles) == 0 {
		return
	}

	var anyOfSchemas openapi3.SchemaRefs
	var profiles []interface{}
	for name, item := range upload.UploadProfiles {
		// 添加枚举项X-Upload-Profile
		profiles = append(profiles, name)
		if item.MetadataJSONSchema == "" {
			continue
		}

		// 当profile中填写了MetadataJSONSchema将添加到X-Metadata头部参数中
		var itemMeta *openapi3.SchemaRef
		if err := json.Unmarshal([]byte(item.MetadataJSONSchema), &itemMeta); err != nil {
			continue
		}

		anyOfSchemas = append(anyOfSchemas, itemMeta)
	}
	param := openapi3.NewHeaderParameter(s3uploadclient.HeaderUploadProfile)
	param.WithSchema(&openapi3.Schema{Type: openapi3.TypeString, Enum: profiles})
	result = append(result, &openapi3.ParameterRef{Value: param})
	if len(anyOfSchemas) > 0 {
		metaParam := openapi3.NewHeaderParameter(s3uploadclient.HeaderMetadata)
		metaParam.WithSchema(&openapi3.Schema{Type: openapi3.TypeObject, AnyOf: anyOfSchemas})
		result = append(result, &openapi3.ParameterRef{Value: metaParam})
	}
	return
}

// 构建上传的文件参数定义
func (s *document) makeApiUploadRequestBody() (result *openapi3.RequestBodyRef) {
	formDataContent := openapi3.NewContentWithFormDataSchema(&openapi3.Schema{
		Type:     openapi3.TypeObject,
		Required: []string{"file"},
		Properties: openapi3.Schemas{
			"file": openapi3.NewSchemaRef("", &openapi3.Schema{
				Type: openapi3.TypeArray,
				Items: openapi3.NewSchemaRef("", &openapi3.Schema{
					Type: openapi3.TypeString, Format: "binary",
				}),
			}),
		},
	})
	return &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{Content: formDataContent},
	}
}

// 构建上传返回参数的定义
func (s *document) makeApiUploadResponse() openapi3.Responses {
	uploadedFilesSchemaRefName := utils.ReflectStructToOpenapi3Schema(s3uploadclient.UploadedFiles{}, s.doc.Components.Schemas)
	uploadedFilesSchema := s.doc.Components.Schemas[uploadedFilesSchemaRefName]
	delete(s.doc.Components.Schemas, uploadedFilesSchemaRefName)
	return openapi3.Responses{
		strconv.Itoa(http.StatusOK): {
			Value: &openapi3.Response{
				Description: &utils.StatusOkText,
				Content:     openapi3.NewContentWithJSONSchemaRef(uploadedFilesSchema),
			},
		},
		strconv.Itoa(http.StatusBadRequest): {
			Value: &openapi3.Response{Description: &utils.StatusBadRequestText},
		},
	}
}
