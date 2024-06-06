// Package i18n
/*
 查询引擎错误国际化配置，使用i18n-stringer实现
*/
package i18n

import (
	"github.com/spf13/cast"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
)

// First check
//go:generate $GOPATH/bin/i18n-stringer -type PrismaError -tomlpath prisma_error -check

// Second generation
//go:generate $GOPATH/bin/i18n-stringer -type PrismaError -tomlpath prisma_error -defaultlocale zh_cn

func init() {
	database.TranslateErrorFunc = func(code string) string {
		return PrismaError(cast.ToInt(code[1:])).String()
	}
}

func SwitchPrismaErrorLocale(locale string) bool {
	result := _PrismaError_isLocaleSupport(locale)
	if result {
		_PrismaError_defaultLocale = locale
	}
	return result
}

type PrismaError uint16

// CommonError
const (
	PrismaError_P1000 PrismaError = 1000 + iota
	PrismaError_P1001
	PrismaError_P1002
	PrismaError_P1003
)
const (
	PrismaError_P1008 PrismaError = 1008 + iota
	PrismaError_P1009
	PrismaError_P1010
	PrismaError_P1011
	PrismaError_P1012
	PrismaError_P1013
	PrismaError_P1014
	PrismaError_P1015
	PrismaError_P1016
	PrismaError_P1017
	PrismaError_P1018
	PrismaError_P1019
)

// PrismaQueryError
const (
	PrismaError_P2000 PrismaError = 2000 + iota
	PrismaError_P2001
	PrismaError_P2002
	PrismaError_P2003
	PrismaError_P2004
	PrismaError_P2005
	PrismaError_P2006
	PrismaError_P2007
	PrismaError_P2008
	PrismaError_P2009
	PrismaError_P2010
	PrismaError_P2011
	PrismaError_P2012
	PrismaError_P2013
	PrismaError_P2014
	PrismaError_P2015
	PrismaError_P2016
	PrismaError_P2017
	PrismaError_P2018
	PrismaError_P2019
	PrismaError_P2020
	PrismaError_P2021
	PrismaError_P2022
	PrismaError_P2023
	PrismaError_P2024
	PrismaError_P2025
	PrismaError_P2026
	PrismaError_P2027
	PrismaError_P2028
	PrismaError_P2029
	PrismaError_P2030
	PrismaError_P2031
)
const (
	PrismaError_P2033 PrismaError = 2033 + iota
	PrismaError_P2034
	PrismaError_P2035
	PrismaError_P2036
	PrismaError_P2037
)

// PrismaSchemaError
const (
	PrismaError_P3000 PrismaError = 3000 + iota
	PrismaError_P3001
	PrismaError_P3002
	PrismaError_P3003
	PrismaError_P3004
	PrismaError_P3005
	PrismaError_P3006
	PrismaError_P3007
	PrismaError_P3008
	PrismaError_P3009
	PrismaError_P3010
	PrismaError_P3011
	PrismaError_P3012
	PrismaError_P3013
	PrismaError_P3014
	PrismaError_P3015
	PrismaError_P3016
	PrismaError_P3017
	PrismaError_P3018
	PrismaError_P3019
	PrismaError_P3020
	PrismaError_P3021
	PrismaError_P3022
)
const (
	PrismaError_P4000 PrismaError = 4000 + iota
	PrismaError_P4001
	PrismaError_P4002
)
