// Package fileloader
/*
 不同类型配置文件的读写
 EmbedDataRW 内嵌文件配置，如.env
 SingleDataRW 单个文件配置，如globalSetting
 MultipleDataRW 多个文件配置，如operation, role, datasource
 SingleDataRW和MultipleDataRW都可以使用EmbedDataRW作为默认数据
 MultipleDataRW 支持自定义过滤和逻辑删除
 EmbedDataRW 仅支持反序列化，SingleDataRW和MultipleDataRW支持正反序列化
*/
package fileloader

import (
	"embed"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	json "github.com/json-iterator/go"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

type rwType uint8

const (
	embedRW rwType = iota
	singleRW
	multipleRW
)

type (
	DataRW interface {
		dataRWType() rwType
	}
	EmbedDataRW[T any] struct {
		EmbedFiles *embed.FS `valid:"required"`
		DataName   string    `valid:"required"`
		Unmarshal  func([]byte) (*T, error)
	}
	SingleDataRW[T any] struct {
		InitDataBytes        []byte
		IgnoreMergeIfExisted bool
		DataName             string `valid:"required"`
		Marshal              func(*T) ([]byte, error)
		Unmarshal            func([]byte) (*T, error)
	}
	MultipleDataRW[T any] struct {
		MergeDataBytes []byte
		GetDataName    func(*T) string  `valid:"required"`
		SetDataName    func(*T, string) `valid:"required"`
		Filter         func(*T) bool
		LogicDelete    func(*T)
		Marshal        func(*T) ([]byte, error)
		Unmarshal      func([]byte) (*T, error)
	}
)

func (e *EmbedDataRW[T]) dataRWType() rwType {
	return embedRW
}

func (e *SingleDataRW[T]) dataRWType() rwType {
	return singleRW
}

func (e *MultipleDataRW[T]) dataRWType() rwType {
	return multipleRW
}

func (p *Model[T]) load() (result []error) {
	switch rw := p.DataRW.(type) {
	case *MultipleDataRW[T]:
		// 多个文件会递归目录加载数据
		_ = filepath.Walk(p.Root, func(path string, info fs.FileInfo, err error) error {
			if info == nil || info.IsDir() || !strings.HasSuffix(path, string(p.Extension)) {
				return nil
			}

			result = append(result, p.readToCache(filepath.ToSlash(path), utils.ReadFile, nil, false)...)
			return nil
		})
	case *SingleDataRW[T]:
		result = append(result, p.readToCache(p.GetPath(rw.DataName), utils.ReadFile, rw.InitDataBytes, rw.IgnoreMergeIfExisted)...)
	case *EmbedDataRW[T]:
		result = append(result, p.readToCache(p.GetPath(rw.DataName), rw.EmbedFiles.ReadFile, nil, false)...)
	}
	return
}

func (p *Model[T]) unmarshal(dataBytes []byte) (result *T, err error) {
	var unmarshalFunc func([]byte) (*T, error)
	switch rw := p.DataRW.(type) {
	case *MultipleDataRW[T]:
		unmarshalFunc = rw.Unmarshal
	case *SingleDataRW[T]:
		unmarshalFunc = rw.Unmarshal
	case *EmbedDataRW[T]:
		unmarshalFunc = rw.Unmarshal
	}

	if unmarshalFunc == nil {
		unmarshalFunc = func(bytes []byte) (data *T, err error) {
			err = json.Unmarshal(bytes, &data)
			return
		}
	}
	result, err = unmarshalFunc(dataBytes)
	return
}

func (p *Model[T]) marshal(data *T) (dataBytes []byte, err error) {
	var marshalFunc func(*T) ([]byte, error)
	switch rw := p.DataRW.(type) {
	case *MultipleDataRW[T]:
		marshalFunc = rw.Marshal
	case *SingleDataRW[T]:
		marshalFunc = rw.Marshal
	case *EmbedDataRW[T]:
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderEmbedNotAllowModifyErr)
		return
	}

	if marshalFunc == nil {
		marshalFunc = func(item *T) ([]byte, error) {
			return json.MarshalIndent(item, "", "  ")
		}
	}
	dataBytes, err = marshalFunc(data)
	return
}

func (p *Model[T]) checkMustMultiple() (rw *MultipleDataRW[T], err error) {
	rw, ok := p.DataRW.(*MultipleDataRW[T])
	if !ok {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderMultipleOnlyError)
		return
	}

	return
}

// GetDataName 根据类型获取数据名称
func (p *Model[T]) GetDataName(data *T) string {
	switch rw := p.DataRW.(type) {
	case *MultipleDataRW[T]:
		return rw.GetDataName(data)
	case *SingleDataRW[T]:
		return rw.DataName
	case *EmbedDataRW[T]:
		return rw.DataName
	}
	return ""
}

// GetPath 获取文件配置路径
// SingleDataRW和EmbedDataRW可以无参
func (p *Model[T]) GetPath(dataName ...string) string {
	switch rw := p.DataRW.(type) {
	case *MultipleDataRW[T]:
		if len(dataName) == 0 {
			return ""
		}
		return utils.NormalizePath(p.Root, dataName[0]+string(p.Extension))
	case *SingleDataRW[T]:
		return utils.NormalizePath(p.Root, rw.DataName+string(p.Extension))
	case *EmbedDataRW[T]:
		return utils.NormalizePath(p.Root, rw.DataName+string(p.Extension))
	}
	return ""
}

// GetPathRegexp 获取路径匹配正则，适用于导入数据时筛选文件
func (p *Model[T]) GetPathRegexp() (result *regexp.Regexp) {
	return regexp.MustCompile(fmt.Sprintf("^%s.*%s$", p.Root, p.Extension))
}
