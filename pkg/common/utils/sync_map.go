package utils

import "sync"

// LoadFromSyncMap 根据范型转换sync.Map的value
// 解决sync.Map返回的值没有具体类型的问题
func LoadFromSyncMap[T any](syncMap *sync.Map, key string) (v T, ok bool) {
	if syncMap == nil {
		return
	}

	data, ok := syncMap.Load(key)
	if !ok {
		return
	}

	v, ok = data.(T)
	return
}
