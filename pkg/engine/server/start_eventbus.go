package server

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/models"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/build"
	"fmt"
	"github.com/wundergraph/wundergraph/pkg/eventbus"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
)

func (s *EngineStart) Subscribe() {
	s.eventbusSubscribeOperation()
}

func (s *EngineStart) BreakData() {
	if !utils.GetBoolWithLockViper(consts.DevMode) || utils.InvokeFunctionLimit(consts.LicenseIncrementBuild) {
		return
	}

	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBreak, func(data any) any {
		s.printBreakStart(eventbus.ChannelOperation, data)
		return nil
	})
	eventbus.Subscribe(eventbus.ChannelStorage, eventbus.EventBreak, func(data any) any {
		s.printBreakStart(eventbus.ChannelStorage, data)
		return nil
	})
	eventbus.Subscribe(eventbus.ChannelSdk, eventbus.EventBreak, func(data any) any {
		return nil
	})
}

func (s *EngineStart) printBreakStart(channel eventbus.Channel, data any) {
	breakData := data.(*eventbus.BreakData)
	s.logger.Info(string(eventbus.EventBreak),
		zap.String(consts.EngineStatusField, consts.EngineStartSucceed),
		zap.String("cause", fmt.Sprintf("%s %s", channel, breakData.Event)),
		zap.Duration("cost", breakData.Cost))
}

func (s *EngineStart) printIncrementStart(event eventbus.Event, field zap.Field) {
	s.logger.Info(string(event), zap.String(consts.EngineStatusField, consts.EngineIncrementStart), field)
}

func (s *EngineStart) eventbusSubscribeOperation() {
	operationsConfig := build.GeneratedOperationsConfigRoot.FirstData()
	insertHandler := func(data any, printFunc func(eventbus.Event, zap.Field)) (next any) {
		operation := data.(*wgpb.Operation)
		switch operation.Engine {
		case wgpb.OperationExecutionEngine_ENGINE_FUNCTION,
			wgpb.OperationExecutionEngine_ENGINE_PROXY,
			wgpb.OperationExecutionEngine_ENGINE_GRAPHQL:
			next = operation
		}
		if next == nil {
			return
		}

		if printFunc != nil {
			printFunc(eventbus.EventInsert, zap.String(string(eventbus.ChannelOperation), operation.Path))
		}
		return
	}
	updateHandler := func(data any, printFunc func(eventbus.Event, zap.Field)) (next any) {
		operation := data.(*wgpb.Operation)
		defer func() {
			if next != nil && printFunc != nil {
				printFunc(eventbus.EventUpdate, zap.String(string(eventbus.ChannelOperation), operation.Path))
			}
		}()
		if operation.Name == "" {
			next = operation
			return
		}

		next = insertHandler(operation, nil)
		return next
	}
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventInsert, func(data any) any {
		return insertHandler(data, s.printIncrementStart)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchInsert, func(data any) any {
		var insertedPaths []string
		var inserted []*wgpb.Operation
		for _, operation := range data.([]*wgpb.Operation) {
			if insertedItem := insertHandler(operation, nil); insertedItem != nil {
				insertedPaths = append(insertedPaths, operation.Path)
				inserted = append(inserted, insertedItem.(*wgpb.Operation))
			}
		}
		if len(inserted) == 0 {
			return nil
		}

		s.printIncrementStart(eventbus.EventBatchInsert, zap.Strings(string(eventbus.ChannelOperation), insertedPaths))
		return inserted
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventDelete, func(data any) any {
		operationPath := data.(string)
		s.printIncrementStart(eventbus.EventDelete, zap.String(string(eventbus.ChannelOperation), operationPath))
		return operationPath
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchDelete, func(data any) any {
		operationPaths := data.([]string)
		s.printIncrementStart(eventbus.EventBatchDelete, zap.Strings(string(eventbus.ChannelOperation), operationPaths))
		return operationPaths
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventUpdate, func(data any) any {
		return updateHandler(data, s.printIncrementStart)
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventBatchUpdate, func(data any) any {
		var updatedPaths []string
		var updated []*wgpb.Operation
		for _, operation := range data.([]*wgpb.Operation) {
			if updatedItem := updateHandler(operation, nil); updatedItem != nil {
				updatedPaths = append(updatedPaths, operation.Path)
				updated = append(updated, updatedItem.(*wgpb.Operation))
			}
		}
		if len(updated) == 0 {
			return nil
		}

		s.printIncrementStart(eventbus.EventBatchUpdate, zap.Strings(string(eventbus.ChannelOperation), updatedPaths))
		return updated
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventInvalid, func(data any) any {
		operationPath := data.(string)
		operation, _ := models.OperationRoot.GetByDataName(operationPath)
		if operation == nil {
			return nil
		}

		operation.Invalid = true
		if !operation.Enabled {
			return nil
		}

		s.printIncrementStart(eventbus.EventInvalid, zap.String(string(eventbus.ChannelOperation), operationPath))
		return operationPath
	})
	eventbus.Subscribe(eventbus.ChannelOperation, eventbus.EventRuntime, func(data any) any {
		operation := data.(*wgpb.Operation)
		s.runtimeOperationItem(operationsConfig, operation)
		return nil
	})
}
