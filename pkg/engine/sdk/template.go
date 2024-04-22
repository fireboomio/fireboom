// Package sdk
/*
 sdk生成模板上下文，提供sdk模板的解析和生成
*/
package sdk

import (
	"crypto/md5"
	"fireboom-server/pkg/common/configs"
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/flowchartsman/handlebars/v3"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
)

const (
	templateFilesDirname    = "files"
	templatePartialsDirname = "partials"
	templateExtension       = ".hbs"
	ignoreFile              = ".fbignore"
)

func init() {
	utils.RegisterInitMethod(30, func() {
		build.AddAsyncGenerate(func() build.AsyncGenerate { return &templateContext{modelName: models.SdkRoot.GetModelName()} })
	})
}

type (
	templateContext struct {
		modelName string

		Webhooks []string
		Fields   []*wgpb.FieldConfiguration
		Types    []*wgpb.TypeField

		ApplicationHash    string // 唯一哈希
		EnableCSRFProtect  bool
		NodeEnvFilepath    string
		BaseURL            string
		InternalBaseURL    string
		ServerOptions      *wgpb.ServerOptions
		Roles              []string                      // 角色列表
		AuthProviders      []*wgpb.AuthProvider          // 认证配置
		S3Providers        []*wgpb.S3UploadConfiguration // S3上传配置
		HooksConfiguration *hooksConfiguration           // 钩子配置
		Operations         []*operationInfo

		MaxLengthMap     map[string]int
		EnumFieldArray   []*enumField   // 枚举类型定义
		ObjectFieldArray []*objectField // 对象类型定义
		TypeFormatArray  []string

		ServerEnumFieldArray   []*enumField   // 枚举类型定义
		ServerObjectFieldArray []*objectField // 对象类型定义
		ServerTypeFormatArray  []string

		OnceMap   map[string]any
		LengthMap map[string]any
	}
)

func (t *templateContext) Generate(builder *build.Builder) {
	enabledSdks := models.SdkRoot.ListByCondition(func(item *models.Sdk) bool { return item.Enabled })
	if len(enabledSdks) == 0 {
		return
	}

	t.Webhooks = make([]string, 0)
	t.EnableCSRFProtect = configs.GlobalSettingRoot.FirstData().EnableCSRFProtect
	t.BaseURL = utils.GetVariableString(builder.DefinedApi.NodeOptions.PublicNodeUrl)
	t.InternalBaseURL = utils.GetVariableString(builder.DefinedApi.NodeOptions.NodeUrl)
	t.ServerOptions = builder.DefinedApi.ServerOptions
	t.ApplicationHash = fmt.Sprintf("%x", md5.New().Sum(nil))[0:8]
	t.NodeEnvFilepath = utils.NormalizePath("..", configs.EnvEffectiveRoot.GetPath())

	t.MaxLengthMap = make(map[string]int, len(builder.DefinedApi.Operations)*3+1)
	t.S3Providers = builder.DefinedApi.S3UploadConfiguration
	t.AuthProviders = builder.DefinedApi.AuthenticationConfig.CookieBased.Providers
	t.HooksConfiguration = &hooksConfiguration{
		Global:        &globalHooksConfiguration{},
		Queries:       make(map[string][]consts.MiddlewareHook),
		Mutations:     make(map[string][]consts.MiddlewareHook),
		Subscriptions: make(map[string][]consts.MiddlewareHook),
	}

	for _, item := range builder.DefinedApi.EngineConfiguration.FieldConfigurations {
		if item.TypeName != consts.TypeQuery {
			continue
		}

		t.Fields = append(t.Fields, item)
	}

	for _, item := range builder.DefinedApi.EngineConfiguration.DatasourceConfigurations {
		t.Types = append(t.Types, item.ChildNodes...)
	}

	for _, item := range models.RoleRoot.List() {
		t.Roles = append(t.Roles, item.Code)
	}

	sdkRequiredConfig := &wgpb.WunderGraphConfiguration{
		Api: &wgpb.UserDefinedApi{
			NodeOptions:   builder.DefinedApi.NodeOptions,
			ServerOptions: builder.DefinedApi.ServerOptions,
		},
	}
	t.buildGlobalOperationHooks()
	t.buildAuthenticationHooks()
	t.buildFromDefinedApiSchema(builder.DefinedApi, sdkRequiredConfig.Api)
	t.clearEmptyGlobalOperationHooks()

	sdkRequiredConfigBytes, _ := json.MarshalIndent(&sdkRequiredConfig, "", "  ")
	for _, item := range enabledSdks {
		t.generateTemplate(item)
		if item.Type == models.SdkServer {
			_ = hookGraphqlConfigText.Write(hookGraphqlConfigText.Title, fileloader.SystemUser, sdkRequiredConfigBytes)
		}
	}
	if utils.EngineStarted() {
		debug.FreeOSMemory()
	}
}

func (t *templateContext) generateTemplate(sdk *models.Sdk) {
	outputPath := sdk.OutputPath
	if !sdk.Enabled || outputPath == "" {
		return
	}

	var err error
	defer func() {
		if err != nil {
			logger.Error("generate sdk template failed", zap.Error(err), zap.String(sdkModelName, sdk.Name))
		} else {
			logger.Debug("generate sdk template succeed", zap.String(sdkModelName, sdk.Name))
		}
	}()

	// 读取并加载片段函数
	partialsDirname := utils.NormalizePath(consts.RootTemplate, sdk.Name, templatePartialsDirname)
	var partials []string
	partialFiles, _ := os.ReadDir(partialsDirname)
	for _, item := range partialFiles {
		partials = append(partials, utils.NormalizePath(partialsDirname, item.Name()))
	}

	// 读取模板文件目录
	filesDirname := utils.NormalizePath(consts.RootTemplate, sdk.Name, templateFilesDirname)
	if utils.NotExistFile(filesDirname) {
		err = i18n.NewCustomErrorWithMode(t.modelName, nil, i18n.DirectoryReadError, filesDirname)
		return
	}

	t.OnceMap = make(map[string]any)
	ignoredFiles := getIgnoredFiles(utils.NormalizePath(outputPath, ignoreFile))
	existIgnoredFiles := len(ignoredFiles) > 0
	if existIgnoredFiles {
		logger.Debug("found ignored files", zap.String(t.modelName, sdk.Name), zap.Strings("files", ignoredFiles))
	}

	err = filepath.Walk(filesDirname, func(path string, info fs.FileInfo, _ error) (walkErr error) {
		if info == nil || info.IsDir() {
			return
		}

		relPath, _ := filepath.Rel(filesDirname, path)
		outputFilepath := utils.NormalizePath(outputPath, relPath)
		if !strings.HasSuffix(path, templateExtension) {
			// 在.fbignore中的忽略文件需要跳过
			if existIgnoredFiles && isIgnoredFile(relPath, ignoredFiles) {
				return
			}

			outputFile, _ := os.Stat(outputFilepath)
			// 文件存在且是.fbignore或模版文件时间小于生成文件时间则跳过
			if outputFile != nil && (relPath == ignoreFile || info.ModTime().Before(outputFile.ModTime())) {
				return
			}

			walkErr = utils.CopyFile(path, outputFilepath)
			return
		}

		fileBytes, walkErr := utils.ReadFile(path)
		if walkErr != nil {
			return
		}

		tpl, walkErr := handlebars.Parse(string(fileBytes))
		if walkErr != nil {
			return
		}

		if walkErr = tpl.RegisterPartialFiles(partials...); walkErr != nil {
			return
		}

		outputFilepath = strings.TrimSuffix(outputFilepath, templateExtension)
		prefix, _, _ := strings.Cut(info.Name(), utils.StringDot)
		// 如果使用了多文件生成定义则执行此逻辑
		if iterator, ok := multipleTemplateMap[prefix]; ok {
			iterator(t, tpl, outputFilepath)
			return
		}

		content, walkErr := tpl.Exec(t)
		if walkErr != nil {
			return
		}

		tpl = nil
		walkErr = utils.WriteFile(outputFilepath, []byte(content))
		return
	})
	return
}

// 获取忽略修改的文件，用于用户自定义跳过覆盖的文件
func getIgnoredFiles(ignoreFilePath string) (pathList []string) {
	pathList = make([]string, 0)
	ignoreFileBytes, _ := utils.ReadFile(ignoreFilePath)
	if len(ignoreFileBytes) == 0 {
		return
	}

	for _, line := range strings.Split(string(ignoreFileBytes), "\n") {
		if len(line) == 0 || slices.Contains(pathList, line) {
			continue
		}

		pathList = append(pathList, line)
	}
	return
}

func isIgnoredFile(path string, pathList []string) bool {
	for _, p := range pathList {
		if p == path {
			return true
		}

		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
	}
	return false
}

const (
	objectFieldArrayPrefix       = "${objectFieldArray}"
	enumFieldArrayPrefix         = "${enumFieldArray}"
	serverObjectFieldArrayPrefix = "${serverObjectFieldArray}"
	serverEnumFieldArrayPrefix   = "${serverEnumFieldArray}"
)

var (
	multipleTemplateMap    map[string]iteratorMultiple
	multipleFileNameRegexp = regexp.MustCompile(`<#fileName#>([^}]+)<#fileName#>`)
)

type iteratorMultiple func(*templateContext, *handlebars.Template, string)

// 多文件生成的实现，例如java中的class定义
// 匹配fileName特殊标识来获取生成的文件路径
func writeMultiples(tpl *handlebars.Template, object any, name string, outputFilepath, prefix string) {
	content, err := tpl.Exec(object)
	if err != nil {
		return
	}

	fileNameArray := multipleFileNameRegexp.FindStringSubmatch(content)
	if len(fileNameArray) < 2 {
		return
	}

	fileName := strings.TrimSpace(fileNameArray[1])
	content = strings.ReplaceAll(content, fileNameArray[0], name)
	outputFilepath = strings.ReplaceAll(outputFilepath, prefix, fileName)
	_ = utils.WriteFile(outputFilepath, []byte(content))
}

func init() {
	multipleTemplateMap = make(map[string]iteratorMultiple)
	multipleTemplateMap[objectFieldArrayPrefix] = func(tplCtx *templateContext, tpl *handlebars.Template, outputFilepath string) {
		for _, field := range tplCtx.ObjectFieldArray {
			writeMultiples(tpl, field, field.Name, outputFilepath, objectFieldArrayPrefix)
		}
	}
	multipleTemplateMap[serverObjectFieldArrayPrefix] = func(tplCtx *templateContext, tpl *handlebars.Template, outputFilepath string) {
		for _, field := range tplCtx.ServerObjectFieldArray {
			writeMultiples(tpl, field, field.Name, outputFilepath, serverObjectFieldArrayPrefix)
		}
	}
	multipleTemplateMap[enumFieldArrayPrefix] = func(tplCtx *templateContext, tpl *handlebars.Template, outputFilepath string) {
		for _, field := range tplCtx.EnumFieldArray {
			writeMultiples(tpl, field, field.Name, outputFilepath, enumFieldArrayPrefix)
		}
	}
	multipleTemplateMap[serverEnumFieldArrayPrefix] = func(tplCtx *templateContext, tpl *handlebars.Template, outputFilepath string) {
		for _, field := range tplCtx.ServerEnumFieldArray {
			writeMultiples(tpl, field, field.Name, outputFilepath, serverEnumFieldArrayPrefix)
		}
	}
}
