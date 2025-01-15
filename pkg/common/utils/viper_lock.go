package utils

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"sync"
	"time"
)

var viperMutex = &sync.Mutex{}

// SetWithLockViper 带锁设置viper的值
func SetWithLockViper(key string, value interface{}) {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	viper.Set(key, value)
	_ = os.Setenv(key, fmt.Sprintf("%v", value))
}

// GetBoolWithLockViper 带锁获取viper中的bool值
func GetBoolWithLockViper(key string) bool {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	return viper.GetBool(key)
}

// GetStringWithLockViper 带锁获取viper中的string值
func GetStringWithLockViper(key string) string {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	return viper.GetString(key)
}

// GetTimeWithLockViper 带锁获取viper中的time值
func GetTimeWithLockViper(key string) time.Time {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	return viper.GetTime(key)
}

// GetInt32WithLockViper 带锁获取viper中的int32值
func GetInt32WithLockViper(key string) int32 {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	return viper.GetInt32(key)
}
