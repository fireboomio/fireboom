package utils

import (
	"path/filepath"
	"strings"
)

const ArrayPath = "[]"

// NormalizePath 格式化路径成linux系统路径
func NormalizePath(elem ...string) string {
	return filepath.ToSlash(filepath.Join(elem...))
}

// AppendIfMissSlash 追加路径分隔符（仅当缺失情况）
func AppendIfMissSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}

// CopyAndAppendItem 拷贝并追加字符，并判断是否添加'[]'
func CopyAndAppendItem(path []string, item string, appendArrayFunc ...func() bool) []string {
	pathLen := len(path)
	pathCap := pathLen + 1
	var appendRequired bool
	if len(appendArrayFunc) > 0 && appendArrayFunc[0]() {
		appendRequired = true
		pathCap++
	}
	itemPath := make([]string, pathLen, pathCap)
	copy(itemPath, path)
	itemPath = append(itemPath, item)
	if appendRequired {
		itemPath = append(itemPath, ArrayPath)
	}
	return itemPath
}
