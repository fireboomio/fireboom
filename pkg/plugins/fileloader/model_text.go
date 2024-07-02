// Package fileloader
/*
 文本数据(钩子代码文件)定义
 通过Root和Extension定义文本数据的路径匹配信息
 通过TextRW定义文本数据的读写类型
 通过主动注册事件实现数据变更的通知调用
 若有依赖项，文本数据变更时会判断依赖项数据是否加锁
 optional 添加额外参数获取文件路径，参考上传钩子文件依赖与storage的uploadProfile的key
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

type ModelText[T any] struct {
	Title                  string `json:"title" valid:"required_if_multiple"` // 标题
	Root                   string
	TextRW                 TextRW    `valid:"required"`
	Extension              Extension `valid:"required_unless_ignored"`
	UpperFirstBasename     bool      // 是否大写文件名(由sdk决定)
	ExtensionIgnored       bool
	ReadCacheRequired      bool      // 使用读缓存
	RelyModelActionIgnored bool      // 忽略操作变更函数的注册
	RelyModel              *Model[T] `valid:"required_if_multiple"`
	RelyModelWatchPath     []string  // 监听模型数据的字段路径

	rwType    rwType
	logger    *zap.Logger
	readCache *utils.SyncMap[string, *textCache]
}

// ResetRootDirectory 重置目录字典数据
// 当初始化和sdk变更时触发
func (t *ModelText[T]) ResetRootDirectory() {
	rootDirectories[t.Title] = t.Root
}

// GetFirstCache 获取文本数据缓存，目前仅适用于EmbedTextRW
func (t *ModelText[T]) GetFirstCache() (result []byte) {
	if value := t.readCache.FirstValue(); value != nil {
		result = value.content
	}
	return
}

func (t *ModelText[T]) Init() {
	if t.RelyModel != nil {
		t.RelyModel.textItems = append(t.RelyModel.textItems, t)
	}

	t.rwType = t.TextRW.textRWType()
	_, err := govalidator.ValidateStruct(t)
	if err = filterIgnoreUnsupportedTypeError(err); err != nil {
		panic(err)
	}

	if t.ReadCacheRequired || t.rwType == embedRW {
		t.readCache = &utils.SyncMap[string, *textCache]{}
	}
	t.logger = zap.L()
	t.load()
}

// Enabled 结合enabled和RelyModel中数据判断是有效
func (t *ModelText[T]) Enabled(dataName string, optional ...string) (ok bool) {
	enabledFunc := t.enabled()
	if enabledFunc == nil || t.RelyModel == nil {
		return
	}

	model, err := t.RelyModel.GetByDataName(dataName)
	if err != nil {
		return
	}

	ok = enabledFunc(model, optional...)
	return
}

// WriteCustom 自定义文本数据写入
func (t *ModelText[T]) WriteCustom(dataName, user string, writeFunc func(*os.File) error, optional ...string) (err error) {
	if err = t.checkAllowModify(); err != nil {
		return
	}

	if t.RelyModel == nil {
		return t.writeFile(dataName, writeFunc)
	}

	defer func() {
		data, _ := t.RelyModel.GetByDataName(dataName)
		if data == nil {
			return
		}
		t.RelyModel.afterUpdate(err, data, &DataModifies{}, user, optional...)
	}()
	writeAction := func(d *dataLock) error { return t.writeFile(dataName, writeFunc) }
	err = t.RelyModel.actionWithLock(t.RelyModel.GetPath(dataName), user, writeAction, t.RelyModel.existedEditorMatch(user))
	return
}

// Write 字节数据写入
func (t *ModelText[T]) Write(dataName, user string, content []byte, optional ...string) error {
	return t.WriteCustom(dataName, user, func(file *os.File) error {
		_, err := file.Write(content)
		return err
	}, optional...)
}

// Stat 文本文件状态
func (t *ModelText[T]) Stat(dataName string, optional ...string) (fileInfo os.FileInfo, err error) {
	path, err := t.path(dataName, 0, optional...)
	if err != nil {
		return
	}

	fileInfo, err = os.Stat(path)
	return
}

// GetModifiedTime 获取文本文件修改时间
func (t *ModelText[T]) GetModifiedTime(dataName string, optional ...string) (modTime time.Time, err error) {
	fileInfo, err := t.Stat(dataName, optional...)
	if err != nil {
		return
	}
	modTime = fileInfo.ModTime()
	return
}

// Read 文本数据读取
func (t *ModelText[T]) Read(dataName string, optional ...string) (content string, err error) {
	if err = t.checkAllowModify(); err != nil {
		return
	}

	path, err := t.path(dataName, 0, optional...)
	if err != nil {
		return
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(t.Title, err, i18n.LoaderFileReadError, path)
		return
	}

	modifiedTime := fileInfo.ModTime()
	if t.ReadCacheRequired {
		if cache, ok := t.readCache.Load(path); ok && modifiedTime.Equal(cache.lastModified) {
			content = string(cache.content)
			return
		}
	}

	contentBytes, err := t.TextRW.readFile(path)
	if err != nil {
		err = i18n.NewCustomErrorWithMode(t.Title, err, i18n.LoaderFileReadError, path)
		return
	}

	content = string(contentBytes)
	if t.ReadCacheRequired {
		t.readCache.Store(path, &textCache{content: contentBytes, lastModified: modifiedTime})
	}
	return
}

// ReadWithSkip 分页读取文件
func (t *ModelText[T]) ReadWithSkip(dataName string, skip, take int, optional ...string) ([]string, error) {
	return t.readWithCondition(dataName,
		func(line int, _ string) bool { return line <= skip },
		func(_ int, _ string) bool { take--; return take <= 0 },
		optional...)
}

// ReadWithSearch 带条件搜索读取文件
// 日志文件仅返回${prepareTime}后的日志内容
func (t *ModelText[T]) ReadWithSearch(dataName string, search string, optional ...string) ([]string, error) {
	matchStartLine := -1
	return t.readWithCondition(dataName,
		func(line int, text string) bool {
			if !strings.Contains(text, search) && matchStartLine == -1 {
				return true
			}

			if matchStartLine == -1 {
				matchStartLine = line
			}
			return false
		}, nil, optional...)
}

// Remove 删除文本数据文件(不建议直接使用)
// 建议通过事件注册方式由依赖项触发
func (t *ModelText[T]) Remove(dataName, user string, optional ...string) (err error) {
	if err = t.checkAllowModify(); err != nil {
		return
	}

	if t.RelyModel == nil {
		return t.remove(dataName, optional...)
	}

	removeAction := func(d *dataLock) error { return t.remove(dataName, optional...) }
	err = t.RelyModel.actionWithLock(t.RelyModel.GetPath(dataName), user, removeAction, t.RelyModel.existedEditorMatch(user))
	return
}

// 带条件读取文本数据
func (t *ModelText[T]) readWithCondition(dataName string, skipFunc, breakFunc func(int, string) bool, optional ...string) (content []string, err error) {
	if err = t.checkAllowModify(); err != nil {
		return
	}

	path, writeErr := t.path(dataName, 0, optional...)
	if writeErr != nil {
		return
	}

	content, err = utils.ReadWithCondition(path, skipFunc, breakFunc)
	return
}
