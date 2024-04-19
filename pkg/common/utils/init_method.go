package utils

import (
	"fireboom-server/pkg/plugins/i18n"
	"sync"

	"golang.org/x/exp/slices"
)

var (
	initMethods []*initMethod
	initMutex   = &sync.Mutex{}
)

type initMethod struct {
	order  int
	method func()
	caller string
}

// RegisterInitMethod 注册初始化函数，使得原本不可控的init函数得以按顺序执行
// 编排系统启动时的初始化函数
func RegisterInitMethod(order int, method func()) {
	initMutex.Lock()
	defer initMutex.Unlock()

	initMethods = append(initMethods, &initMethod{
		order:  order,
		method: method,
		caller: i18n.GetCallerMode(),
	})
}

// ExecuteInitMethods 执行初始化函数，order优先，再按照caller排序
func ExecuteInitMethods() {
	slices.SortFunc(initMethods, func(a, b *initMethod) bool {
		return a.order < b.order || a.order == b.order && a.caller < b.caller
	})

	for _, item := range initMethods {
		item.method()
	}
}
