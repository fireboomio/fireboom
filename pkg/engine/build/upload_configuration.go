// Package build
/*
 读取store/storage配置并转换成引擎所需的配置
*/
package build

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	json "github.com/json-iterator/go"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
)

func init() {
	utils.RegisterInitMethod(30, func() {
		addResolve(4, func() Resolve { return &uploadConfiguration{modelName: models.StorageRoot.GetModelName()} })
	})
}

type uploadConfiguration struct {
	modelName string
}

func (u *uploadConfiguration) Resolve(builder *Builder) (err error) {
	storages := models.StorageRoot.ListByCondition(func(item *models.Storage) bool { return item.Enabled })
	for _, storage := range storages {
		builder.DefinedApi.S3UploadConfiguration = append(builder.DefinedApi.S3UploadConfiguration, u.buildStorageItem(storage))
	}
	return
}

func (u *uploadConfiguration) buildStorageItem(storage *models.Storage) (config *wgpb.S3UploadConfiguration) {
	config = &wgpb.S3UploadConfiguration{
		Name:            storage.Name,
		Endpoint:        storage.Endpoint,
		AccessKeyID:     storage.AccessKeyID,
		SecretAccessKey: storage.SecretAccessKey,
		BucketName:      storage.BucketName,
		BucketLocation:  storage.BucketLocation,
		UseSSL:          storage.UseSSL,
		UploadProfiles:  make(map[string]*wgpb.S3UploadProfile),
	}
	for profileName, profile := range storage.UploadProfiles {
		copyProfile := &wgpb.S3UploadProfile{
			RequireAuthentication:     profile.RequireAuthentication,
			MaxAllowedUploadSizeBytes: profile.MaxAllowedUploadSizeBytes,
			MaxAllowedFiles:           profile.MaxAllowedFiles,
			AllowedMimeTypes:          profile.AllowedMimeTypes,
			AllowedFileExtensions:     profile.AllowedFileExtensions,
			MetadataJSONSchema:        profile.MetadataJSONSchema,
		}
		// 合成上传钩子配置
		hookConfigMap := make(map[string]bool)
		for hook, option := range models.GetStorageProfileHookOptions(storage.Name, profileName) {
			hookConfigMap[string(hook)] = option.Enabled && option.Existed
		}
		configBytes, _ := json.Marshal(hookConfigMap)
		_ = json.Unmarshal(configBytes, &copyProfile.Hooks)
		config.UploadProfiles[profileName] = copyProfile
	}
	logger.Debug("build storage succeed", zap.String(u.modelName, storage.Name))
	return
}
