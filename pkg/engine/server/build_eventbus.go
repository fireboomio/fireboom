package server

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/engine/build"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func (b *EngineBuild) Subscribe() {
	b.eventbusSubscribeStorage()
	b.eventbusSubscribeOperation()
}

func (b *EngineBuild) printIncrementBuild(event eventbus.Event, field zap.Field) {
	go func() { _ = b.emitGraphqlConfigCache() }()
	build.CallAsyncGenerates(b.builder)
	b.logger.Info(string(event), zap.String(consts.EngineStatusField, consts.EngineIncrementBuild), field)
}

func (b *EngineBuild) eventbusSubscribeOperation() {
	api := b.builder.DefinedApi
	insertHandler := func(data any, reload func(eventbus.Event, zap.Field)) any {
		operation := data.(*wgpb.Operation)
		api.Operations = append(api.Operations, operation)
		if reload != nil {
			reload(eventbus.EventInsert, zap.String(string(eventbus.ChannelOperation), operation.Path))
		}
		return operation
	}
	deleteHandler := func(data any, reload func(eventbus.Event, zap.Field)) any {
		operationPath := data.(string)
		operationIndex := slices.IndexFunc(api.Operations, func(item *wgpb.Operation) bool { return item.Path == operationPath })
		if operationIndex == -1 {
			return nil
		}

		api.Operations = slices.Delete(api.Operations, operationIndex, operationIndex+1)
		if reload != nil {
			reload(eventbus.EventDelete, zap.String(string(eventbus.ChannelOperation), operationPath))
		}
		return operationPath
	}
	updateHandler := func(data any, reload func(eventbus.Event, zap.Field)) any {
		operation := data.(*wgpb.Operation)
		deleteHandler(operation.Path, nil)
		if operation.Name != "" && insertHandler(operation, nil) == nil {
			return nil
		}

		if reload != nil {
			b.printIncrementBuild(eventbus.EventUpdate, zap.String(string(eventbus.ChannelOperation), operation.Path))
		}
		return operation
	}
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventInsert, func(data any) any {
		return insertHandler(data, b.printIncrementBuild)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchInsert, func(data any) any {
		var inserted []string
		for _, operation := range data.([]*wgpb.Operation) {
			insertHandler(operation, nil)
			inserted = append(inserted, operation.Path)
		}

		b.printIncrementBuild(eventbus.EventBatchInsert, zap.Strings(string(eventbus.ChannelOperation), inserted))
		return data
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventDelete, func(data any) any {
		return deleteHandler(data, b.printIncrementBuild)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchDelete, func(data any) any {
		var deleted []string
		for _, path := range data.([]string) {
			if deleteHandler(path, nil) != nil {
				deleted = append(deleted, path)
			}
		}
		if len(deleted) == 0 {
			return nil
		}

		b.printIncrementBuild(eventbus.EventBatchDelete, zap.Strings(string(eventbus.ChannelOperation), deleted))
		return data
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventUpdate, func(data any) any {
		return updateHandler(data, b.printIncrementBuild)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchUpdate, func(data any) any {
		var updatedPaths []string
		var updated []*wgpb.Operation
		for _, operation := range data.([]*wgpb.Operation) {
			if updateHandler(operation, nil) != nil {
				updatedPaths = append(updatedPaths, operation.Path)
				updated = append(updated, operation)
			}
		}
		if len(updated) == 0 {
			return nil
		}

		b.printIncrementBuild(eventbus.EventBatchUpdate, zap.Strings(string(eventbus.ChannelOperation), updatedPaths))
		return updated
	})
}

func (b *EngineBuild) eventbusSubscribeStorage() {
	api := b.builder.DefinedApi
	insertHandler := func(data any, reload func(eventbus.Event, zap.Field)) any {
		storage := data.(*wgpb.S3UploadConfiguration)
		api.S3UploadConfiguration = append(api.S3UploadConfiguration, storage)
		if reload != nil {
			reload(eventbus.EventInsert, zap.String(string(eventbus.ChannelStorage), storage.Name))
		}
		return storage
	}
	deleteHandler := func(data any, reload func(eventbus.Event, zap.Field)) any {
		storageName := data.(string)
		storageIndex := slices.IndexFunc(api.S3UploadConfiguration, func(item *wgpb.S3UploadConfiguration) bool { return item.Name == storageName })
		if storageIndex == -1 {
			return nil
		}

		api.S3UploadConfiguration = slices.Delete(api.S3UploadConfiguration, storageIndex, storageIndex+1)
		if reload != nil {
			reload(eventbus.EventDelete, zap.String(string(eventbus.ChannelStorage), storageName))
		}
		return storageName
	}
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventInsert, func(data any) any {
		return insertHandler(data, b.printIncrementBuild)
	})
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventDelete, func(data any) any {
		return deleteHandler(data, b.printIncrementBuild)
	})
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventUpdate, func(data any) any {
		storage := data.(*wgpb.S3UploadConfiguration)
		deleteResult := deleteHandler(storage.Name, nil)
		insertResult := insertHandler(storage, nil)
		if deleteResult == nil && insertResult == nil {
			return nil
		}

		b.printIncrementBuild(eventbus.EventUpdate, zap.String(string(eventbus.ChannelStorage), storage.Name))
		return storage
	})
}
