// Package configs
/*
 通过fileloader.ModelText和fileloader.Model来管理.env配置
 envDefaultText读取embed内嵌的默认配置
 EnvEffectiveRoot读取项目工作目录下的.env配置
 envDefaultName在非dev模型下添加.prod后缀，即使用.env.prod作为默认配置
 envEffectiveName根据命令行参数--active的值来添加后缀
 env变更会触发问题收集和引擎编译重启
*/
package configs

import (
	"bytes"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
	"strings"
)

var EnvEffectiveRoot *fileloader.Model[gotenv.Env]

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	utils.RegisterInitMethod(10, func() {
		envDefaultName := consts.DefaultEnv
		if !utils.GetBoolWithLockViper(consts.DevMode) {
			envDefaultName += utils.StringDot + consts.DefaultProdActive
		}
		envDefaultText := &fileloader.ModelText[gotenv.Env]{
			Root:             embed.DefaultRoot,
			ExtensionIgnored: true,
			TextRW: &fileloader.EmbedTextRW{
				EmbedFiles: &embed.DefaultFs,
				Name:       envDefaultName,
			},
		}
		envDefaultText.Init()

		envEffectiveName := consts.DefaultEnv
		if mode := utils.GetStringWithLockViper(consts.ActiveMode); mode != "" {
			envEffectiveName += utils.StringDot + mode
		}
		EnvEffectiveRoot = &fileloader.Model[gotenv.Env]{
			Root:             utils.StringDot,
			ExtensionIgnored: true,
			DataHook: &fileloader.DataHook[gotenv.Env]{
				AfterInit: func(datas map[string]*gotenv.Env) {
					for k, v := range *datas[envEffectiveName] {
						utils.SetWithLockViper(k, v, true)
					}
				},
				AfterUpdate: func(data *gotenv.Env, modifies *fileloader.DataModifies, _ string, _ ...string) {
					for key, detail := range *modifies {
						switch detail.Name {
						case fileloader.Add, fileloader.Overwrite:
							utils.SetWithLockViper(key, string(detail.Target))
						case fileloader.Remove:
							utils.SetWithLockViper(key, nil)
						}
					}
				},
			},
			DataRW: &fileloader.SingleDataRW[gotenv.Env]{
				InitDataBytes:        envDefaultText.GetFirstCache(),
				IgnoreMergeIfExisted: utils.GetBoolWithLockViper(consts.IgnoreMergeEnvironment),
				DataName:             envEffectiveName,
				Unmarshal: func(dataBytes []byte) (*gotenv.Env, error) {
					data := gotenv.Parse(bytes.NewReader(dataBytes))
					return &data, nil
				},
				Marshal: func(data *gotenv.Env) ([]byte, error) {
					envStr, err := gotenv.Marshal(*data)
					return []byte(envStr), err
				},
			},
		}
		EnvEffectiveRoot.Init(lazyLogger)
		AddFileLoaderQuestionCollector(EnvEffectiveRoot.GetModelName(), nil)
		utils.AddBuildAndStartFuncWatcher(func(f func()) { EnvEffectiveRoot.DataHook.AfterMutate = f })
	})
}
