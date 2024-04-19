// Package fileloader
/*
 数据行锁实现，记录操作用户和最后访问时间
 锁${autoUnlockIntervalMinute}后会自动失效
 系统内置变更(user为${SystemUser})强制重置锁
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"go.uber.org/zap"
	"sync"
	"time"
)

type (
	dataLock struct {
		*sync.Mutex
		user       string    // 当前操作用户
		lastModify time.Time // 最后访问时间
	}
	DataWithLockUser[T any] struct {
		Data *T     `json:"data"`
		User string `json:"user"`
	}
	DataWithLockUser_data DataWithLockUser[any]
)

type (
	batchLockAction func() error
	lockAction      func(d *dataLock) error
	lockMatch       func(d *dataLock) (bool, error)
)

func (d *dataLock) actionWithLock(action lockAction, matches ...lockMatch) (err error) {
	d.Lock()
	defer d.Unlock()

	var ok bool
	for _, match := range matches {
		ok, err = match(d)
		if err != nil || !ok {
			return
		}
	}

	err = action(d)
	return
}

func (d *dataLock) reset() {
	d.user = ""
	d.lastModify = time.Time{}
}

func (d *dataLock) refresh(user string) {
	d.user = user
	d.lastModify = time.Now()
}

func (p *Model[T]) existedEditorMatch(user string) lockMatch {
	return func(d *dataLock) (ok bool, err error) {
		if d.user != "" && d.user != user {
			ok = false
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataExistEditorError)
			return
		}

		ok = true
		return
	}
}

func (p *Model[T]) addEmptyDataLock(key string) {
	dataLockMap.Store(key, &dataLock{Mutex: &sync.Mutex{}})
}

func (p *Model[T]) getLockByKey(key, user string) (dataLocker *dataLock, err error) {
	if user == SystemUser {
		p.addEmptyDataLock(key)
	}

	dataLocker, ok := utils.LoadFromSyncMap[*dataLock](dataLockMap, key)
	if !ok {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderLockNotFoundError, key)
		return
	}

	return
}

func (p *Model[T]) actionWithBatchLock(keys []string, user string, action batchLockAction) (err error) {
	batchLocks := make([]*dataLock, 0)
	var itemLock *dataLock
	for _, key := range keys {
		if itemLock, err = p.getLockByKey(key, user); err != nil {
			return err
		}

		if _, err = p.existedEditorMatch(user)(itemLock); err != nil {
			return err
		}

		itemLock.refresh(user)
		batchLocks = append(batchLocks, itemLock)
	}

	defer func() {
		for _, item := range batchLocks {
			item.reset()
		}
	}()

	err = action()
	return
}

func (p *Model[T]) actionWithLock(key, user string, action lockAction, matches ...lockMatch) (err error) {
	dataLocker, err := p.getLockByKey(key, user)
	if err != nil {
		return
	}

	err = dataLocker.actionWithLock(action, matches...)
	return
}

const (
	autoUnlockIntervalMinute = 5
	autoUnlockTickerSecond   = 10

	SystemUser = "$$system$$"
)

var dataLockMap *sync.Map

func init() {
	dataLockMap = &sync.Map{}
	go func() {
		lockTimeout := time.Minute * autoUnlockIntervalMinute * -1
		ticker := time.NewTicker(time.Second * autoUnlockTickerSecond)
		for {
			select {
			case c := <-ticker.C:
				c = c.Add(lockTimeout)
				dataLockMap.Range(func(key, value any) bool {
					lock := value.(*dataLock)
					autoResetAction := func(d *dataLock) error {
						d.reset()
						zap.S().Infof("%d分钟未操作[%s]自动解锁", autoUnlockIntervalMinute, key)
						return nil
					}
					autoResetMatch := func(d *dataLock) (ok bool, err error) {
						ok = lock.lastModify.After(c)
						return
					}
					_ = lock.actionWithLock(autoResetAction, autoResetMatch)
					return true
				})
			}
		}
	}()
}
