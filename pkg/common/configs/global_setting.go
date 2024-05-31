// Package configs
/*
 通过fileloader.ModelText和fileloader.Model来管理GlobalSetting配置
 settingDefaultText读取embed内嵌的默认配置
 GlobalSettingRoot读取store/config/global_setting.json的配置
 envDefaultName在非dev模型下添加.prod后缀，即使用.env.prod作为默认配置
 envEffectiveName根据命令行参数--active的值来添加后缀
 env变更会触发问题收集和引擎编译重启
 通过afterInit和afterUpdate实现日志文件初始化、日志事件响应、日志等级初始化/切换、语言初始化/切换等功能
*/
package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/wundergraph/wundergraph/pkg/node"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

type (
	GlobalSetting struct {
		NodeOptions       *wgpb.NodeOptions       `json:"nodeOptions"`       // WunderGraphConfiguration.UserDefinedApi.NodeOptions
		ServerOptions     *wgpb.ServerOptions     `json:"serverOptions"`     // WunderGraphConfiguration.UserDefinedApi.ServerOptions
		CorsConfiguration *wgpb.CorsConfiguration `json:"corsConfiguration"` // WunderGraphConfiguration.UserDefinedApi.Cors
		SecurityConfig

		Appearance    *Appearance       `json:"appearance"`
		ConsoleLogger *lumberjackLogger `json:"consoleLogger"`
		BuildInfo     *node.BuildInfo   `json:"buildInfo"`
	}
	Appearance struct {
		Language string `json:"language"`
	}
	SecurityConfig struct {
		AllowedHostNames []*wgpb.ConfigurationVariable `json:"allowedHostNames"` // WunderGraphConfiguration.AllowedHostNames

		AuthorizedRedirectUris       []*wgpb.ConfigurationVariable `json:"authorizedRedirectUris"`
		AuthorizedRedirectUriRegexes []*wgpb.ConfigurationVariable `json:"authorizedRedirectUriRegexes"`
		AllowedReport                bool                          `json:"allowedReport"`
		EnableCSRFProtect            bool                          `json:"enableCSRFProtect"`
		ForceHttpsRedirects          bool                          `json:"forceHttpsRedirects"`
		GlobalRateLimit              *wgpb.OperationRateLimit      `json:"globalRateLimit"`
	}
)

var (
	GlobalSettingRoot     *fileloader.Model[GlobalSetting]
	AuthenticationKeyText *fileloader.ModelText[GlobalSetting]
)

func init() {
	AuthenticationKeyText = &fileloader.ModelText[GlobalSetting]{
		Root:      utils.StringDot,
		Extension: fileloader.ExtKey,
		TextRW:    &fileloader.SingleTextRW[GlobalSetting]{Name: consts.KeyAuthentication},
	}
	utils.RegisterInitMethod(11, func() {
		settingDefaultText := &fileloader.ModelText[GlobalSetting]{
			Root:      embed.DefaultRoot,
			Extension: fileloader.ExtJson,
			TextRW:    &fileloader.EmbedTextRW{EmbedFiles: &embed.DefaultFs, Name: consts.GlobalSetting},
		}
		settingDefaultText.Init()

		GlobalSettingRoot = &fileloader.Model[GlobalSetting]{
			Root:      utils.NormalizePath(consts.RootStore, consts.StoreConfigParent),
			Extension: fileloader.ExtJson,
			DataHook: &fileloader.DataHook[GlobalSetting]{
				AfterInit: func(datas map[string]*GlobalSetting) {
					data := datas[consts.GlobalSetting]
					if consoleLogger := data.ConsoleLogger; consoleLogger != nil && len(consoleLogger.Filename) > 0 {
						addLoggerWriteSyncer(data.ConsoleLogger)
						consoleLogger.initConsoleLogger()
					}

					if appearance := data.Appearance; appearance != nil {
						i18n.SwitchErrcodeLocale(appearance.Language)
						i18n.SwitchDirectiveLocale(appearance.Language)
						i18n.SwitchPrismaErrorLocale(appearance.Language)
					}

					if nodeLogger := data.NodeOptions.Logger; nodeLogger != nil {
						setLoggerLevel(utils.GetVariableString(nodeLogger.Level, lazyLogger))
					}
				},
				AfterUpdate: func(data *GlobalSetting, modifies *fileloader.DataModifies, _ string, _ ...string) {
					if detail, ok := modifies.GetModifyDetail("appearance.language"); ok {
						language := string(detail.Target)
						i18n.SwitchErrcodeLocale(language)
						i18n.SwitchDirectiveLocale(language)
						i18n.SwitchPrismaErrorLocale(language)
					}

					if _, ok := modifies.GetModifyDetail("nodeOptions.logger.level"); ok {
						setLoggerLevel(utils.GetVariableString(data.NodeOptions.Logger.Level))
					}
				},
			},
			DataRW: &fileloader.SingleDataRW[GlobalSetting]{
				InitDataBytes: settingDefaultText.GetFirstCache(),
				DataName:      consts.GlobalSetting,
			},
		}
		GlobalSettingRoot.Init(lazyLogger)
		AuthenticationKeyText.RelyModel = GlobalSettingRoot
		AuthenticationKeyText.Init()
		utils.AddBuildAndStartFuncWatcher(func(f func()) { GlobalSettingRoot.DataHook.AfterMutate = f })
	})
}
