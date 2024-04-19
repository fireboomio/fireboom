// Package i18n
/*
 指令描述国际化配置，使用i18n-stringer实现
*/
package i18n

// First check
//go:generate $GOPATH/bin/i18n-stringer -type Directive -tomlpath directive -check

// Second generation
//go:generate $GOPATH/bin/i18n-stringer -type Directive -tomlpath directive -defaultlocale zh_cn

func SwitchDirectiveLocale(locale string) bool {
	result := _Directive_isLocaleSupport(locale)
	if result {
		_Directive_defaultLocale = locale
	}
	return result
}

type Directive uint16

const ExportDesc Directive = iota

const (
	FormatDateTimeDesc Directive = iota + 10101
	FormatDateTimeArgFormatDesc
	FormatDateTimeArgCustomFormatDesc
)

const (
	FromClaimDesc Directive = iota + 10201
	FromClaimArgNameDesc
	FromClaimArgCustomJsonPathDesc
	FromClaimArgRemoveIfNoneMatchDesc
)

const FromHeaderDesc Directive = iota + 10301

const InjectCurrentDateTimeDesc Directive = iota

const InjectEnvironmentVariableDesc Directive = iota

const InjectGeneratedUUIDDesc Directive = iota

const InternalDesc Directive = iota + 10401

const InternalOperationDesc Directive = iota + 10501

const (
	JsonschemaDesc Directive = iota + 10601
	JsonschemaArgMinimumDesc
	JsonschemaArgMaximumDesc
	JsonschemaArgMinItemsDesc
	JsonschemaArgMaxItemsDesc
	JsonschemaArgUniqueItemsDesc
	JsonschemaArgMaxLengthDesc
	JsonschemaArgMinLengthDesc
	JsonschemaArgPatternDesc
	JsonschemaArgCommonPatternDesc
)

const (
	RbacDesc Directive = iota + 10701
	RbacRequireMatchAnyDesc
	RbacRequireMatchAllDesc
	RbacDenyMatchAllDesc
	RbacDenyMatchAnyDesc
)

const (
	TransactionDesc Directive = iota + 10801
	TransactionArgMaxWaitSecondsDesc
	TransactionArgTimeoutSecondsDesc
	TransactionArgIsolationLevelDesc
)

const (
	TransformDesc Directive = iota + 10901
	TransformArgGetDesc
)

const (
	WhereInputDesc Directive = iota + 11001
	WhereInputFieldNotDesc
	WhereInputFieldFilterDesc
	WhereInputFilterFieldFieldDesc
	WhereInputFilterFieldScalarDesc
	WhereInputFilterFieldRelationDesc
	WhereInputFilterCommonFieldTypeDesc
	WhereInputScalarFilterFieldInsensitiveDesc
	WhereInputRelationFilterFieldWhereDesc
)

const InjectRuleValueDesc Directive = iota + 11101

const DisallowParallelDesc Directive = iota + 11201
const CustomizedFieldDesc Directive = iota + 11301
