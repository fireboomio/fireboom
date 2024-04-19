// Package fileloader
/*
 文本数据定义的私有方法，包括拷贝，删除，重命名等
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"os"
)

func (t *ModelText[T]) copy(srcDataName, dstDataName string) error {
	if t.emptyRootOrExt() {
		return nil
	}

	srcPath, err := t.path(srcDataName, 0)
	if err != nil {
		return err
	}

	if utils.NotExistFile(srcPath) {
		return nil
	}

	dstPath, err := t.path(dstDataName, 1)
	if err != nil {
		return err
	}

	return utils.CopyFile(srcPath, dstPath)
}

func (t *ModelText[T]) remove(dataName string, optional ...string) error {
	if t.emptyRootOrExt() {
		return nil
	}

	path, err := t.path(dataName, 0, optional...)
	if err != nil {
		return err
	}

	if utils.NotExistFile(path) {
		return nil
	}

	return os.RemoveAll(path)
}

func (t *ModelText[T]) rename(srcDataName, dstDataName string, optional ...string) error {
	if t.emptyRootOrExt() {
		return nil
	}

	srcPath, err := t.path(srcDataName, 0, optional...)
	if err != nil {
		return err
	}

	if utils.NotExistFile(srcPath) {
		return nil
	}

	dstPath, err := t.path(dstDataName, 1, optional...)
	if err != nil {
		return err
	}

	return renameFile(srcPath, dstPath)
}
