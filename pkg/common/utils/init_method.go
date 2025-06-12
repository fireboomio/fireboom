package utils

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"os"
)

var initMethods SyncMap[*initMethod, bool]

type initMethod struct {
	order  int
	method func()
	caller string
}

// RegisterInitMethod 注册初始化函数，使得原本不可控的init函数得以按顺序执行
// 编排系统启动时的初始化函数
func RegisterInitMethod(order int, method func()) {
	initMethods.Store(&initMethod{
		order:  order,
		method: method,
		caller: i18n.GetCallerMode(),
	}, true)
}

// ExecuteInitMethods 执行初始化函数，order优先，再按照caller排序
func ExecuteInitMethods() {
	workdir := viper.GetString(consts.Workdir)
	if workdir != "" {
		if err := os.Chdir(workdir); err != nil {
			panic(err)
		}
		defer func() {
			zap.L().Info("Change working directory", zap.String("path", workdir))
		}()
	}
	inits := initMethods.Keys()
	slices.SortFunc(inits, func(a, b *initMethod) bool {
		return a.order < b.order || a.order == b.order && a.caller < b.caller
	})

	for _, item := range inits {
		item.method()
	}
}
