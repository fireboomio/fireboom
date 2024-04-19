// Package configs
/*
 通过fileloader.ModelText和fileloader.Model来管理License信息
 licenseDefault读取embed内嵌的默认配置
 licenseText读取当前工作目录下license.key的文本
 licenseText校验成后会覆盖licenseDefault的配置
 license验证合法性、有效期、功能限制等，并通过日志收集器发送警告消息
*/
package configs

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	json "github.com/json-iterator/go"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"time"
)

var (
	logger          *zap.Logger
	LicenseInfoData licenseInfo
)

type (
	licenseInfo struct {
		Existed        bool           `json:"existed"`
		DefaultLimits  map[string]int `json:"defaultLimits"`
		WsPushRequired []string       `json:"wsPushRequired"`
		UserCode       string         `json:"userCode"`
		userLicense
	}
	userLicense struct {
		Type       consts.LicenseType `json:"type"`
		UserLimits map[string]int     `json:"userLimits"`
		ExpireTime time.Time          `json:"expireTime"`
	}
)

func init() {
	licenseDefault := &fileloader.Model[licenseInfo]{
		Root:      embed.DefaultRoot,
		Extension: fileloader.ExtJson,
		DataRW: &fileloader.EmbedDataRW[licenseInfo]{
			EmbedFiles: &embed.DefaultFs,
			DataName:   consts.KeyLicense,
		},
	}
	licenseRoot := &fileloader.Model[userLicense]{
		Root:             utils.StringDot,
		Extension:        fileloader.ExtKey,
		LoadErrorIgnored: true,
		DataRW: &fileloader.SingleDataRW[userLicense]{
			DataName: consts.KeyLicense,
			Unmarshal: func(dataBytes []byte) (data *userLicense, _ error) {
				licenseBytes := utils.DecodeLicenseKey(string(dataBytes))
				_ = json.Unmarshal(licenseBytes, &data)
				return
			},
		},
	}
	utils.RegisterInitMethod(12, func() { licenseDefault.Init() })
	utils.RegisterInitMethod(15, func() {
		logger = zap.L()
		licenseRoot.Init()

		defaultData := licenseDefault.FirstData()
		LicenseInfoData.DefaultLimits = defaultData.DefaultLimits
		LicenseInfoData.WsPushRequired = defaultData.WsPushRequired
		LicenseInfoData.UserCode = utils.GenerateUserCode()
		if licenseData := licenseRoot.FirstData(); licenseData != nil {
			LicenseInfoData.userLicense, LicenseInfoData.Existed = *licenseData, true
		}
		if LicenseInfoData.Type == "" {
			LicenseInfoData.Type = consts.LicenseTypeCommunity
		}
	})
	utils.InvokeFunctionLimit = invokeFunctionLimit
}

// 根据模块、数量限制验证license合法性、有效期、功能限制等，并通过日志收集器发送警告消息
func invokeFunctionLimit(modelName string, amount ...int) (invoked bool) {
	modelAmount := 1
	if len(amount) > 0 {
		modelAmount = amount[0]
	}
	var limits map[string]int
	validLicense := LicenseInfoData.ExpireTime.After(time.Now())
	if validLicense {
		limits = LicenseInfoData.UserLimits
	} else {
		limits = LicenseInfoData.DefaultLimits
	}

	functionLimit := limits[modelName]
	if functionLimit == -1 || modelAmount <= functionLimit {
		return
	}

	invoked = true
	var cause string
	defer func() {
		if invoked && slices.Contains(LicenseInfoData.WsPushRequired, modelName) {
			logger.Warn("invoke function use limits, please contact developer with userCode",
				zap.Any(consts.LicenseStatusField, map[string]any{
					"function":        modelName,
					"limits":          functionLimit,
					"cause":           cause,
					"userCode":        LicenseInfoData.UserCode,
					"contractAddress": ApplicationData.ContactAddress,
				}))
		}
	}()
	if !LicenseInfoData.Existed {
		cause = consts.LicenseStatusEmpty
		return
	}
	if LicenseInfoData.ExpireTime.IsZero() {
		cause = consts.LicenseStatusInvalid
		return
	}
	if !validLicense {
		cause = consts.LicenseStatusExpired
		return
	}
	cause = consts.LicenseStatusLimited
	return
}
