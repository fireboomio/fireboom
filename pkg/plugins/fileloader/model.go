// Package fileloader
/*
 数据模型定义
 通过Root和Extension定义文件配置的路径匹配信息
 通过DataRW定义文件配置的读写类型
 通过DataHook实现自定义钩子拓展实现
 通过copyActions, removeActions, renameActions实现依赖子项的事件通知变更
 通过removeWatchers, renameWatchers实现字段值变更监听
 使用表锁和行锁对数据共同管理
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	json "github.com/json-iterator/go"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"os"
	"strings"
	"sync"
)

type Model[T any] struct {
	Root                  string       `valid:"required"`                // 根目录，必填
	Extension             Extension    `valid:"required_unless_ignored"` // 文件扩展名
	ExtensionIgnored      bool         // 忽略扩展名
	LoadErrorIgnored      bool         // 忽略加载报错(文件不存在)
	DataRW                DataRW       `valid:"required"` // 模型类型，必填
	DataHook              *DataHook[T] // 操作钩子
	DataTreeExtra         func(*T) any // 返回目录树在节点上额外添加数据
	InsertBatchExtraField string       // 批量插入时额外拓展字段(不在模型中定义的字段)

	copyActions   []func(string, string) error // 模型拷贝触发的一系列的子项变更
	removeActions []func(string) error         // 模型删除触发的一系列的子项变更
	renameActions []func(string, string) error // 模型重命名触发的一系列的子项变更

	removeWatchers map[string][]func(string, ...string) error         // 监听模型字段值删除
	renameWatchers map[string][]func(string, string, ...string) error // 监听模型字段值重命名

	loadErrored bool
	once        sync.Once
	rwType      rwType                     // 类型
	logger      *zap.Logger                // 日志
	modelName   string                     // 模型名称，结构体的名称
	modelLock   *sync.Mutex                // 数据表锁
	dataCache   *utils.SyncMap[string, *T] // 数据缓存
	textItems   []*ModelText[T]            // 依赖子项
}

type (
	DataMutation struct {
		Src      string `json:"src"`
		Dst      string `json:"dst"`
		User     string `json:"-"`
		Overload bool   `json:"overload,omitempty"`
	}
	DataBatchResult struct {
		DataName string `json:"dataName"`
		Succeed  bool   `json:"succeed"`
	}
)

func (p *Model[T]) LoadErrored() bool {
	return p.loadErrored
}

func (p *Model[T]) Init(lazyLogger ...func() *zap.Logger) {
	p.once.Do(func() {
		p.rwType = p.DataRW.dataRWType()
		_, err := govalidator.ValidateStruct(p)
		if err = filterIgnoreUnsupportedTypeError(err); err != nil {
			panic(err)
		}

		p.initData(lazyLogger...)
	})
}

// ExistedDataName 根据名称判断数据存在
func (p *Model[T]) ExistedDataName(dataName string) bool {
	_, ok := p.getDataFromCache(dataName)
	return ok
}

// ExistedDataNameThrowError 根据名称判断数据存在并返回error
func (p *Model[T]) ExistedDataNameThrowError(dataName string) error {
	if p.ExistedDataName(dataName) {
		return i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataExistError, dataName)
	}
	return nil
}

// NotExistDataThrowError 根据名称(多个)判断数据不存在并返回error
func (p *Model[T]) NotExistDataThrowError(dataNames ...string) (err error) {
	for _, dataName := range dataNames {
		if !p.ExistedDataName(dataName) {
			err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataNotExistError, dataName)
			return
		}
	}

	return
}

// EmptyDataNameThrowError 获取名称，在为空时返回error
func (p *Model[T]) EmptyDataNameThrowError(data *T) (dataName string, err error) {
	if dataName = p.GetDataName(data); dataName == "" {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderNameEmptyError)
		return
	}

	return
}

// TryLockData 尝试对数据加锁，仅当无锁时可成功
func (p *Model[T]) TryLockData(dataName, user string) (err error) {
	if err = p.NotExistDataThrowError(dataName); err != nil {
		return
	}

	refreshAction := func(d *dataLock) error {
		d.refresh(user)
		return nil
	}
	err = p.actionWithLock(p.GetPath(dataName), user, refreshAction, p.existedEditorMatch(user))
	return
}

// ClearCache 清除缓存
func (p *Model[T]) ClearCache() {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	p.dataCache.Range(func(key string, value *T) bool {
		p.dataCache.Delete(key)
		dataLockMap.Delete(p.GetPath(key))
		_ = p.callRemoveAction(key)
		p.afterDelete(nil, key, SystemUser)
		return true
	})
}

// InsertOrUpdate 直接插入或更新，适用于系统配置文件的更新
func (p *Model[T]) InsertOrUpdate(data *T, opts ...func(*T)) (err error) {
	dataName := p.GetDataName(data)
	existedData, exist := p.getDataFromCache(dataName)
	if exist {
		err = p.onUpdate(existedData, data, SystemUser)
	} else {
		p.modelLock.Lock()
		defer p.modelLock.Unlock()
		err = p.onInsert(data)
	}
	if err != nil {
		return
	}

	for _, opt := range opts {
		opt(data)
	}
	err = p.storeModel(data)
	if exist {
		var dataModifies *DataModifies
		if p.LoadErrorIgnored {
			dataModifies = &DataModifies{}
		} else {
			dstBytes, _ := p.marshal(data)
			if _, dataModifies, _ = p.mergeData(dataName, existedData, dstBytes, SystemUser); dataModifies.NoneModified() {
				return
			}
		}
		p.afterUpdate(err, data, dataModifies, SystemUser)
	} else {
		p.afterInsert(err, data, SystemUser)
	}
	return
}

// Insert 新增数据
func (p *Model[T]) Insert(dataBytes []byte, user string) (data *T, err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	if err = json.Unmarshal(dataBytes, &data); err != nil {
		return
	}

	err = p.insertNotLock(data)
	p.afterInsert(err, data, user)
	return
}

// InsertBatch 批量新增数据
func (p *Model[T]) InsertBatch(batchBodyBytes []byte, user string, overwriteExisted bool) (batchResult []*DataBatchResult, err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	batchResult = make([]*DataBatchResult, 0)
	// 先抽取出[][]byte参数
	batchDatas, err := p.extractBatchBytes(batchBodyBytes)
	if err != nil {
		return
	}

	hasInsertBatchExtraField := p.InsertBatchExtraField != ""
	var datas []*T
	var extraBytesArray [][]byte
	for _, item := range batchDatas {
		if item.existed {
			if !overwriteExisted {
				continue
			}

			// 数据存在并且强制更新
			err = p.onUpdate(item.existedData, item.data, user)
		} else {
			err = p.onInsert(item.data)
		}
		if err != nil {
			continue
		}

		// 记录执行结果
		itemResult := &DataBatchResult{DataName: item.dataName}
		if err = p.storeModel(item.data); err == nil {
			itemResult.Succeed = true
			datas = append(datas, item.data)
		}
		batchResult = append(batchResult, itemResult)
		if hasInsertBatchExtraField {
			// 抽取额外存在的字段，并按顺序记录
			extraBytes, _, _, _ := jsonparser.Get(item.bytes, p.InsertBatchExtraField)
			extraBytesArray = append(extraBytesArray, extraBytes)
		}
	}
	p.afterBatchInsert(err, datas, user, extraBytesArray)
	return
}

// DeleteByDataName 根据名称删除数据
func (p *Model[T]) DeleteByDataName(dataName, user string) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	if err = p.NotExistDataThrowError(dataName); err != nil {
		return
	}

	deleteAction := func(d *dataLock) error {
		return p.deleteByDataNamesNotLock(true, []string{dataName})
	}

	// 带锁进行删除操作
	err = p.actionWithLock(p.GetPath(dataName), user, deleteAction, p.existedEditorMatch(user))
	p.afterDelete(err, dataName, user)
	return
}

// DeleteBatchByDataNames 根据名称(多个)批量删除数据
func (p *Model[T]) DeleteBatchByDataNames(dataNames []string, user string) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	if err = p.NotExistDataThrowError(dataNames...); err != nil {
		return
	}

	var keys []string
	for _, dataName := range dataNames {
		keys = append(keys, p.GetPath(dataName))
	}

	err = p.actionWithBatchLock(keys, user, func() error {
		return p.deleteByDataNamesNotLock(true, dataNames)
	})
	p.afterBatchDelete(err, dataNames, user)
	return
}

// UpdateByDataName 更新数据
// modifyBytes 增量更新数据
// watchAction 与removeWatchers和renameWatchers相对应，使用场景为storage的uploadProfiles
func (p *Model[T]) UpdateByDataName(modifyBytes []byte, user string, watchAction ...string) (dst *T, err error) {
	var modify *DataModifies
	if err = json.Unmarshal(modifyBytes, &dst); err != nil {
		return
	}

	dataName := p.GetDataName(dst)
	dst = nil
	if err = p.NotExistDataThrowError(dataName); err != nil {
		return
	}

	updateAction := func(d *dataLock) (actionErr error) {
		defer d.reset()

		srcData, actionErr := p.GetByDataName(dataName)
		if actionErr != nil {
			return actionErr
		}

		// 将更新的数据与已有数据进行合并
		dst, modify, actionErr = p.mergeData(dataName, srcData, modifyBytes, user, watchAction...)
		if actionErr != nil {
			return actionErr
		}

		// 合并结果发现无变更则返回error
		if modify.NoneModified() {
			actionErr = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderNoneModifiedError, dataName)
			return
		}

		return
	}

	err = p.actionWithLock(p.GetPath(dataName), user, updateAction, p.existedEditorMatch(user))
	p.afterUpdate(err, dst, modify, user, watchAction...)
	return
}

// UpdateBatch 批量更新数据
func (p *Model[T]) UpdateBatch(batchBodyBytes []byte, user string, watchAction ...string) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	batchDatas, err := p.extractBatchBytes(batchBodyBytes)
	if err != nil {
		return
	}

	var keys []string
	var srcDatas, dstDatas []*T
	for _, item := range batchDatas {
		if !item.existed {
			return i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataNotExistError, item.dataName)
		}

		// 先检查所有数据是否存在且数据均未加锁
		srcDatas = append(srcDatas, item.existedData)
		keys = append(keys, p.GetPath(item.dataName))
	}

	var modifies []*DataModifies
	err = p.actionWithBatchLock(keys, user, func() error {
		for i, item := range batchDatas {
			srcData := srcDatas[i]
			dst, modify, mergeErr := p.mergeData(p.GetDataName(srcData), srcData, item.bytes, user, watchAction...)
			if mergeErr != nil || modify.NoneModified() {
				continue
			}

			// 记录所有成功修改的数据
			modifies = append(modifies, modify)
			dstDatas = append(dstDatas, dst)
		}
		return nil
	})
	p.afterBatchUpdate(err, dstDatas, modifies, user)
	return
}

// FirstData 返回数据缓存中第一个数据，适用于EmbedDataRW和SingleDataRW
func (p *Model[T]) FirstData(condition ...func(*T) bool) (result *T) {
	list := p.ListByCondition(condition...)
	if len(list) == 0 {
		return
	}

	result = list[0]
	return
}

// GetByDataName 根据名称查询数据
func (p *Model[T]) GetByDataName(dataName string) (result *T, err error) {
	if data, ok := p.getDataFromCache(dataName); ok {
		result = data
	} else {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataNotExistError, dataName)
	}
	return
}

// GetWithLockUserByDataName 根据名称查询数据，返回结果带锁信息
func (p *Model[T]) GetWithLockUserByDataName(dataName string) (result *DataWithLockUser[T], err error) {
	data, err := p.GetByDataName(dataName)
	if err != nil {
		return
	}

	err = p.actionWithLock(p.GetPath(dataName), "", func(d *dataLock) error {
		result = &DataWithLockUser[T]{
			User: d.user,
			Data: data,
		}
		return nil
	})
	return
}

// List 返回数据列表，不带查询条件
func (p *Model[T]) List() (result []*T) {
	return p.ListByCondition()
}

// ListByDataNames 返回数据列表，根据名称查询
func (p *Model[T]) ListByDataNames(dataNames []string) (result []*T) {
	return p.ListByCondition(func(item *T) bool {
		return slices.ContainsFunc(dataNames, func(itemName string) bool {
			return p.GetDataName(item) == itemName
		})
	})
}

// ListByCondition 返回数据列表，自定义查询条件
func (p *Model[T]) ListByCondition(condition ...func(*T) bool) (result []*T) {
	result = make([]*T, 0)
	p.dataCache.Range(func(key string, data *T) bool {
		if !p.filter(data) {
			return true
		}

		if len(condition) > 0 {
			var match bool
			for _, itemCond := range condition {
				if match = itemCond(data); match {
					break
				}
			}
			if !match {
				return true
			}
		}

		result = append(result, data)
		return true
	})
	slices.SortFunc(result, p.sortLess)
	return
}

// CopyByDataName 拷贝数据并触发copyActions中依赖子项的拷贝操作
func (p *Model[T]) CopyByDataName(modify *DataMutation) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	_, dstModel, err := p.checkModifyByDataName(modify.Src, modify.Dst)
	if err != nil {
		return err
	}

	err = p.callCopyAction(modify.Src, modify.Dst)
	if err != nil {
		return err
	}

	err = p.insertNotLock(dstModel)
	p.afterInsert(err, dstModel, modify.User)
	return
}

// RenameByDataName 重命名数据并触发renameActions中依赖子项的拷贝操作
func (p *Model[T]) RenameByDataName(modify *DataMutation) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	fromModel, dstModel, err := p.checkModifyByDataName(modify.Src, modify.Dst)
	if err != nil {
		return
	}

	renameAction := func(d *dataLock) error {
		defer d.reset()
		if err := p.onUpdate(fromModel, dstModel, modify.User); err != nil {
			return err
		}

		return p.migrateModel(fromModel, dstModel)
	}

	err = p.actionWithLock(p.GetPath(modify.Src), modify.User, renameAction, p.existedEditorMatch(modify.User))
	p.afterRename(err, fromModel, dstModel, modify.User)
	return
}

// RenameByParentDataName 重命名数据父目录(重命名目录)
func (p *Model[T]) RenameByParentDataName(modify *DataMutation) (repeatModelArr []*T, err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	srcModelArr, dstModelArr, repeatModelArr, err := p.checkModifyByParentDataName(modify.Src, modify.Dst)
	if err != nil {
		return
	}

	if len(repeatModelArr) > 0 && !modify.Overload {
		return
	}

	var keys []string
	for _, item := range srcModelArr {
		keys = append(keys, p.GetPath(p.GetDataName(item)))
	}
	err = p.actionWithBatchLock(keys, modify.User, func() error {
		dstDir := utils.NormalizePath(p.Root, modify.Dst)
		_ = utils.MkdirAll(dstDir)
		for index, srcItem := range srcModelArr {
			dstItem := dstModelArr[index]
			if err := p.onUpdate(srcItem, dstItem, modify.User); err != nil {
				continue
			}

			_ = p.migrateModel(srcItem, dstItem)
		}
		srcDir := utils.NormalizePath(p.Root, modify.Src)
		if strings.EqualFold(srcDir, dstDir) {
			return os.Rename(srcDir, dstDir)
		}
		return os.RemoveAll(srcDir)
	})
	if len(srcModelArr) > 0 {
		p.afterBatchRename(err, srcModelArr, dstModelArr, modify.User)
	}
	return
}

// DeleteByParentDataName 根据父目录删除数据(删除目录)
func (p *Model[T]) DeleteByParentDataName(src, user string) (err error) {
	p.modelLock.Lock()
	defer p.modelLock.Unlock()

	srcs := p.listByParentDataName(src)
	var deleteDataNames, keys []string
	for _, srcItem := range srcs {
		srcDataName := p.GetDataName(srcItem)
		keys = append(keys, p.GetPath(srcDataName))
		deleteDataNames = append(deleteDataNames, srcDataName)
	}
	err = p.actionWithBatchLock(keys, user, func() (actionErr error) {
		actionErr = p.deleteByDataNamesNotLock(false, deleteDataNames)
		if actionErr != nil {
			return
		}

		actionErr = os.RemoveAll(utils.NormalizePath(p.Root, src))
		return
	})
	if len(deleteDataNames) > 0 {
		p.afterBatchDelete(err, deleteDataNames, user)
	}
	return
}

// GetModelName 获取模型名称
func (p *Model[T]) GetModelName() string {
	return p.modelName
}

// GetTextItems 获取依赖子项
func (p *Model[T]) GetTextItems() []*ModelText[T] {
	return p.textItems
}
