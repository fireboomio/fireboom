// Package models
/*
 使用fileloader.Model管理sdk配置
 读取store/sdk下的文件，支持逻辑删除，变更后会触发引擎编译
 通过switchServerSdkWatchers实现动态的钩子父目录切换
*/
package models

import (
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"go.uber.org/zap"
	"math"
	"regexp"
)

type sdkType string

const (
	SdkClient       sdkType = "client"
	SdkServer       sdkType = "server"
	fieldEnabled            = "enabled"
	fieldOutputPath         = "outputPath"
	latestFlag              = "latest"
	sdkModelName            = "sdk"
)

type Sdk struct {
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	Type               sdkType  `json:"type"`
	Language           string   `json:"language"`
	Extension          string   `json:"extension"`
	GitUrl             string   `json:"gitUrl"`
	GitBranch          string   `json:"gitBranch"`
	GitCommitHash      string   `json:"gitCommitHash"`
	OutputPath         string   `json:"outputPath"`
	CodePackage        string   `json:"codePackage"`
	UpperFirstBasename bool     `json:"upperFirstBasename"`
	Keywords           []string `json:"keywords"`

	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
	DeleteTime  string `json:"deleteTime"`
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

var SdkRoot *fileloader.Model[Sdk]

func init() {
	SdkRoot = &fileloader.Model[Sdk]{
		Root:      utils.NormalizePath(consts.RootStore, consts.StoreSdkParent),
		Extension: fileloader.ExtJson,
		DataHook: &fileloader.DataHook[Sdk]{
			OnInsert: func(item *Sdk) (err error) {
				item.CreateTime = utils.TimeFormatNow()
				item.GitCommitHash = ""
				return item.gitClone(sdkOnInsert)
			},
			OnUpdate: func(src, dst *Sdk, user string) error {
				if user != fileloader.SystemUser {
					dst.UpdateTime = utils.TimeFormatNow()
				}
				if dst.GitCommitHash != latestFlag {
					return nil
				}

				dst.GitCommitHash = src.GitCommitHash
				return dst.gitResetAndPull(sdkOnUpdate)
			},
			AfterInit: func(datas map[string]*Sdk) {
				for _, sdk := range datas {
					if err := sdk.gitResetAndPull(sdkAfterInit); err == nil {
						_ = SdkRoot.InsertOrUpdate(sdk)
					}
				}
			},
			AfterInsert: func(item *Sdk, user string) bool {
				item.callSwitchServerSdkWatchers()
				return item.Enabled
			},
			AfterUpdate: func(item *Sdk, modify *fileloader.DataModifies, user string, _ ...string) {
				_, enabledOk := (*modify)[fieldEnabled]
				_, outputPathOk := (*modify)[fieldOutputPath]
				if enabledOk || outputPathOk {
					item.callSwitchServerSdkWatchers()
				}
			},
		},
		DataRW: &fileloader.MultipleDataRW[Sdk]{
			GetDataName: func(item *Sdk) string { return item.Name },
			SetDataName: func(item *Sdk, name string) { item.Name = name },
			Filter:      func(item *Sdk) bool { return item.DeleteTime == "" },
			LogicDelete: func(item *Sdk) { item.DeleteTime = utils.TimeFormatNow() },
		},
	}

	utils.RegisterInitMethod(20, func() {
		logger = zap.L()
		SdkRoot.Init()
		configs.AddFileLoaderQuestionCollector(SdkRoot.GetModelName(), func(dataName string) map[string]any {
			data, _ := SdkRoot.GetByDataName(dataName)
			if data == nil {
				return nil
			}

			return map[string]any{fieldEnabled: data.Enabled}
		})
		utils.AddBuildAndStartFuncWatcher(func(f func()) { SdkRoot.DataHook.AfterMutate = f })
	})
	utils.RegisterInitMethod(math.MaxInt, func() {
		if enabledSdk := GetEnabledServerSdk(); enabledSdk != nil {
			enabledSdk.callSwitchServerSdkWatchers()
		}
	})
}

type switchServerSdkWatcher func(outputPath, codePackage, extension string, upperFirstBasename bool)

var switchServerSdkWatchers []switchServerSdkWatcher

func AddSwitchServerSdkWatcher(watcher switchServerSdkWatcher) {
	switchServerSdkWatchers = append(switchServerSdkWatchers, watcher)
}

var (
	logger          *zap.Logger
	githubRegexp    = regexp.MustCompile("https://github.com")
	githubRawRegexp = regexp.MustCompile("https://raw.githubusercontent.com")
)

func (s *Sdk) callSwitchServerSdkWatchers() {
	if s.Type != SdkServer {
		return
	}

	var outputPath, codePackage, extension string
	if s.Enabled {
		outputPath = s.OutputPath
		extension = s.Extension
		codePackage = s.CodePackage

		for _, other := range SdkRoot.ListByCondition(func(item *Sdk) bool { return item.Type == SdkServer && item.Enabled && item.Name != s.Name }) {
			other.Enabled = false
			_ = SdkRoot.InsertOrUpdate(other)
		}
	}

	for _, watcher := range switchServerSdkWatchers {
		watcher(outputPath, codePackage, extension, s.UpperFirstBasename)
	}
}

func ReplaceGithubProxyUrl(url string) string {
	replaceEnvFunc := func(name string, regexp *regexp.Regexp) {
		if value := utils.GetStringWithLockViper(name); value != "" {
			url = regexp.ReplaceAllString(url, value)
		}
	}

	replaceEnvFunc(consts.GithubProxyUrl, githubRegexp)
	replaceEnvFunc(consts.GithubRawProxyUrl, githubRawRegexp)
	return url
}

func GetEnabledServerSdk() *Sdk {
	return SdkRoot.FirstData(func(item *Sdk) bool { return item.Type == SdkServer && item.Enabled })
}
