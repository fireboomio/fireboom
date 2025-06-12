package server

import (
	"bytes"
	"context"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/datasource"
	"fireboom-server/pkg/websocket"
	"fmt"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/swaggo/swag"
	"go.uber.org/zap"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
)

type mcpServer struct {
	e        *echo.Echo
	s        *server.MCPServer
	basePath string
}

func initMpcServer(e *echo.Echo) {
	swagger, err := swag.ReadDoc(cloudInstanceName)
	if err != nil {
		logger.Error("read swagger failed",
			zap.String("name", cloudInstanceName))
		return
	}

	var doc *openapi2.T
	if err = json.Unmarshal([]byte(swagger), &doc); err != nil {
		logger.Error("unmarshal swagger failed",
			zap.String("name", cloudInstanceName),
			zap.Error(err))
		return
	}

	docV3, err := openapi2conv.ToV3(doc)
	if err != nil {
		return
	}

	if err = openapi3.NewLoader().ResolveRefsIn(docV3, nil); err != nil {
		return
	}

	// Create a new MCP server
	m := &mcpServer{
		e: e,
		s: server.NewMCPServer(
			docV3.Info.Title,
			docV3.Info.Version,
			server.WithResourceCapabilities(true, true),
			server.WithLogging(),
			server.WithRecovery(),
		),
		basePath: doc.BasePath,
	}

	for path, pathItem := range docV3.Paths {
		m.buildMpcTool(path, http.MethodPost, pathItem.Post)
		m.buildMpcTool(path, http.MethodDelete, pathItem.Delete)
		m.buildMpcTool(path, http.MethodPut, pathItem.Put)
		m.buildMpcTool(path, http.MethodGet, pathItem.Get)

		m.buildMpcTool(path, http.MethodHead, pathItem.Head)
		m.buildMpcTool(path, http.MethodPatch, pathItem.Patch)
		m.buildMpcTool(path, http.MethodOptions, pathItem.Options)
	}

	mcpAddress := fmt.Sprintf(":%s", utils.GetStringWithLockViper(consts.McpPort))
	websocket.AddOnFirstStartedHook(func() { logger.Info("mcp server started", zap.String("address", mcpAddress)) }, math.MaxInt)
	// Start the server
	if err = server.NewSSEServer(m.s).Start(mcpAddress); err != nil {
		logger.Error("server error",
			zap.String("name", cloudInstanceName),
			zap.Error(err))
	}
}

const parameterInBody = "body"

func (m *mcpServer) buildMpcTool(path, method string, operation *openapi3.Operation) {
	if operation == nil {
		return
	}

	schema := &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	if operation.RequestBody != nil {
		requestSchema, _ := datasource.FetchRequestBodyResolveSchema(operation.RequestBody.Value)
		if requestSchema == nil {
			return
		}
		schema.Value.Properties[parameterInBody] = requestSchema
		if len(requestSchema.Value.Required) > 0 {
			schema.Value.Required = append(schema.Value.Required, parameterInBody)
		}
	}
	var pathSchema, querySchema, headerSchema *openapi3.SchemaRef
	for _, item := range operation.Parameters {
		switch item.Value.In {
		case openapi3.ParameterInPath:
			if pathSchema == nil {
				pathSchema = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
			}
			if item.Value.Required {
				pathSchema.Value.Required = append(pathSchema.Value.Required, item.Value.Name)
			}
			pathSchema.Value.Properties[item.Value.Name] = item.Value.Schema
		case openapi3.ParameterInQuery:
			if querySchema == nil {
				querySchema = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
			}
			if item.Value.Required {
				querySchema.Value.Required = append(querySchema.Value.Required, item.Value.Name)
			}
			querySchema.Value.Properties[item.Value.Name] = item.Value.Schema
		case openapi3.ParameterInHeader:
			if headerSchema == nil {
				headerSchema = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
			}
			if item.Value.Required {
				headerSchema.Value.Required = append(headerSchema.Value.Required, item.Value.Name)
			}
			headerSchema.Value.Properties[item.Value.Name] = item.Value.Schema
		}
	}
	if pathSchema != nil {
		schema.Value.Properties[openapi3.ParameterInPath] = pathSchema
		if len(pathSchema.Value.Required) > 0 {
			schema.Value.Required = append(schema.Value.Required, openapi3.ParameterInPath)
		}
	}
	if querySchema != nil {
		schema.Value.Properties[openapi3.ParameterInQuery] = querySchema
		if len(querySchema.Value.Required) > 0 {
			schema.Value.Required = append(schema.Value.Required, openapi3.ParameterInQuery)
		}
	}
	if headerSchema != nil {
		schema.Value.Properties[openapi3.ParameterInHeader] = headerSchema
		if len(headerSchema.Value.Required) > 0 {
			schema.Value.Required = append(schema.Value.Required, openapi3.ParameterInHeader)
		}
	}

	removeRefFromSchema(schema)
	schemaBytes, _ := json.Marshal(schema)
	tool := mcp.NewToolWithRawSchema(utils.JoinStringWithDot(path, method), operation.Description, schemaBytes)
	m.s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		finalPath := m.basePath + path
		if _path, ok := request.Params.Arguments[openapi3.ParameterInPath]; ok {
			for k, v := range _path.(map[string]interface{}) {
				finalPath = strings.ReplaceAll(finalPath, fmt.Sprintf("{%s}", k), fmt.Sprintf("%v", v))
			}
		}
		var queryValues []string
		if _query, ok := request.Params.Arguments[openapi3.ParameterInQuery]; ok {
			for k, v := range _query.(map[string]interface{}) {
				queryValues = append(queryValues, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if len(queryValues) > 0 {
			finalPath += "?" + strings.Join(queryValues, "&")
		}
		var bodyBytes []byte
		if _body, ok := request.Params.Arguments[parameterInBody]; ok {
			bodyBytes, _ = json.Marshal(_body)
		}
		req := httptest.NewRequest(method, finalPath, bytes.NewReader(bodyBytes))
		if _header, ok := request.Params.Arguments[openapi3.ParameterInHeader]; ok {
			for k, v := range _header.(map[string]interface{}) {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
		// 创建ResponseRecorder
		rec := httptest.NewRecorder()
		// 调用Echo处理请求
		m.e.ServeHTTP(rec, req)

		return &mcp.CallToolResult{
			IsError: rec.Code >= http.StatusBadRequest,
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: rec.Body.String()}},
		}, nil
	})
}

func removeRefFromSchema(schema *openapi3.SchemaRef) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		schema.Ref = ""
	}
	for _, v := range schema.Value.Properties {
		removeRefFromSchema(v)
	}
	for _, v := range schema.Value.AllOf {
		removeRefFromSchema(v)
	}
	for _, v := range schema.Value.AnyOf {
		removeRefFromSchema(v)
	}
	for _, v := range schema.Value.OneOf {
		removeRefFromSchema(v)
	}
	removeRefFromSchema(schema.Value.Items)
	removeRefFromSchema(schema.Value.Not)
	removeRefFromSchema(schema.Value.AdditionalProperties.Schema)
}
