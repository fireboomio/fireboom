// Package sdk
/*
 注册handlerBars的helper函数，可以在编写sdk模板时使用
*/
package sdk

import (
	"fireboom-server/pkg/common/utils"
	"fmt"
	"github.com/flowchartsman/handlebars/v3"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"reflect"
	"regexp"
	"strings"
)

func init() {
	// 截取指定字符后的字符串
	handlebars.RegisterHelper("subStringAfter", func(str string, cut string) string {
		_, after, _ := strings.Cut(str, cut)
		return after
	})
	// 去除前置空格
	handlebars.RegisterHelper("trimPrefix", func(str string, cut string) string {
		cutLength := len(cut)
		for {
			if !strings.HasPrefix(str, cut) {
				break
			}

			str = str[cutLength:]
		}
		return str
	})
	// 判断是否唯一，可用于判断import等唯一性
	handlebars.RegisterHelper("isAbsent", func(onceMap map[string]any, name string, val any) bool {
		if utils.IsZeroValue(val) {
			return false
		}

		if _, ok := onceMap[name]; ok {
			return false
		}

		onceMap[name] = val
		return true
	})
	// 字符串长度
	handlebars.RegisterHelper("stringLen", func(name string) int {
		return len(name)
	})
	// 切片长度
	handlebars.RegisterHelper("sliceLen", func(name []string) int {
		return len(name)
	})
	// 根据最大长度填充空格，用于格式化代码
	handlebars.RegisterHelper("fillSpaces", func(maxLengthMap map[string]int, documentPath ...string) string {
		lastIndex := len(documentPath) - 1
		maxLength := maxLengthMap[utils.JoinStringWithDot(documentPath[:lastIndex]...)]
		fillCount := maxLength - len(documentPath[lastIndex])
		return strings.Repeat(" ", fillCount)
	})
	// 首字母小写
	handlebars.RegisterHelper("lowerFirst", func(str string) string {
		strLen := len(str)
		if strLen == 0 {
			return ""
		}

		result := strings.ToLower(str[:1])
		if strLen > 1 {
			result += str[1:]
		}
		return result
	})
	// 首字母大写
	handlebars.RegisterHelper("upperFirst", func(str string) string {
		strLen := len(str)
		if strLen == 0 {
			return ""
		}

		result := strings.ToUpper(str[:1])
		if strLen > 1 {
			result += str[1:]
		}
		return result
	})
	// 判断字符串是否在数组中
	handlebars.RegisterHelper("stringInArray", func(str string, strArr []string) bool {
		return slices.Contains(strArr, str)
	})
	// 以指定字符串连接数组
	handlebars.RegisterHelper("joinString", func(sep string, strArr []string) string {
		return strings.Join(strArr, sep)
	})
	// 打印日志，用于定位错误和输出对象
	handlebars.RegisterHelper("logger", func(val ...any) string {
		fmt.Printf("%#v\n", val)
		return ""
	})
	// 实现参数化format
	handlebars.RegisterHelper("fmtSprintf", func(format string, args ...any) string {
		return fmt.Sprintf(format, args...)
	})
	// 替换特殊字符
	handlebars.RegisterHelper("replaceSpecial", func(str, sep string) string {
		if strings.HasPrefix(str, "/") {
			str = str[1:]
		}

		reg := regexp.MustCompile("[^A-Za-z0-9_]+")
		return reg.ReplaceAllString(str, sep)
	})
	// 判断任意等于，target为','拼接的多个变量
	handlebars.RegisterHelper("equalAny", func(source string, target string) bool {
		return slices.Contains(strings.Split(target, utils.StringComma), source)
	})
	// 判断任意等于某一个值(暂未使用)
	handlebars.RegisterHelper("deepEqualAny", func(source any, target ...any) bool {
		return slices.ContainsFunc(target, func(e any) bool { return reflect.DeepEqual(source, e) })
	})
	// 判断是否零值
	handlebars.RegisterHelper("isNotEmpty", func(val any) bool {
		return !utils.IsZeroValue(val)
	})
	// bool值反转
	handlebars.RegisterHelper("invertBool", func(val bool) bool {
		return !val
	})
	// 判断字符串以指定字符开头
	handlebars.RegisterHelper("startWith", func(str string, prefix string) bool {
		return strings.HasPrefix(str, prefix)
	})
	// 全部满足条件
	handlebars.RegisterHelper("isAllTrue", func(val ...bool) bool {
		for _, b := range val {
			if !b {
				return false
			}
		}
		return true
	})
	// 任意满足条件
	handlebars.RegisterHelper("isAnyTrue", func(val ...bool) bool {
		for _, b := range val {
			if b {
				return true
			}
		}
		return false
	})
	// 获取wgpb.ConfigurationVariable的真实值
	handlebars.RegisterHelper("getVariableString", func(variable wgpb.ConfigurationVariable) string {
		return utils.GetVariableString(&variable)
	})
	// 获取值或默认值
	handlebars.RegisterHelper("getOrDefault", func(maps map[string]any, key, defaultValue any) any {
		if val, ok := maps[fmt.Sprintf("%v", key)]; ok {
			return val
		}

		return defaultValue
	})
}
