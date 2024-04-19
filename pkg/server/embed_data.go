// Package server
/*
 初始化内嵌数据
 包括默认角色和默认数据源的初始化
 其中数据源为rest数据源，内置模板文件
*/
package server

import (
	"embed"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	pluginEmbed "fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/swaggo/swag"
	"go.uber.org/zap"
	"io/fs"
	"strings"
)

const systemDatasource = "system"

func initEmbedModelDatas() (callbacks []func()) {
	initEmbedModel(pluginEmbed.RoleFs, pluginEmbed.RoleRoot, models.RoleRoot)
	initEmbedModel(pluginEmbed.DatasourceFs, pluginEmbed.DatasourceRoot, models.DatasourceRoot, func(item *models.Datasource) {
		if item.Name != systemDatasource {
			return
		}

		itemFilepath := models.GetDatasourceUploadFilepath(item)
		if itemFilepath == "" || !utils.NotExistFile(itemFilepath) {
			return
		}

		callbacks = append(callbacks, func() {
			dsModelName := models.DatasourceRoot.GetModelName()
			itemContent, err := swag.ReadDoc(cloudInstanceName)
			if err != nil {
				logger.Warn("readDoc from swagger failed", zap.Error(err), zap.String(dsModelName, item.Name))
				return
			}
			if err = utils.WriteFile(itemFilepath, []byte(itemContent)); err != nil {
				logger.Warn("store datasource failed", zap.Error(err), zap.String(dsModelName, item.Name))
			}
		})
	})
	return
}

func initEmbedModel[T any](embedFs embed.FS, root string, modelRoot *fileloader.Model[T], extraFunc ...func(*T)) {
	modelName := modelRoot.GetModelName()
	sources, err := fs.ReadDir(embedFs, root)
	if err != nil {
		logger.Warn("read embed failed", zap.String("model", modelName), zap.Error(err))
		return
	}

	extraExecute := func(item *T) {
		for _, extra := range extraFunc {
			extra(item)
		}
	}
	var itemBytes []byte
	var itemData *T
	for _, item := range sources {
		itemName := strings.TrimSuffix(item.Name(), string(fileloader.ExtJson))
		itemPath := utils.NormalizePath(root, item.Name())
		if itemBytes, err = fs.ReadFile(embedFs, itemPath); err != nil {
			logger.Warn("read embed failed", zap.Error(err), zap.String(modelName, itemName))
			continue
		}

		if itemData, err = modelRoot.GetByDataName(itemName); err == nil {
			extraExecute(itemData)
			continue
		}

		if itemData, err = modelRoot.Insert(itemBytes, fileloader.SystemUser); err != nil {
			logger.Warn("insert embed failed", zap.Error(err), zap.String(modelName, itemName))
			continue
		}

		extraExecute(itemData)
	}
	return
}
