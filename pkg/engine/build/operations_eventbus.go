package build

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"strings"
)

func (o *operations) Subscribe() {
	reloadFunc := func() { _ = GeneratedOperationsConfigRoot.InsertOrUpdate(o.operationsConfigData) }
	insertHandler := func(data any, reload func()) any {
		operation := data.(*models.Operation)
		itemResult, succeed, invoked := o.buildOperationItem(operation)
		if itemResult == nil {
			eventbus.Publish(eventbus.ChannelOperation, eventbus.EventInvalid, operation.Path)
			return nil
		}

		eventbus.Publish(eventbus.ChannelOperation, eventbus.EventRuntime, itemResult)
		if invoked || !succeed {
			return nil
		}

		if reload != nil {
			reload()
		}
		return itemResult
	}
	deleteHandler := func(data any, reload func()) any {
		operationPath := data.(string)
		if !o.removeOperationItem(operationPath) {
			return nil
		}

		if reload != nil {
			reload()
		}
		return operationPath
	}
	updateHandler := func(data any, reload func()) any {
		operation := data.(*models.Operation)
		removeFailed := deleteHandler(operation.Path, nil) == nil
		if removeFailed && !operation.Invalid && operation.AuthorizationConfig != nil {
			return nil
		}

		data = insertHandler(operation, nil)
		if data == nil {
			if removeFailed {
				return nil
			}
			data = &wgpb.Operation{Path: operation.Path}
		}
		if reload != nil {
			reload()
		}
		return data
	}
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventInsert, func(data any) any {
		return insertHandler(data, reloadFunc)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchInsert, func(data any) any {
		var inserted []*wgpb.Operation
		for _, operation := range data.([]*models.Operation) {
			if result := insertHandler(operation, nil); result != nil {
				inserted = append(inserted, result.(*wgpb.Operation))
			}
		}
		if len(inserted) == 0 {
			return nil
		}

		reloadFunc()
		return inserted
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventDelete, func(data any) any {
		return deleteHandler(data, reloadFunc)
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

		reloadFunc()
		return deleted
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventUpdate, func(data any) any {
		return updateHandler(data, reloadFunc)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchUpdate, func(data any) any {
		var updated []*wgpb.Operation
		for _, operation := range data.([]*models.Operation) {
			if result := updateHandler(operation, nil); result != nil {
				updated = append(updated, result.(*wgpb.Operation))
			}
		}
		if len(updated) == 0 {
			return nil
		}

		reloadFunc()
		return updated
	})
}

func (o *operations) removeOperationItem(operationPath string) (removed bool) {
	switch {
	case strings.HasPrefix(operationPath, consts.HookProxyParent+"/"):
		if _, ok := o.operationsConfigData.ProxyOperationFiles[operationPath]; ok {
			removed = true
			delete(o.operationsConfigData.ProxyOperationFiles, operationPath)
		}
	case strings.HasPrefix(operationPath, consts.HookFunctionParent+"/"):
		if _, ok := o.operationsConfigData.FunctionOperationFiles[operationPath]; ok {
			removed = true
			delete(o.operationsConfigData.FunctionOperationFiles, operationPath)
		}
	default:
		if _, ok := o.operationsConfigData.GraphqlOperationFiles[operationPath]; ok {
			removed = true
			delete(o.operationsConfigData.GraphqlOperationFiles, operationPath)
		}
	}

	return
}
