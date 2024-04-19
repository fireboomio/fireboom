// Package configs
/*
 通过fileloader.ModelText和fileloader.Model来管理资源配置
 applicationRoot主要是为了验证对properties文件的支持，添加代码级别的配置支持，包括跟路径、认证路由、日志白名单等
 BannerText读取内置的文本，即控制台直接输出的飞布LOGO
 IntrospectText读取内置的文本，用作graphql数据源的内省查询的请求体
*/
package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/magiconair/properties"
	"strings"
)

type (
	applicationProperties struct {
		ContextPath           string `properties:"context.path"`
		RequestLoggerSkippers string `properties:"request.logger.skippers"`
		AuthenticationUrls    string `properties:"authentication.urls"`
		EngineForwardRequests string `properties:"engine.forward.requests"`
		ContactAddress        string `properties:"contact.address"`
	}
	application struct {
		ContextPath           string
		RequestLoggerSkippers []string
		AuthenticationUrls    []string
		EngineForwardRequests []string
		ContactAddress        string
	}
)

var (
	BannerText      *fileloader.ModelText[any]
	IntrospectText  *fileloader.ModelText[any]
	ApplicationData *application
)

func init() {
	BannerText = &fileloader.ModelText[any]{
		Root:      embed.ResourceRoot,
		Extension: fileloader.ExtTxt,
		TextRW:    &fileloader.EmbedTextRW{EmbedFiles: &embed.ResourceFs, Name: consts.ResourceBanner},
	}

	IntrospectText = &fileloader.ModelText[any]{
		Root:      embed.ResourceRoot,
		Extension: fileloader.ExtJson,
		TextRW:    &fileloader.EmbedTextRW{EmbedFiles: &embed.ResourceFs, Name: consts.ResourceIntrospect},
	}

	applicationRoot := &fileloader.Model[applicationProperties]{
		Root:      embed.ResourceRoot,
		Extension: fileloader.ExtProperties,
		DataRW: &fileloader.EmbedDataRW[applicationProperties]{
			EmbedFiles: &embed.ResourceFs,
			DataName:   consts.ResourceApplication,
			Unmarshal: func(dataBytes []byte) (data *applicationProperties, err error) {
				props, err := properties.Load(dataBytes, properties.UTF8)
				if err != nil {
					return
				}

				data = &applicationProperties{}
				err = props.Decode(data)
				return
			},
		},
	}

	utils.RegisterInitMethod(12, func() {
		BannerText.Init()
		IntrospectText.Init()
		applicationRoot.Init()

		data := applicationRoot.FirstData()
		ApplicationData = &application{
			ContextPath:           data.ContextPath,
			RequestLoggerSkippers: strings.Split(data.RequestLoggerSkippers, utils.StringComma),
			AuthenticationUrls:    strings.Split(data.AuthenticationUrls, utils.StringComma),
			EngineForwardRequests: strings.Split(data.EngineForwardRequests, utils.StringComma),
			ContactAddress:        data.ContactAddress,
		}
	})
}
