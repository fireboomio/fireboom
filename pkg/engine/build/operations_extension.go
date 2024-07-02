// Package build
/*
 编译proxy/function类型的operation
*/
package build

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

type functionSchemaDefinitions struct {
	Definitions openapi3.Schemas `json:"definitions"`
}

func (o *operations) extractExtensionOperation(operation *models.Operation, extensionText *fileloader.ModelText[models.Operation],
	extensions, builtExtensions map[string]*ExtensionOperationFile) (defRefs []string, extracted bool) {
	modifiedTime, _ := extensionText.GetModifiedTime(operation.Path)
	defer func() { operation.ContentModifiedTime = modifiedTime }()
	if !modifiedTime.Equal(operation.ContentModifiedTime) {
		return
	}
	extension, ok := builtExtensions[operation.Path]
	if !ok {
		return
	}

	extensions[operation.Path] = extension
	defRefs, extracted = extension.VariablesRefs, true
	return
}

func (o *operations) resolveExtensionOperation(operation *models.Operation,
	extensionText *fileloader.ModelText[models.Operation],
	extensions map[string]*ExtensionOperationFile) (operationResult *wgpb.Operation, err error) {
	if !operation.Enabled {
		return
	}

	content, err := extensionText.Read(operation.Path)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(content), &operationResult); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.LoaderFileUnmarshalError, extensionText.GetPath(operation.Path))
		return
	}

	operationResult.Name = normalizeOperationName(operation.Path)
	operationResult.Path = operation.Path
	operationResult.Engine = operation.Engine
	operationResult.RateLimit = operation.RateLimit
	operationResult.Semaphore = operation.Semaphore
	if operationResult.AuthorizationConfig == nil {
		operationResult.AuthorizationConfig = &wgpb.OperationAuthorizationConfig{}
	}

	extensionFile := &ExtensionOperationFile{
		BaseOperationFile: BaseOperationFile{
			OperationName:       operationResult.Name,
			FilePath:            extensionText.GetPath(operationResult.Path),
			OperationType:       operationResult.OperationType,
			AuthorizationConfig: operationResult.AuthorizationConfig,
		},
		ModulePath: extensionText.Root,
	}
	extensionFile.Variables = o.resolveExtensionSchema([]byte(operationResult.VariablesSchema), extensionFile)
	extensionFile.Response = o.resolveExtensionSchema([]byte(operationResult.ResponseSchema), extensionFile)
	operationResult.VariablesSchema = ""
	operationResult.ResponseSchema = ""

	extensions[operation.Path] = extensionFile
	return
}

func (o *operations) resolveExtensionSchema(schemaBytes []byte, extensionFile *ExtensionOperationFile) *openapi3.SchemaRef {
	var schema openapi3.SchemaRef
	if err := json.Unmarshal(schemaBytes, &schema); err != nil || schema.Value == nil {
		return nil
	}

	var additionalDefinitions functionSchemaDefinitions
	_ = json.Unmarshal(schemaBytes, &additionalDefinitions)
	for name, itemSchema := range additionalDefinitions.Definitions {
		extensionFile.VariablesRefs = append(extensionFile.VariablesRefs, name)
		o.operationsConfigData.Definitions.Store(name, openapi2conv.ToV3SchemaRef(itemSchema))
	}
	schema.Value.Extensions = nil
	return openapi2conv.ToV3SchemaRef(&schema)
}
