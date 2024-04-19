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

func (o *operations) resolveFunctionSchema(schemaBytes []byte, extensionFile *ExtensionOperationFile) *openapi3.SchemaRef {
	var schema openapi3.SchemaRef
	if err := json.Unmarshal(schemaBytes, &schema); err != nil || schema.Value == nil {
		return nil
	}

	var additionalDefinitions functionSchemaDefinitions
	_ = json.Unmarshal(schemaBytes, &additionalDefinitions)
	for name, itemSchema := range additionalDefinitions.Definitions {
		extensionFile.VariablesRefs = append(extensionFile.VariablesRefs, name)
		o.operationsConfigData.Definitions[name] = openapi2conv.ToV3SchemaRef(itemSchema)
	}
	return openapi2conv.ToV3SchemaRef(&schema)
}

func (o *operations) resolveExtensionOperation(extensionFunc func(string, *ExtensionOperationFile), operation *models.Operation, modelText *fileloader.ModelText[models.Operation]) (operationResult *wgpb.Operation, err error) {
	if !operation.Enabled {
		return
	}

	content, err := modelText.Read(operation.Path)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(content), &operationResult); err != nil {
		err = i18n.NewCustomErrorWithMode(o.modelName, err, i18n.LoaderFileUnmarshalError, modelText.GetPath(operation.Path))
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
	o.mergeGlobalOperation(operation, operationResult)
	o.resolveOperationHook(operationResult)
	operationResult.HooksConfiguration = &wgpb.OperationHooksConfiguration{
		HttpTransportBeforeRequest: operationResult.HooksConfiguration.HttpTransportBeforeRequest,
		HttpTransportAfterResponse: operationResult.HooksConfiguration.HttpTransportAfterResponse,
	}

	extensionFile := &ExtensionOperationFile{
		BaseOperationFile: BaseOperationFile{
			OperationName:       operationResult.Name,
			FilePath:            modelText.GetPath(operationResult.Path),
			OperationType:       operationResult.OperationType,
			AuthorizationConfig: operationResult.AuthorizationConfig,
		},
		ModulePath: modelText.Root,
	}
	extensionFile.Variables = o.resolveFunctionSchema([]byte(operationResult.VariablesSchema), extensionFile)
	extensionFile.Response = o.resolveFunctionSchema([]byte(operationResult.ResponseSchema), extensionFile)
	operationResult.VariablesSchema = ""
	operationResult.ResponseSchema = ""

	extensionFunc(operation.Path, extensionFile)
	return
}
