// Package fileloader
/*
 数据增量修改的实现
*/
package fileloader

import (
	"bytes"
	"fireboom-server/pkg/common/utils"
	"github.com/buger/jsonparser"
	json "github.com/json-iterator/go"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"strings"
)

const (
	Add       dataModifyName = "add"
	Remove    dataModifyName = "remove"
	Overwrite dataModifyName = "overwrite"
)

var (
	ignoreDataTypes      = []jsonparser.ValueType{jsonparser.Unknown}
	emptyDataTypes       = []jsonparser.ValueType{jsonparser.Null, jsonparser.NotExist}
	emptyValidateMap     map[jsonparser.ValueType]func([]byte, bool) bool
	emptyBytesNumber     = []byte("0")
	emptyBytesObject     = []byte("{}")
	emptyBytesArray      = []byte("[]")
	notEmptyBytesBoolean = []byte("true")
)

func init() {
	// 零值数据校验为空逻辑
	emptyValidateMap = make(map[jsonparser.ValueType]func([]byte, bool) bool)
	trueFunc := func([]byte, bool) bool { return true }
	emptyValidateMap[jsonparser.Null] = trueFunc
	emptyValidateMap[jsonparser.Unknown] = trueFunc
	emptyValidateMap[jsonparser.NotExist] = trueFunc

	emptyValidateMap[jsonparser.String] = func(p []byte, _ bool) bool { return len(p) == 0 }
	emptyValidateMap[jsonparser.Number] = func(p []byte, force bool) bool { return force && bytes.Equal(p, emptyBytesNumber) }
	emptyValidateMap[jsonparser.Object] = func(p []byte, _ bool) bool { return bytes.Equal(p, emptyBytesObject) }
	emptyValidateMap[jsonparser.Array] = func(p []byte, _ bool) bool { return bytes.Equal(p, emptyBytesArray) }
	emptyValidateMap[jsonparser.Boolean] = func(p []byte, force bool) bool { return force && !bytes.Equal(p, notEmptyBytesBoolean) }
}

type (
	DataWithModifies[T any] struct {
		Data *T
		*DataModifies
	}
	dataModifyName   string
	DataModifyDetail struct {
		Name       dataModifyName
		Target     []byte
		Origin     []byte
		TargetType jsonparser.ValueType
		OriginType jsonparser.ValueType
		Items      *DataModifies
	}
	DataModifies map[string]*DataModifyDetail
)

// 获取变更结果，根据path和modifyNames筛选
// path 监听的字段路径
// modifyNames 变更类型
func (d *DataModifies) getItemsModifies(path string, modifyNames ...dataModifyName) (result map[dataModifyName]*DataModifies) {
	itemModify := d
	pathArr := strings.Split(path, utils.StringDot)
	lastIndex := len(pathArr) - 1
	for i, item := range pathArr {
		if itemModify.NoneModified() {
			return
		}

		itemData, ok := (*itemModify)[item]
		if !ok || itemData.Items.NoneModified() {
			return
		}

		itemModify = itemData.Items
		if i < lastIndex {
			continue
		}

		result = make(map[dataModifyName]*DataModifies)
		for field, detail := range *itemData.Items {
			if !slices.Contains(modifyNames, detail.Name) {
				continue
			}

			detailGroup, ok := result[detail.Name]
			if !ok {
				detailGroup = &DataModifies{}
				result[detail.Name] = detailGroup
			}
			(*detailGroup)[field] = detail
		}
	}
	return
}

// GetModifyDetail 获取path路径上字段的变更详情
func (d *DataModifies) GetModifyDetail(path string) (detail *DataModifyDetail, ok bool) {
	if d.NoneModified() {
		return
	}

	itemModify := d
	dotLastIndex := strings.LastIndex(path, utils.StringDot)
	if dotLastIndex > -1 {
		modifies := maps.Values(itemModify.getItemsModifies(path[:dotLastIndex]))
		if len(modifies) == 0 {
			return
		}

		path = path[dotLastIndex+1:]
		itemModify = modifies[0]
	}

	detail, ok = (*itemModify)[path]
	return
}

// NoneModified 判断无变更
func (d *DataModifies) NoneModified() bool {
	return d == nil || len(*d) == 0
}

func (d *DataModifies) multipleModified() bool {
	return d != nil && len(*d) > 1
}

func (d *DataModifies) firstField() string {
	return maps.Keys(*d)[0]
}

// 增量修改数据实现
// 根据src和modify不同条件设置action类型
// object类型数据会递归修改，并挂载在父数据Items字段上(array类型数据不支持递归)
func (d *DataModifies) mergeBytes(srcBytes, modifyBytes []byte) (result []byte, err error) {
	result = srcBytes
	err = jsonparser.ObjectEach(modifyBytes, func(key []byte, modifyValue []byte, modifyDataType jsonparser.ValueType, _ int) (modifyErr error) {
		setKey := string(key)
		srcValue, srcDataType, _, _ := jsonparser.Get(srcBytes, setKey)
		if slices.Contains(ignoreDataTypes, srcDataType) {
			return
		}

		var itemsModifies *DataModifies
		var setAction dataModifyName
		var setValue []byte
		defer func() {
			if setAction == "" || modifyErr != nil {
				return
			}

			(*d)[setKey] = &DataModifyDetail{
				Name:       setAction,
				Target:     setValue,
				Origin:     srcValue,
				TargetType: modifyDataType,
				OriginType: srcDataType,
				Items:      itemsModifies,
			}
			switch setAction {
			case Remove:
				result = jsonparser.Delete(result, setKey)
			case Add, Overwrite:
				if modifyDataType == jsonparser.String {
					setValue = []byte(`"` + string(modifyValue) + `"`)
				}
				result, modifyErr = jsonparser.Set(result, setValue, setKey)
			}
		}()

		srcEmpty := slices.Contains(emptyDataTypes, srcDataType)
		modifyNull := modifyDataType == jsonparser.Null
		if srcEmpty && !modifyNull {
			setAction = Add
			setValue = modifyValue
			return
		}

		if modifyNull && !srcEmpty {
			setAction = Remove
			setValue = modifyValue
			return
		}

		if srcDataType != modifyDataType || bytes.Equal(srcValue, modifyValue) {
			return
		}

		if srcDataType == jsonparser.Object {
			nextModifies := &DataModifies{}
			setValue, modifyErr = nextModifies.mergeBytes(srcValue, modifyValue)
			if nextModifies.NoneModified() {
				return
			}

			itemsModifies = nextModifies
			setAction = Overwrite
			return
		}

		setAction = Overwrite
		setValue = modifyValue
		return
	})
	return
}

// 合并数据并返回合并结果
// 记录所有变更详情
func (p *Model[T]) mergeData(dataName string, srcData *T, modifyBytes []byte, user string, watchAction ...string) (dst *T, modify *DataModifies, err error) {
	srcBytes, err := json.Marshal(srcData)
	if err != nil {
		return
	}

	modify = &DataModifies{}
	afterMergeBytes, err := modify.mergeBytes(srcBytes, modifyBytes)
	if err != nil {
		return
	}

	if modify.NoneModified() {
		return
	}

	if rw, ok := p.DataRW.(*MultipleDataRW[T]); ok {
		afterMergeBytes, _ = p.overwriteIfEmpty(afterMergeBytes, rw.MergeDataBytes, true)
	}

	if err = json.Unmarshal(afterMergeBytes, &dst); err != nil {
		return
	}

	if err = p.onUpdate(srcData, dst, user); err != nil {
		return
	}

	if err = p.callWatchers(dataName, modify, watchAction...); err != nil {
		return
	}

	err = p.storeModel(dst)
	return
}

// 仅合并为空的数据
func (p *Model[T]) overwriteIfEmpty(srcBytes, modifyBytes []byte, force bool) (result []byte, modified bool) {
	result = srcBytes
	if modifyBytes == nil {
		return
	}

	_ = jsonparser.ObjectEach(modifyBytes, func(key []byte, modifyValue []byte, modifyDataType jsonparser.ValueType, _ int) (err error) {
		setKey := string(key)
		if emptyValidateMap[modifyDataType](modifyValue, true) {
			return
		}

		srcValue, srcDataType, _, _ := jsonparser.Get(srcBytes, setKey)
		setValue := modifyValue
		if srcDataType == jsonparser.Object {
			var objectModified bool
			if setValue, objectModified = p.overwriteIfEmpty(srcValue, modifyValue, force); objectModified {
				modified = true
				result, err = jsonparser.Set(result, setValue, setKey)
			}
			return
		}

		if !emptyValidateMap[srcDataType](srcValue, force) || bytes.Equal(srcValue, setValue) {
			return
		}

		if modifyDataType == jsonparser.String {
			setValue = []byte(`"` + string(modifyValue) + `"`)
		}

		modified = true
		result, err = jsonparser.Set(result, setValue, setKey)
		return
	})
	return
}
