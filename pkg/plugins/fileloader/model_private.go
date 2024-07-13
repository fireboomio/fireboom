// Package fileloader
/*
 文件配置模型定义的私有方法，包括初始化函数，无锁操作，读取文件等
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/buger/jsonparser"
	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"os"
	"reflect"
	"strings"
	"sync"
)

type batchData[T any] struct {
	bytes       []byte
	data        *T
	dataName    string
	existed     bool
	existedData *T
}

func (p *Model[T]) initData(lazyLogger ...func() *zap.Logger) {
	p.logger = zap.L()
	p.setModelName()
	rootDirectories[p.modelName] = p.Root
	p.modelLock = &sync.Mutex{}
	p.dataCache = &utils.SyncMap[string, *T]{}

	p.removeWatchers = make(map[string][]func(string, ...string) error)
	p.renameWatchers = make(map[string][]func(string, string, ...string) error)

	errs := p.load()
	succeed := p.dataCache.Size()
	fields := []zap.Field{zap.String("model", p.modelName), zap.Int("succeed", succeed)}
	if len(errs) > 0 {
		fields = append(fields, zap.Int("failed", len(errs)))
		if !p.LoadErrorIgnored {
			p.loadErrored = succeed == 0
			fields = append(fields, zap.Errors("errors", errs))
		}
	}
	switch p.rwType {
	case multipleRW:
		fields = append(fields, zap.String("directory", p.Root))
	case singleRW:
		fields = append(fields, zap.String("path", p.GetPath()))
	}
	printFunc := func() {
		if p.loadErrored {
			p.logger.Error("Load file errored", fields...)
		} else {
			p.logger.Debug("Load file completed", fields...)
		}
	}
	if len(lazyLogger) > 0 {
		go func() {
			p.logger = lazyLogger[0]()
			printFunc()
		}()
	} else {
		printFunc()
	}

	p.afterInit(errs)
	return
}

func (p *Model[T]) ensureJsonBytes(dataBytes []byte) []byte {
	data, _ := p.unmarshal(dataBytes)
	dataBytes, _ = json.Marshal(data)
	return dataBytes
}

func (p *Model[T]) readFile(path string, readFunc func(string) ([]byte, error), initDataBytes []byte, ignoreMergeIfExisted bool) (data *T, err error) {
	dataBytes, err := readFunc(path)
	var isNotExist bool
	if err != nil {
		if !os.IsNotExist(err) {
			err = i18n.NewCustomErrorWithMode(p.modelName, err, i18n.LoaderFileReadError, path)
			return
		}

		isNotExist = true
	}

	if initDataBytes != nil {
		var inited, modified bool
		var modifiedBytes []byte
		if isNotExist {
			dataBytes = initDataBytes
			inited = true
		} else if !ignoreMergeIfExisted {
			modifiedBytes = p.ensureJsonBytes(dataBytes)
			if !gjson.ValidBytes(initDataBytes) {
				initDataBytes = p.ensureJsonBytes(initDataBytes)
			}
			modifiedBytes, modified = p.overwriteIfEmpty(modifiedBytes, initDataBytes, false)
		}

		if inited || modified {
			if modified {
				_ = json.Unmarshal(modifiedBytes, &data)
				dataBytes, _ = p.marshal(data)
			}

			if err = utils.WriteFile(path, dataBytes); err != nil {
				err = i18n.NewCustomErrorWithMode(p.modelName, err, i18n.FileWriteError, path)
				return
			}
		}
	} else if isNotExist {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderFileNotExistError, path)
		return
	}

	data, err = p.unmarshal(dataBytes)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(p.modelName, err, i18n.LoaderFileUnmarshalError, path)
		return
	}

	return
}

func (p *Model[T]) readToCache(path string, readFunc func(string) ([]byte, error), initDataBytes []byte, ignoreMergeIfExisted bool) (result []error) {
	data, err := p.readFile(path, readFunc, initDataBytes, ignoreMergeIfExisted)
	if err != nil {
		result = append(result, err)
		return
	}

	dataName := p.GetDataName(data)
	dataFilepath := p.GetPath(dataName)
	if dataFilepath != path {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderDataFilepathError, dataFilepath, path)
		result = append(result, err)
		return
	}

	p.addDataToCache(dataName, data)
	p.addEmptyDataLock(dataFilepath)
	return
}

func (p *Model[T]) sortLess(a, b *T) bool {
	return p.GetDataName(a) < p.GetDataName(b)
}

func (p *Model[T]) insertNotLock(data *T) (err error) {
	if err = p.ExistedDataNameThrowError(p.GetDataName(data)); err != nil {
		return err
	}

	if err = p.onInsert(data); err != nil {
		return
	}

	err = p.storeModel(data)
	return
}

func (p *Model[T]) deleteByDataNamesNotLock(checkExist bool, dataNames []string) (err error) {
	multiple, err := p.checkMustMultiple()
	if err != nil {
		return
	}

	if checkExist {
		if err = p.NotExistDataThrowError(dataNames...); err != nil {
			return
		}
	}

	for _, dataName := range dataNames {
		data, ok := p.getDataFromCache(dataName)
		if !ok {
			continue
		}

		if err = p.callRemoveAction(dataName); err != nil {
			return
		}

		if logic := multiple.LogicDelete; logic != nil {
			logic(data)
			if err = p.storeModel(data); err != nil {
				return
			}
		} else {
			if err = os.Remove(p.GetPath(dataName)); err != nil {
				return
			}

			p.removeDataFromCache(dataName)
			dataLockMap.Delete(p.GetPath(dataName))
		}
	}
	return
}

func (p *Model[T]) hasParentDataCondition(prefix string) func(data *T) bool {
	return func(data *T) bool {
		return strings.HasPrefix(p.GetDataName(data), prefix)
	}
}

func (p *Model[T]) filter(data *T) bool {
	multiple, err := p.checkMustMultiple()
	if err != nil {
		return true
	}

	filter := multiple.Filter
	return filter == nil || filter(data)
}

func (p *Model[T]) storeModel(data *T) (err error) {
	if p.rwType == embedRW {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderEmbedNotAllowModifyErr)
		return
	}

	dataName, err := p.EmptyDataNameThrowError(data)
	if err != nil {
		return
	}

	p.addDataToCache(dataName, data)
	path := p.GetPath(dataName)
	p.addEmptyDataLock(path)
	dataBytes, err := p.marshal(data)
	if err != nil {
		return
	}

	err = utils.WriteFile(path, dataBytes)
	return
}

func (p *Model[T]) addDataToCache(dataName string, data *T) {
	p.dataCache.Store(dataName, data)
}

func (p *Model[T]) removeDataFromCache(dataName string) {
	p.dataCache.Delete(dataName)
}

func (p *Model[T]) getDataFromCache(dataName string) (data *T, ok bool) {
	data, ok = p.dataCache.Load(dataName)
	ok = ok && p.filter(data)
	return
}

func (p *Model[T]) migrateModel(src, dst *T) (err error) {
	if p.rwType != multipleRW {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderMultipleOnlyError)
		return
	}

	if err = p.storeModel(dst); err != nil {
		return err
	}

	srcDataName, dstDataName := p.GetDataName(src), p.GetDataName(dst)
	if err = p.callRenameAction(srcDataName, dstDataName); err != nil {
		return err
	}

	srcPath, dstPath := p.GetPath(srcDataName), p.GetPath(dstDataName)
	p.removeDataFromCache(srcDataName)
	dataLockMap.Delete(srcPath)
	if strings.EqualFold(srcPath, dstPath) {
		return os.Rename(srcPath, dstPath)
	}
	return os.Remove(srcPath)
}

func (p *Model[T]) checkModifyByDataName(src, dst string) (srcModel, dstModel *T, err error) {
	multiple, err := p.checkMustMultiple()
	if err != nil {
		return
	}

	if err = p.ExistedDataNameThrowError(dst); err != nil {
		return
	}

	if srcModel, err = p.GetByDataName(src); err != nil {
		return
	}

	dstModel = p.cloneData(srcModel)
	multiple.SetDataName(dstModel, dst)
	return
}

func (p *Model[T]) cloneData(src *T) (dst *T) {
	copyModel := *src
	return &copyModel
}

func (p *Model[T]) listByParentDataName(src string) []*T {
	return p.ListByCondition(p.hasParentDataCondition(utils.AppendIfMissSlash(src)))
}

func (p *Model[T]) checkModifyByParentDataName(src, dst string) (srcModelArr, dstModelArr, repeatModelArr []*T, err error) {
	multiple, err := p.checkMustMultiple()
	if err != nil {
		return
	}

	srcs := p.listByParentDataName(src)
	repeatModelArr = make([]*T, 0)
	src, dst = utils.AppendIfMissSlash(src), utils.AppendIfMissSlash(dst)
	for _, srcItem := range srcs {
		srcModelArr = append(srcModelArr, srcItem)
		dstDataName := strings.ReplaceAll(p.GetDataName(srcItem), src, dst)
		dstItem := p.cloneData(srcItem)
		multiple.SetDataName(dstItem, dstDataName)
		dstModelArr = append(dstModelArr, dstItem)
		if p.ExistedDataName(dstDataName) {
			repeatModelArr = append(repeatModelArr, dstItem)
		}
	}
	return
}

func (p *Model[T]) setModelName() {
	var data *T
	typeName := reflect.TypeOf(data).String()
	typeNameIndex := strings.LastIndexByte(typeName, '.')
	if typeNameIndex == -1 {
		typeName = "any"
	} else {
		typeName = typeName[typeNameIndex+1:]
	}

	p.modelName = strings.ToLower(typeName[:1]) + typeName[1:]
}

func (p *Model[T]) extractBatchBytes(batchBodyBytes []byte) (datas []*batchData[T], err error) {
	_, _ = jsonparser.ArrayEach(batchBodyBytes, func(value []byte, dataType jsonparser.ValueType, _ int, _ error) {
		if dataType != jsonparser.Object || err != nil {
			return
		}

		var data *T
		if err = json.Unmarshal(value, &data); err != nil {
			return
		}

		dataName := p.GetDataName(data)
		itemData := &batchData[T]{bytes: value, data: data, dataName: dataName}
		itemData.existedData, itemData.existed = p.getDataFromCache(dataName)
		datas = append(datas, itemData)
	})
	return
}
