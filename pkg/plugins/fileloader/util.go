// Package fileloader
/*
 文件加载插件的私有工具集
*/
package fileloader

import (
	"fireboom-server/pkg/common/utils"
	"github.com/asaskevich/govalidator"
	"os"
	"path/filepath"
	"reflect"
)

func renameFile(srcPath, dstPath string) error {
	if err := utils.MkdirAll(filepath.Dir(dstPath)); err != nil {
		return err
	}

	return os.Rename(srcPath, dstPath)
}

func init() {
	// 注册自定义结构体字段校验逻辑
	govalidator.CustomTypeTagMap.Set("required_unless_ignored", func(i interface{}, o interface{}) bool {
		ignored := reflect.ValueOf(o).FieldByName("ExtensionIgnored").Bool()
		return ignored || !utils.IsZeroValue(i)
	})
	govalidator.CustomTypeTagMap.Set("required_if_multiple", func(i interface{}, o interface{}) bool {
		rwTypeUint := reflect.ValueOf(o).FieldByName("rwType").Uint()
		return rwTypeUint != uint64(multipleRW) || !utils.IsZeroValue(i)
	})
}

func filterIgnoreUnsupportedTypeError(err error) error {
	if err == nil {
		return nil
	}

	if _, match := err.(*govalidator.UnsupportedTypeError); match {
		return nil
	}

	errs, ok := err.(govalidator.Errors)
	if !ok {
		return err
	}

	for _, item := range errs {
		if err = filterIgnoreUnsupportedTypeError(item); err != nil {
			return err
		}
	}

	return nil
}
