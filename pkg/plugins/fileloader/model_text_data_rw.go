// Package fileloader
/*
 不同类型文本数据的读写
 EmbedTextRW 内嵌文本数据，如banner
 SingleTextRW 单个文本数据，如全局钩子
 MultipleTextRW 多个文件配置，如operation钩子，上传钩子
*/
package fileloader

import (
	"embed"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type (
	TextRW interface {
		textRWType() rwType
		readFile(string) ([]byte, error)
	}
	EmbedTextRW struct {
		EmbedFiles *embed.FS `valid:"required"`
		Name       string    `valid:"required"`
	}
	SingleTextRW[T any] struct {
		Enabled func(*T, ...string) bool
		Name    string `valid:"required"`
	}
	MultipleTextRW[T any] struct {
		Enabled func(*T, ...string) bool
		Name    func(string, int, ...string) (string, bool) `valid:"required"`
	}
	textCache struct {
		content      []byte
		lastModified time.Time
	}
)

func (e *EmbedTextRW) textRWType() rwType {
	return embedRW
}

func (e *EmbedTextRW) readFile(path string) ([]byte, error) {
	return e.EmbedFiles.ReadFile(path)
}

func (e *SingleTextRW[T]) textRWType() rwType {
	return singleRW
}

func (e *SingleTextRW[T]) readFile(path string) ([]byte, error) {
	return utils.ReadFile(path)
}

func (e *MultipleTextRW[T]) textRWType() rwType {
	return multipleRW
}

func (e *MultipleTextRW[T]) readFile(path string) ([]byte, error) {
	return utils.ReadFile(path)
}

func DefaultBasenameFunc(elem ...string) func(string, int, ...string) (string, bool) {
	return func(dataName string, _ int, _ ...string) (string, bool) {
		path := []string{dataName}
		path = append(path, elem...)
		return utils.NormalizePath(path...), true
	}
}

func (t *ModelText[T]) load() {
	switch rw := t.TextRW.(type) {
	case *MultipleTextRW[T]:
		p := t.RelyModel
		if !t.RelyModelActionIgnored {
			// 注册变更事件通知
			p.addCopyAction(t.copy)
			p.addRenameAction(func(src, dst string) error {
				return t.rename(src, dst)
			})
			p.addRemoveAction(func(dataName string) error {
				return t.remove(dataName)
			})
		}
		if len(t.RelyModelWatchPath) > 0 {
			// 注册监听字段变更事件通知
			p.addRemoveWatcher(t.RelyModelWatchPath, t.remove)
			p.addRenameWatcher(t.RelyModelWatchPath, t.rename)
		}
	case *SingleTextRW[T]:
		t.Title = rw.Name
	case *EmbedTextRW:
		t.Title = rw.Name
		path := utils.NormalizePath(t.Root, rw.Name+string(t.Extension))
		embedBytes, _ := rw.EmbedFiles.ReadFile(path)
		t.readCache[path] = &textCache{content: embedBytes}
	}
	return
}

func (t *ModelText[T]) enabled() func(*T, ...string) bool {
	switch rw := t.TextRW.(type) {
	case *MultipleTextRW[T]:
		return rw.Enabled
	case *SingleTextRW[T]:
		return rw.Enabled
	case *EmbedTextRW:
		return nil
	}
	return nil
}

func (t *ModelText[T]) path(dataName string, offset int, optional ...string) (string, error) {
	switch rw := t.TextRW.(type) {
	case *MultipleTextRW[T]:
		// 根据函数Name获取文件名
		basename, appendExtension := rw.Name(dataName, offset, optional...)
		if basename == "" {
			return "", i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderBasenameEmptyErr)
		}

		if appendExtension {
			basename += string(t.Extension)
		}

		return utils.NormalizePath(t.Root, basename), nil
	case *SingleTextRW[T]:
		// 名称等于rw.name获取在依赖项缓存数据中存在
		if dataName != rw.Name && (t.RelyModel == nil || t.RelyModel.dataCache[dataName] == nil) {
			return "", i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderDataFilepathError, rw.Name, dataName)
		}
		return utils.NormalizePath(t.Root, rw.Name+string(t.Extension)), nil
	case *EmbedTextRW:
		if dataName != rw.Name {
			return "", i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderDataFilepathError, rw.Name, dataName)
		}
		return utils.NormalizePath(t.Root, rw.Name+string(t.Extension)), nil
	}
	return "", i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderRWNotSupportError, t.TextRW)
}

// 目录或扩展名为空或内嵌数据时禁止修改
func (t *ModelText[T]) checkAllowModify() (err error) {
	if t.emptyRootOrExt() {
		err = i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderRootOrExtensionEmptyErr)
		return
	}

	if t.rwType == embedRW {
		err = i18n.NewCustomErrorWithMode(t.Title, nil, i18n.LoaderEmbedNotAllowModifyErr)
		return
	}

	return
}

func (t *ModelText[T]) writeFile(dataName string, writeFunc func(*os.File) error, optional ...string) error {
	path, actionErr := t.path(dataName, 0, optional...)
	if actionErr != nil {
		return actionErr
	}

	if t.ReadCacheRequired {
		t.readCacheMutex.Lock()
		defer t.readCacheMutex.Unlock()

		delete(t.readCache, path)
	}

	file, actionErr := utils.CreateFile(path)
	if actionErr != nil {
		return actionErr
	}

	return writeFunc(file)
}

func (t *ModelText[T]) emptyRootOrExt() bool {
	return t.Root == "" || !t.ExtensionIgnored && t.Extension == ""
}

// GetPath 获取文件配置路径
func (t *ModelText[T]) GetPath(dataName string, optional ...string) string {
	path, _ := t.path(dataName, 0, optional...)
	if len(path) == 0 || !t.UpperFirstBasename {
		return path
	}

	return utils.NormalizePath(filepath.Dir(path), utils.UppercaseFirst(filepath.Base(path)))
}

// GetPathRegexp 获取路径匹配正则，适用于导入数据时筛选文件
func (t *ModelText[T]) GetPathRegexp() (result *regexp.Regexp) {
	if t.emptyRootOrExt() {
		return
	}

	return regexp.MustCompile(fmt.Sprintf("^%s.*%s$", t.Root, t.Extension))
}
