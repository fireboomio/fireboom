package utils

import (
	json "github.com/json-iterator/go"
	"sync"
)

type SyncMap[K comparable, V any] struct {
	sync.Map
}

func (s *SyncMap[K, V]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return nil, nil
	}

	return json.Marshal(s.ToMap())
}

func (s *SyncMap[K, V]) UnmarshalJSON(data []byte) error {
	dataMap := make(map[K]V)
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}

	*s = SyncMap[K, V]{}
	for k, v := range dataMap {
		s.Map.Store(k, v)
	}
	return nil
}

func (s *SyncMap[K, V]) Load(key K) (v V, ok bool) {
	data, ok := s.Map.Load(key)
	if !ok {
		return
	}

	v, ok = data.(V)
	return
}

func (s *SyncMap[K, V]) Contains(key K) (ok bool) {
	_, ok = s.Load(key)
	return
}

func (s *SyncMap[K, V]) LoadOrStore(key K, value V) (v V, ok bool) {
	data, existed := s.Map.LoadOrStore(key, value)
	v, ok = data.(V)
	ok = ok && existed
	return
}

func (s *SyncMap[K, V]) Store(key K, value V) {
	s.Map.Store(key, value)
}

func (s *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	if s == nil {
		return
	}
	s.Map.Range(func(key, value any) bool {
		keyStr, keyOk := key.(K)
		valObj, valOk := value.(V)
		if !keyOk || !valOk {
			return false
		}
		return f(keyStr, valObj)
	})
}

func (s *SyncMap[K, V]) ToMap() (dataMap map[K]V) {
	dataMap = make(map[K]V)
	s.Range(func(key K, value V) bool {
		dataMap[key] = value
		return true
	})
	return
}

func (s *SyncMap[K, V]) Keys() (keys []K) {
	keys = make([]K, 0)
	s.Range(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return
}

func (s *SyncMap[K, V]) Size() (count int) {
	s.Range(func(key K, _ V) bool {
		count++
		return true
	})
	return
}

func (s *SyncMap[K, V]) FirstValue() (v V) {
	s.Range(func(_ K, value V) bool {
		v = value
		return false
	})
	return
}

func (s *SyncMap[K, V]) IsEmpty() (b bool) {
	b = true
	s.Range(func(_ K, value V) bool {
		b = false
		return false
	})
	return
}

func (s *SyncMap[K, V]) Clear() (keys []K) {
	s.Range(func(key K, _ V) bool {
		s.Map.Delete(key)
		return true
	})
	return
}
