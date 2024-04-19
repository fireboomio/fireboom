// Package server
/*
 结合swag和openapi2，利用反射动态生成swagger文档
 其中元数据定义在base.AddRouterMetas中注册(运行时自动注册)
 支持定义$modelName$动态替换路由地址
 支持定义#/definitions/$modelName$动态替换schema及范型类型
*/
package server

import (
	"fireboom-server/pkg/api/base"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/swaggo/swag"
	"go.uber.org/zap"
	"net/http"
	"path"
	"strings"
)

const (
	cloudInstanceName       = "swagger-boom"
	openapi2SchemaRefPrefix = "#/definitions/"
)

var setOperationPathItemFuncMap map[string]func(item *openapi2.PathItem, operation *openapi2.Operation)

func init() {
	setOperationPathItemFuncMap = make(map[string]func(item *openapi2.PathItem, operation *openapi2.Operation))
	setOperationPathItemFuncMap[http.MethodPost] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Post = operation }
	setOperationPathItemFuncMap[http.MethodDelete] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Delete = operation }
	setOperationPathItemFuncMap[http.MethodPut] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Put = operation }
	setOperationPathItemFuncMap[http.MethodGet] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Get = operation }

	setOperationPathItemFuncMap[http.MethodHead] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Head = operation }
	setOperationPathItemFuncMap[http.MethodPatch] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Patch = operation }
	setOperationPathItemFuncMap[http.MethodOptions] = func(item *openapi2.PathItem, operation *openapi2.Operation) { item.Options = operation }
}

type cloudSwagger struct {
	doc *openapi2.T
}

func rewriteDynamicSwagger() {
	swagger, err := swag.ReadDoc()
	if err != nil {
		logger.Error("read swagger failed")
		return
	}

	var doc *openapi2.T
	if err = json.Unmarshal([]byte(swagger), &doc); err != nil {
		logger.Error("unmarshal swagger failed", zap.Error(err))
		return
	}

	cloudSwag := &cloudSwagger{doc: doc}
	for pathName, pathItem := range doc.Paths {
		cloudSwag.rewritePathItem(pathName, http.MethodPost, pathItem.Post)
		cloudSwag.rewritePathItem(pathName, http.MethodDelete, pathItem.Delete)
		cloudSwag.rewritePathItem(pathName, http.MethodPut, pathItem.Put)
		cloudSwag.rewritePathItem(pathName, http.MethodGet, pathItem.Get)

		cloudSwag.rewritePathItem(pathName, http.MethodHead, pathItem.Head)
		cloudSwag.rewritePathItem(pathName, http.MethodPatch, pathItem.Patch)
		cloudSwag.rewritePathItem(pathName, http.MethodOptions, pathItem.Options)
	}

	doc.Info = openapi3.Info{Title: cloudInstanceName, Version: utils.GetStringWithLockViper(consts.FbVersion)}
	doc.BasePath = configs.ApplicationData.ContextPath
	docBytes, err := json.Marshal(doc)
	if err != nil {
		logger.Error("marshal swagger failed", zap.Error(err))
		return
	}

	swag.Register(cloudInstanceName, &swag.Spec{
		InfoInstanceName: cloudInstanceName,
		SwaggerTemplate:  string(docBytes),
	})
}

func (c *cloudSwagger) rewritePathItem(name, method string, operation *openapi2.Operation) {
	if operation == nil || !strings.Contains(name, base.ModelNameFlag) {
		return
	}

	delete(c.doc.Paths, name)
	routerMetas := base.RouterMetasMap[path.Join(method, name)]
	for _, itemMeta := range routerMetas {
		definitionName := utils.ReflectStructToOpenapi3Schema(itemMeta.Data, c.doc.Definitions, openapi2SchemaRefPrefix)
		copyOperation := &openapi2.Operation{
			Tags:        []string{itemMeta.Name},
			Summary:     operation.Summary,
			Description: operation.Description,
			Responses:   make(map[string]*openapi2.Response),
		}
		for _, itemParam := range operation.Parameters {
			c.copyParameter(copyOperation, itemParam, definitionName)
		}

		for status, itemResponse := range operation.Responses {
			c.copyResponse(copyOperation, itemResponse, definitionName, status)
		}

		itemPath := strings.ReplaceAll(name, base.ModelNameFlag, itemMeta.Name)
		var copyPathItem *openapi2.PathItem
		if exist, ok := c.doc.Paths[itemPath]; ok {
			copyPathItem = exist
		} else {
			copyPathItem = &openapi2.PathItem{}
			c.doc.Paths[itemPath] = copyPathItem
		}
		setOperationPathItemFuncMap[method](copyPathItem, copyOperation)
	}
}

func (c *cloudSwagger) copyParameter(copyOperation *openapi2.Operation, itemParam *openapi2.Parameter, definitionName string) {
	copyParam := *itemParam
	defer func() { copyOperation.Parameters = append(copyOperation.Parameters, &copyParam) }()

	if !strings.Contains(itemParam.Description, base.ModelNameFlag) {
		return
	}

	definitionRef := &openapi3.SchemaRef{Ref: strings.ReplaceAll(copyParam.Description, base.ModelNameFlag, definitionName)}
	copyParam.Description = ""

	if items := copyParam.Schema.Value.Items; items != nil {
		arraySchema := openapi3.NewArraySchema()
		arraySchema.Items = definitionRef
		copyParam.Schema = &openapi3.SchemaRef{Value: arraySchema}
		return
	}

	copyParam.Schema = definitionRef
}

func (c *cloudSwagger) copyResponse(copyOperation *openapi2.Operation, itemResponse *openapi2.Response, definitionName, status string) {
	copyResponse := *itemResponse
	defer func() { copyOperation.Responses[status] = &copyResponse }()

	if !strings.Contains(copyResponse.Description, base.ModelNameFlag) {
		return
	}

	definitionRef := &openapi3.SchemaRef{Ref: strings.ReplaceAll(copyResponse.Description, base.ModelNameFlag, definitionName)}
	copyResponse.Description = http.StatusText(cast.ToInt(status))
	if itemResponse.Schema.Ref != "" {
		copyResponse.Schema = c.rewriteGenericsDefinition(definitionName, itemResponse.Schema.Ref, definitionRef)
		return
	}

	if items := copyResponse.Schema.Value.Items; items != nil {
		arraySchema := openapi3.NewArraySchema()
		arraySchema.Items = definitionRef
		copyResponse.Schema = &openapi3.SchemaRef{Value: arraySchema}
		return
	}

	copyResponse.Schema = definitionRef
}

func (c *cloudSwagger) rewriteGenericsDefinition(modelName, ref string, refSchema *openapi3.SchemaRef) *openapi3.SchemaRef {
	normalName := strings.TrimPrefix(ref, openapi2SchemaRefPrefix)
	normalSchema := c.doc.Definitions[normalName]
	before, after, ok := strings.Cut(normalName, "_")
	if !ok {
		return nil
	}

	genericsSchema := &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	for name, item := range normalSchema.Value.Properties {
		if after == name {
			item = refSchema
		}

		genericsSchema.Value.Properties[name] = item
	}

	genericsName := utils.JoinStringWithDot(before, modelName)
	c.doc.Definitions[genericsName] = genericsSchema
	return &openapi3.SchemaRef{Ref: openapi2SchemaRefPrefix + genericsName}
}
