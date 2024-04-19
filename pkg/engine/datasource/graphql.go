// Package datasource
/*
 graphql类型数据源的实现
*/
package datasource

import (
	"errors"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	iterator "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

func init() {
	actionMap[wgpb.DataSourceKind_GRAPHQL] = func(ds *models.Datasource, _ string) Action { return &actionGraphql{ds: ds} }
}

const (
	graphqlResultDataPath   = "data.__schema"
	graphqlResultErrorsPath = "errors.0.message"
)

var caseInsensitive = iterator.ConfigCompatibleWithStandardLibrary

type actionGraphql struct {
	ds *models.Datasource
}

func (a *actionGraphql) Introspect() (graphqlSchema string, err error) {
	graphqlConfig := a.ds.CustomGraphql
	if graphqlConfig == nil {
		err = i18n.NewCustomErrorWithMode(datasourceModelName, nil, i18n.StructParamEmtpyError, "customGraphql")
		return
	}

	if graphqlConfig.SchemaFilepath != "" {
		// schemaFilepath不为空即定义了上传的graphql文件路径
		graphqlSchema, err = models.DatasourceUploadGraphql.Read(a.ds.Name)
		return
	}

	var __schema string
	if graphqlConfig.Customized {
		// 自定义数据源直接读取钩子内省的文本
		if __schema, err = models.DatasourceCustomize.Read(a.ds.Name); err != nil {
			return
		}
	} else {
		// 发送post请求获取内省的文本
		introspectBytes := configs.IntrospectText.GetFirstCache()
		headers := map[string]string{echo.HeaderContentType: echo.MIMEApplicationJSON}
		for s, httpHeader := range graphqlConfig.Headers {
			for _, value := range httpHeader.Values {
				headers[s] = utils.GetVariableString(value)
			}
		}
		var respBody []byte
		if respBody, err = utils.HttpPost(graphqlConfig.Endpoint, introspectBytes, headers, 5); err != nil {
			return
		}

		if errorMsg := gjson.GetBytes(respBody, graphqlResultErrorsPath); errorMsg.Exists() {
			err = errors.New(errorMsg.String())
			return
		}

		__schema = gjson.GetBytes(respBody, graphqlResultDataPath).String()
	}

	var result schema
	if err = caseInsensitive.Unmarshal([]byte(__schema), &result); err != nil {
		return
	}

	// 格式化graphql文档成文本并缓存到本地
	graphqlSchema = formatSchemaString(&result)
	cacheGraphqlSchema(a.ds.Name, graphqlSchema)
	return
}

func (a *actionGraphql) BuildDataSourceConfiguration(doc *ast.SchemaDocument) (*wgpb.DataSourceConfiguration, error) {
	graphqlConfig := a.ds.CustomGraphql
	graphqlUrl, err := graphqlConfig.GetGraphqlUrl(a.ds.Name)
	if err != nil {
		return nil, err
	}

	url := utils.MakeStaticVariable(graphqlUrl)
	rewriteHeaders := make(map[string]*wgpb.HTTPHeader)
	rewriteHttpHeaders(graphqlConfig.Headers, rewriteHeaders)
	return &wgpb.DataSourceConfiguration{
		CustomGraphql: &wgpb.DataSourceCustom_GraphQL{
			Fetch: &wgpb.FetchConfiguration{
				Url:    url,
				Method: wgpb.HTTPMethod_POST,
				Header: rewriteHeaders,
			},
			Subscription: &wgpb.GraphQLSubscriptionConfiguration{
				Enabled: doc.Definitions.ForName(consts.TypeSubscription) != nil,
				Url:     url,
				UseSSE:  graphqlConfig.Customized,
			},
			Federation: &wgpb.GraphQLFederationConfiguration{},
		},
	}, nil
}

func (a *actionGraphql) RuntimeDataSourceConfiguration(config *wgpb.DataSourceConfiguration) (configs []*wgpb.DataSourceConfiguration, fields []*wgpb.FieldConfiguration, err error) {
	var graphqlSchema string
	schemaPath := models.DatasourceUploadGraphql.GetPath(config.Id)
	if utils.NotExistFile(schemaPath) {
		graphqlSchema, err = CacheGraphqlSchemaText.Read(config.Id)
	} else {
		graphqlSchema, err = models.DatasourceUploadGraphql.Read(config.Id)
	}
	if err != nil {
		return
	}

	customGraphql := *config.CustomGraphql
	customGraphql.UpstreamSchema = graphqlSchema
	configs, fields = copyDatasourceWithRootNodes(config, func(_ *wgpb.TypeField, configItem *wgpb.DataSourceConfiguration) bool {
		configItem.CustomGraphql = &customGraphql
		return true
	})
	return
}

// 将httpHeader写入到目标对象
// 特殊处理PLACEHOLDER_CONFIGURATION_VARIABLE，添加占位符包裹
func rewriteHttpHeaders(source, target map[string]*wgpb.HTTPHeader) {
	for name, header := range source {
		var headerValues []*wgpb.ConfigurationVariable
		for _, value := range header.Values {
			if value.Kind == wgpb.ConfigurationVariableKind_PLACEHOLDER_CONFIGURATION_VARIABLE {
				val := utils.GetVariableString(value)
				value = utils.MakePlaceHolderVariable(fmt.Sprintf(headerArgumentsFormat, val))
			}
			headerValues = append(headerValues, value)
		}
		target[name] = &wgpb.HTTPHeader{Values: headerValues}
	}
}
