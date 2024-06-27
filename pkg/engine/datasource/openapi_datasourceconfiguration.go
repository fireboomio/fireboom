// Package datasource
/*
 实现resolveOpenapi接口，返回引擎所需的数据源配置
*/
package datasource

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/wundergraph/wundergraph/pkg/interpolate"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"strings"
)

func newResolveDatasourceConfiguration(action *actionOpenapi) *resolveDataSourceConfiguration {
	return &resolveDataSourceConfiguration{
		dsModelName:   models.DatasourceRoot.GetModelName(),
		dsName:        action.ds.Name,
		customRest:    action.ds.CustomRest,
		customRestMap: make(map[string]*wgpb.DataSourceCustom_REST),
		openapi:       action,
	}
}

const (
	argumentsFormat       = `{{ .arguments.%s }}`
	requestBodyFormat     = `{{ .arguments.%s_input_object }}`
	headerArgumentsFormat = `{{ .request.headers.%s }}`
)

type resolveDataSourceConfiguration struct {
	dsModelName, dsName string
	customRest          *models.CustomRest
	customRestMap       map[string]*wgpb.DataSourceCustom_REST
	openapi             *actionOpenapi
}

func (r *resolveDataSourceConfiguration) resolve(item *resolveItem) {
	requestPath := item.path
	var existPathParam bool
	fetchConfig := &wgpb.FetchConfiguration{Method: item.method, Header: make(map[string]*wgpb.HTTPHeader)}
	fetchConfig.BaseUrl = r.customRest.BaseUrl
	rewriteHttpHeaders(r.customRest.Headers, fetchConfig.Header)
	for _, paramItem := range item.parameters {
		paramItemValue := paramItem.Value
		paramItemValueArgs := fmt.Sprintf(argumentsFormat, utils.NormalizeName(paramItemValue.Name))
		switch paramItemValue.In {
		case openapi3.ParameterInQuery:
			fetchConfig.Query = append(fetchConfig.Query, &wgpb.URLQueryConfiguration{
				Name:  paramItemValue.Name,
				Value: paramItemValueArgs,
			})
		case openapi3.ParameterInHeader:
			fetchConfig.Header[paramItemValue.Name] = &wgpb.HTTPHeader{
				Values: []*wgpb.ConfigurationVariable{utils.MakePlaceHolderVariable(paramItemValueArgs)},
			}
		case openapi3.ParameterInPath:
			existPathParam = true
			requestPath = strings.ReplaceAll(requestPath, `{`+paramItemValue.Name+`}`, paramItemValueArgs)
		default:
			logger.Warn("param in cookie not supported",
				zap.String(r.dsModelName, r.dsName),
				zap.String("operationId", item.operationId),
				zap.String("parameterName", paramItemValue.Name))
		}
	}

	if existPathParam {
		fetchConfig.Path = utils.MakePlaceHolderVariable(requestPath)
	} else {
		fetchConfig.Path = utils.MakeStaticVariable(requestPath)
	}

	if item.requestBody.schema != nil {
		bodyArgName := makeBodyArgumentName(item.requestBody.schema, item.operationId)
		fetchConfig.Body = utils.MakePlaceHolderVariable(fmt.Sprintf(requestBodyFormat, utils.NormalizeName(bodyArgName)))
		fetchConfig.UrlEncodeBody = item.requestBody.contentType == echo.MIMEApplicationForm
	}

	fetchConfig.RequestContentType = item.requestBody.contentType
	fetchConfig.ResponseContentType = item.succeedResponse.contentType
	customRest := &wgpb.DataSourceCustom_REST{
		Fetch:             fetchConfig,
		ResponseExtractor: r.customRest.ResponseExtractor,
		Subscription: &wgpb.RESTSubscriptionConfiguration{
			Enabled:  item.subscribed.Enabled,
			DoneData: item.subscribed.DoneData,
		},
	}
	/*for code, content := range item.responses {
		contentType, _, _ := r.graphqlResolve.visitSchema(content.schema, false, utils.JoinString("_", item.operationId, code), ast.Object)
		customRest.StatusCodeTypeMappings = append(customRest.StatusCodeTypeMappings, &wgpb.StatusCodeTypeMapping{
			StatusCode:               cast.ToInt64(code),
			InjectStatusCodeIntoBody: true,
			TypeName:                 contentType.prototype().string(),
		})
	}*/
	fieldName := fmt.Sprintf("%s.%s_%s", item.typeName, r.dsName, item.operationId)
	r.customRestMap[fieldName] = customRest
}

// body类型的参数仅需要引用即可，具体定义在#/components/schemas/中
func makeBodyArgumentName(schema *openapi3.SchemaRef, operationId string) string {
	if ref := schema.Ref; ref != "" {
		return strings.TrimPrefix(ref, interpolate.Openapi3SchemaRefPrefix)
	}

	if items := schema.Value.Items; items != nil {
		return makeBodyArgumentName(items, operationId)
	}

	return operationId
}
