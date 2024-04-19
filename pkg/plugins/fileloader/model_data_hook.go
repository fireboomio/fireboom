// Package fileloader
/*
 在数据模型上拓展钩子实现自定义函数处理
 引入eventbus实现所有数据变更通知，通过eventbus的notice和websocket实现
 示例：
 1. 实现新增时设置createTime，更新时设置updateTime
 2. afterMutate实现变更后触发引擎热重启
 3. after后置事件通知
*/
package fileloader

import "github.com/wundergraph/wundergraph/pkg/eventbus"

type DataHook[T any] struct {
	OnInsert func(*T) error
	OnUpdate func(*T, *T, string) error

	AfterInit   func(map[string]*T)
	AfterMutate func()

	AfterInsert func(*T, string) bool
	AfterUpdate func(*T, *DataModifies, string, ...string)
	AfterRename func(*T, *T, string)
	AfterDelete func(string, string)

	AfterBatchInsert func([]*T, string, ...[]byte) bool
	AfterBatchUpdate func([]*DataWithModifies[T], string)
	AfterBatchRename func([]*T, []*T, string)
	AfterBatchDelete func([]string, string)
}

func (p *Model[T]) onInsert(data *T) error {
	if p.DataHook == nil || p.DataHook.OnInsert == nil {
		return nil
	}

	return p.DataHook.OnInsert(data)
}

func (p *Model[T]) onUpdate(src, dst *T, user string) error {
	if p.DataHook == nil || p.DataHook.OnUpdate == nil {
		return nil
	}

	return p.DataHook.OnUpdate(src, dst, user)
}

func (p *Model[T]) afterInit(errs []error) {
	if len(errs) > 0 || len(p.dataCache) == 0 || p.DataHook == nil || p.DataHook.AfterInit == nil {
		return
	}

	p.DataHook.AfterInit(p.dataCache)
}

func (p *Model[T]) ignoreMutate(err error, user string) bool {
	return err != nil || user == SystemUser
}

func (p *Model[T]) afterMutate(err error, ignore bool) {
	if err != nil || p.DataHook == nil || p.DataHook.AfterMutate == nil || ignore {
		return
	}

	go p.DataHook.AfterMutate()
}

func (p *Model[T]) afterInsert(err error, data *T, user string) {
	var ignore bool
	defer func() {
		ignore = p.ignoreMutate(err, user) || ignore || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventInsert, data)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterInsert == nil {
		return
	}

	ignore = !p.DataHook.AfterInsert(data, user)
}

func (p *Model[T]) afterUpdate(err error, data *T, modify *DataModifies, user string, watchAction ...string) {
	defer func() {
		ignore := p.ignoreMutate(err, user) || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventUpdate, data)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterUpdate == nil {
		return
	}

	p.DataHook.AfterUpdate(data, modify, user, watchAction...)
}

func (p *Model[T]) afterRename(err error, src, dst *T, user string) {
	defer func() {
		ignore := p.ignoreMutate(err, user) || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventDelete, p.GetDataName(src)) &&
			eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventInsert, dst)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterRename == nil {
		return
	}

	p.DataHook.AfterRename(src, dst, user)
}

func (p *Model[T]) afterDelete(err error, dataName, user string) {
	defer func() {
		ignore := p.ignoreMutate(err, user) || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventDelete, dataName)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterDelete == nil {
		return
	}

	p.DataHook.AfterDelete(dataName, user)
}

func (p *Model[T]) afterBatchInsert(err error, datas []*T, user string, extraBytesArray [][]byte) {
	var ignore bool
	defer func() {
		ignore = p.ignoreMutate(err, user) || ignore || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventBatchInsert, datas)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterBatchInsert == nil {
		return
	}

	ignore = !p.DataHook.AfterBatchInsert(datas, user, extraBytesArray...)
}

func (p *Model[T]) afterBatchUpdate(err error, datas []*T, modifies []*DataModifies, user string) {
	if len(datas) == 0 {
		return
	}

	defer func() {
		ignore := p.ignoreMutate(err, user) || len(modifies) == 0 || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventBatchUpdate, datas)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterBatchUpdate == nil {
		return
	}

	var withModifies []*DataWithModifies[T]
	for i, item := range datas {
		withItem := &DataWithModifies[T]{
			Data:         item,
			DataModifies: modifies[i],
		}

		withModifies = append(withModifies, withItem)
	}
	p.DataHook.AfterBatchUpdate(withModifies, user)
}

func (p *Model[T]) afterBatchRename(err error, src, dst []*T, user string) {
	defer func() {
		var srcNames []string
		for _, srcItem := range src {
			srcNames = append(srcNames, p.GetDataName(srcItem))
		}
		ignore := p.ignoreMutate(err, user) || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventBatchDelete, srcNames) &&
			eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventBatchInsert, dst)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterBatchRename == nil {
		return
	}

	p.DataHook.AfterBatchRename(src, dst, user)
}

func (p *Model[T]) afterBatchDelete(err error, dataNames []string, user string) {
	defer func() {
		ignore := p.ignoreMutate(err, user) || eventbus.Publish(eventbus.Channel(p.modelName), eventbus.EventBatchDelete, dataNames)
		p.afterMutate(err, ignore)
	}()
	if err != nil || p.DataHook == nil || p.DataHook.AfterBatchDelete == nil {
		return
	}

	p.DataHook.AfterBatchDelete(dataNames, user)
}
