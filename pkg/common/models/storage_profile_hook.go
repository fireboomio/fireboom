// Package models
/*
 使用fileloader.ModelText管理钩子工作目录下代码文件
 读取钩子工作目录/storage下的代码文件
 当钩子配置变更时，自动切换父目录和重置路径字典
 与其他model不同的是，钩子配置允许多个且在一个文件中，通过WatcherPath监听uploadProfiles属性变更并实现自定义操作
 得益于fileloader中增量变更的实现，可以精确定位到实际变更的属性及其变更规则
*/
package models

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"sync"
)

var (
	StorageProfileHookOptionMap HookOptions
	storageProfileHookOnce      sync.Once
)

func GetStorageProfileHookOptions(dataName, profile string) HookOptions {
	return getHookOptionResultMap(StorageProfileHookOptionMap, dataName, profile)
}

func buildS3UploadProfileHook(hook consts.UploadHook, enabledFunc func(item *wgpb.S3UploadProfileHooksConfiguration) bool) {
	item := &fileloader.ModelText[Storage]{
		Title:              string(hook),
		RelyModelWatchPath: []string{"uploadProfiles"},
		TextRW: &fileloader.MultipleTextRW[Storage]{
			Enabled: func(item *Storage, optional ...string) bool {
				if !item.Enabled || len(optional) == 0 {
					return false
				}

				profile, ok := item.UploadProfiles[optional[0]]
				if !ok || profile.Hooks == nil {
					return false
				}

				return enabledFunc(profile.Hooks)
			},
			Name: func(dataName string, offset int, optional ...string) (basename string, ext bool) {
				if len(optional) < offset+1 {
					basename = dataName
					return
				}

				ext = true
				basename = utils.NormalizePath(dataName, optional[offset], string(hook))
				return
			},
		},
	}
	utils.RegisterInitMethod(20, func() {
		item.RelyModel = StorageRoot
		item.Init()
	})
	AddSwitchServerSdkWatcher(func(outputPath, codePackage, extension string, upperFirstBasename bool) {
		item.Extension = fileloader.Extension(extension)
		if outputPath != "" {
			if codePackage != "" {
				outputPath = utils.NormalizePath(outputPath, codePackage)
			}
			outputPath = utils.NormalizePath(outputPath, consts.HookStorageProfileParent)
		}
		item.Root = outputPath
		item.UpperFirstBasename = upperFirstBasename
		item.ResetRootDirectory()
		storageProfileHookOnce.Do(func() {
			StorageRoot.AddRenameAction(func(srcDataName, _ string) error {
				return removeSdkEmptyDir(outputPath, srcDataName)
			})
			StorageRoot.AddRemoveAction(func(dataName string) error {
				return removeSdkEmptyDir(outputPath, dataName)
			})
			StorageRoot.AddRenameWatcher(item.RelyModelWatchPath, func(srcDataName, _ string, optional ...string) error {
				return removeSdkEmptyDir(outputPath, utils.NormalizePath(srcDataName, optional[0]))
			})
			StorageRoot.AddRemoveWatcher(item.RelyModelWatchPath, func(dataName string, optional ...string) error {
				return removeSdkEmptyDir(outputPath, utils.NormalizePath(dataName, optional[0]))
			})
		})
	})
	StorageProfileHookOptionMap[consts.MiddlewareHook(hook)] = buildHookOption(item)
}

func init() {
	StorageProfileHookOptionMap = make(HookOptions)

	buildS3UploadProfileHook(consts.PreUpload, func(item *wgpb.S3UploadProfileHooksConfiguration) bool {
		return item.PreUpload
	})
	buildS3UploadProfileHook(consts.PostUpload, func(item *wgpb.S3UploadProfileHooksConfiguration) bool {
		return item.PostUpload
	})
}
