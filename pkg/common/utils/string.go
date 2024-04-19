package utils

import (
	"bytes"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

var src = rand.NewSource(time.Now().UnixNano())

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	// 6 bits to represent a letter index
	letterIdBits = 6
	// All 1-bits as many as letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
	StringDot    = "."
	StringComma  = ","
)

// RandStr 生成指定位数的随机字符串
func RandStr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(letters) {
			b[i] = letters[idx]
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return *(*string)(unsafe.Pointer(&b))
}

// JoinString 多个字符串拼接，允许使用动态参数
func JoinString(sep string, str ...string) string {
	return strings.Join(str, sep)
}

// FirstNotEmptyString 获取第一个非空字符串
func FirstNotEmptyString(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return ""
}

// UppercaseFirst 首字母大写
func UppercaseFirst(name string) string {
	if len(name) == 0 {
		return name
	}

	return strings.ToUpper(name[:1]) + name[1:]
}

// JoinStringWithDot 以','拼接字符串
func JoinStringWithDot(path ...string) string {
	return strings.Join(path, StringDot)
}

// MatchNameWithRegexp 匹配字符串，返回匹配的字符串和剩余的字符串
func MatchNameWithRegexp(str string, regexp *regexp.Regexp) (matched, cleared string) {
	if len(str) == 0 {
		return
	}

	cleared = str
	matches := regexp.FindStringSubmatch(str)
	if len(matches) <= 1 {
		return
	}

	matched = strings.TrimSpace(matches[1])
	cleared = strings.ReplaceAll(str, matches[0], "")
	return
}

var placeholderRegexp = regexp.MustCompile(`\${([^}]+)}`)

func ReplacePlaceholder(str string, replace func(string) string) string {
	return placeholderRegexp.ReplaceAllStringFunc(str, func(s string) string {
		before, after, ok := strings.Cut(s[2:len(s)-1], ":")
		if s = replace(before); s == "" && ok {
			s = after
		}
		return s
	})
}

func ReplacePlaceholderFromEnv(str string) string {
	return ReplacePlaceholder(str, func(s string) string { return GetStringWithLockViper(s) })
}

var normalNameRegexp = regexp.MustCompile("[^A-Za-z0-9_]+")

// NormalizeName 格式化命名，处理特殊字符
func NormalizeName(name string) string {
	name = normalNameRegexp.ReplaceAllString(name, "_")
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "_" + name
	}
	return name
}

// Camel2Case 驼峰转小写带指定字符
func Camel2Case(str string, sep string) string {
	buffer := bytes.Buffer{}
	for i, r := range str {
		if unicode.IsUpper(r) {
			if i != 0 {
				buffer.WriteString(sep)
			}
			buffer.WriteRune(unicode.ToLower(r))
		} else {
			buffer.WriteRune(r)
		}
	}
	return buffer.String()
}
