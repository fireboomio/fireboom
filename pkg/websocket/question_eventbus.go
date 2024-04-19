package websocket

import (
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
)

func init() {
	utils.RegisterInitMethod(40, func() {
		eventbusSubscribeModel(models.OperationRoot)
		eventbusSubscribeModel(models.StorageRoot)
		eventbusSubscribeModel(models.SdkRoot)
	})
}

func eventbusSubscribeModel[T any](model *fileloader.Model[T]) {
	modelName := model.GetModelName()
	modelChannel := eventbus.Channel(modelName)
	eventbus.Subscribe(modelChannel, eventbus.EventInsert, func(data any) any {
		removeQuestion(modelName, model.GetDataName(data.(*T)))
		return data
	})
	eventbus.Subscribe(modelChannel, eventbus.EventBatchInsert, func(data any) any {
		for _, item := range data.([]*T) {
			removeQuestion(modelName, model.GetDataName(item))
		}
		return data
	})
	eventbus.Subscribe(modelChannel, eventbus.EventDelete, func(data any) any {
		removeQuestion(modelName, data.(string))
		return data
	})
	eventbus.Subscribe(modelChannel, eventbus.EventBatchDelete, func(data any) any {
		for _, item := range data.([]string) {
			removeQuestion(modelName, item)
		}
		return data
	})
	eventbus.Subscribe(modelChannel, eventbus.EventUpdate, func(data any) any {
		return eventbus.DirectCall(modelChannel, eventbus.EventInsert, data)
	})
	eventbus.Subscribe(modelChannel, eventbus.EventBatchUpdate, func(data any) any {
		return eventbus.DirectCall(modelChannel, eventbus.EventBatchInsert, data)
	})
}

func removeQuestion(model, name string) {
	questionMutex.Lock()
	defer questionMutex.Unlock()

	var qs []*question
	for _, item := range questions {
		if item.Model == model && item.Name == name {
			continue
		}

		qs = append(qs, item)
	}
	questions = qs
}
