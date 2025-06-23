// Package fileloader
/*
 数据变更监听实现，包括数据整体变更及监听某个字段值变更(map)
 注册函数均由依赖的字段主动调用
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
)

const (
	removeWatcher = "remove"
	renameWatcher = "rename"
)

func (p *Model[T]) addCopyAction(action func(string, string) error) {
	if p.rwType != multipleRW {
		return
	}

	p.copyActions = append(p.copyActions, action)
}

func (p *Model[T]) AddRemoveAction(action func(string) error) {
	if p.rwType != multipleRW {
		return
	}

	p.removeActions = append(p.removeActions, action)
}

func (p *Model[T]) AddRenameAction(action func(string, string) error) {
	if p.rwType != multipleRW {
		return
	}

	p.renameActions = append(p.renameActions, action)
}

func (p *Model[T]) callCopyAction(src, dst string) (err error) {
	if p.rwType != multipleRW {
		return
	}

	for _, action := range p.copyActions {
		if err = action(src, dst); err != nil {
			return err
		}
	}
	return
}

func (p *Model[T]) callRemoveAction(dataName string) (err error) {
	if p.rwType != multipleRW {
		return
	}

	for _, action := range p.removeActions {
		if err = action(dataName); err != nil {
			return err
		}
	}
	return
}

func (p *Model[T]) callRenameAction(src, dst string) (err error) {
	if p.rwType != multipleRW {
		return
	}

	for _, action := range p.renameActions {
		if err = action(src, dst); err != nil {
			return err
		}
	}
	return
}

func (p *Model[T]) AddRemoveWatcher(path []string, watcher func(string, ...string) error) {
	if p.rwType != multipleRW {
		return
	}

	watcherName := utils.JoinStringWithDot(path...)
	watchers := p.removeWatchers[watcherName]
	watchers = append(watchers, watcher)
	p.removeWatchers[watcherName] = watchers
}

func (p *Model[T]) AddRenameWatcher(path []string, watcher func(string, string, ...string) error) {
	if p.rwType != multipleRW {
		return
	}

	watcherName := utils.JoinStringWithDot(path...)
	watchers := p.renameWatchers[watcherName]
	watchers = append(watchers, watcher)
	p.renameWatchers[watcherName] = watchers
}

func (p *Model[T]) callWatchers(dataName string, modify *DataModifies, watchAction ...string) (err error) {
	if p.rwType != multipleRW || modify == nil || len(watchAction) == 0 {
		return
	}

	switch watchAction[0] {
	case removeWatcher:
		err = p.callRemoveWatchers(dataName, modify)
		return
	case renameWatcher:
		err = p.callRenameWatchers(dataName, modify)
		return
	}

	err = i18n.NewCustomErrorWithMode(watchAction[0], nil, i18n.LoaderWatcherNotSupport)
	return
}

func (p *Model[T]) callRemoveWatchers(dataName string, modify *DataModifies) (err error) {
	for watchPath, watchFuncs := range p.removeWatchers {
		watchItems := modify.getItemsModifies(watchPath, Remove)
		if watchItems[Remove].NoneModified() {
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderRemoveKeyNotFoundError)
			return
		}

		for _, watchFunc := range watchFuncs {
			for key := range *watchItems[Remove] {
				if err = watchFunc(dataName, key); err != nil {
					return
				}
			}
		}
	}
	return
}

func (p *Model[T]) callRenameWatchers(dataName string, modify *DataModifies) (err error) {
	for watchPath, watchFuncs := range p.renameWatchers {
		watchItems := modify.getItemsModifies(watchPath, Add, Remove, Overwrite)
		if overwrite := watchItems[Overwrite]; !overwrite.NoneModified() {
			existTarget := utils.JoinStringWithDot(watchPath, overwrite.firstField())
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderRenameTargetExistError, existTarget)
			return
		}

		if watchItems[Remove].NoneModified() || watchItems[Add].NoneModified() {
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderRenameKeyNotFoundError)
			return
		}

		if watchItems[Remove].multipleModified() || watchItems[Add].multipleModified() {
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderRenameNotAllowMultipleError)
			return
		}

		removeKey, addKey := watchItems[Remove].firstField(), watchItems[Add].firstField()
		for _, watchFunc := range watchFuncs {
			if err = watchFunc(dataName, dataName, removeKey, addKey); err != nil {
				return
			}
		}
	}
	return
}
