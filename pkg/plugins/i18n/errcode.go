// Package i18n
/*
 错误吗国际化配置，使用i18n-stringer实现
*/
package i18n

import (
	"path/filepath"
	"runtime"
	"strings"
)

// CustomError 自定义error结构体，并重写Error()方法，错误时返回自定义结构
type CustomError struct {
	Mode      string                `json:"mode"`    // 业务模块
	Code      Errcode               `json:"code"`    // 错误码
	Message   string                `json:"message"` // 错误消息
	I18nError *I18nErrcodeErrorWrap `json:"-"`
}

func (e *CustomError) Error() string {
	return e.I18nError.Error()
}

func (e *CustomError) ResetMessageWithLocale(locale string) {
	if locale == _Errcode_defaultLocale || _Errcode_isLocaleSupport(locale) {
		return
	}

	e.I18nError.locale = locale
	if custom, ok := e.I18nError.err.(*CustomError); ok {
		custom.ResetMessageWithLocale(locale)
	}
	e.Message = e.Error()
}

func SwitchErrcodeLocale(locale string) bool {
	result := _Errcode_isLocaleSupport(locale)
	if result {
		_Errcode_defaultLocale = locale
	}
	return result
}

// NewCustomErrorWithMode 新建自定义error实例化
func NewCustomErrorWithMode(mode string, err error, code Errcode, args ...any) error {
	i18nWrap := code.Wrap(err, _Errcode_defaultLocale, args...)
	customErr := &CustomError{
		Mode:      mode,
		Code:      code,
		Message:   i18nWrap.Error(),
		I18nError: i18nWrap,
	}
	return customErr
}

// NewCustomError 新建自定义error实例化
func NewCustomError(err error, code Errcode, args ...any) error {
	return NewCustomErrorWithMode(GetCallerMode(), err, code, args...)
}

func GetCallerMode() string {
	_, file, _, ok := runtime.Caller(2)
	if !ok {
		return ""
	}

	return strings.TrimRight(filepath.Base(file), filepath.Ext(file))
}

// First check
//go:generate $GOPATH/bin/i18n-stringer -type Errcode -tomlpath errcode -check

// Second generation
//go:generate $GOPATH/bin/i18n-stringer -type Errcode -tomlpath errcode -defaultlocale zh_cn

type Errcode uint16 //错误码

const (
	ServerError Errcode = 10101 + iota
)

const (
	ParamIllegalError Errcode = 10201 + iota
	ParamBindError
	StructParamEmtpyError
	BodyParamEmptyError
	PathParamEmptyError
	QueryParamEmptyError
	FormParamEmptyError
)

const (
	RequestResubmitError Errcode = 10301 + iota
	RequestSignatureError
	RequestReadBodyError
	RequestEmptyBodyError
	RequestProxyError
)

const (
	FileReadError Errcode = 10401 + iota
	FileWriteError
	FileZipError
	FileUnZipError
	FileZipAmountZeroError
	FileContentEmptyError
	DirectoryReadError
)

const (
	LoaderFileReadError Errcode = 10501 + iota
	LoaderFileNotExistError
	LoaderFileUnmarshalError
	LoaderDataExistEditorError
	LoaderDataExistError
	LoaderDataNotExistError
	LoaderLockNotFoundError
	LoaderWatcherNotSupport
	LoaderNoneModifiedError
	LoaderRWNotSupportError
	LoaderNameEmptyError
	LoaderBasenameEmptyErr
	LoaderRootOrExtensionEmptyErr
	LoaderMultipleOnlyError
	LoaderEmbedNotAllowModifyErr
	LoaderRemoveKeyNotFoundError
	LoaderRenameKeyNotFoundError
	LoaderRenameTargetExistError
	LoaderRenameNotAllowMultipleError
	LoaderWriteableRelyModelRequiredError
	LoaderDataFilepathError
)

const (
	VscodeOnlyDirectoriesCanWatchError Errcode = 10601 + iota
	VscodeDirectoryExistError
	VscodeFileExistError
	VscodeFileNotExistError
	VscodeSourceNotDirectoryError
	VscodeTargetDirectoryExistError
)

const (
	EngineCreateConfigError Errcode = 20101 + iota
	EngineRestartError      Errcode = 10101 + iota
)

const (
	DataInsertError Errcode = 20201 + iota
	DataDeleteError
	DataUpdateError
	DataSelectError
	DataCopyError
	DataRenameError
	DataBatchInsertError
	DataBatchDeleteError
	DataBatchUpdateError
	DataEmptyListError
	DataNotExistsError
)

const (
	DatasourceConnectionError Errcode = 20301 + iota
	DatasourceKindNotSupportedError
	DatasourceDisabledError
	DatasourceDatabaseUrlEmptyError
	DatabaseOasVersionError
	PrismaQueryError
	PrismaMigrateError
)

const (
	StoragePingError Errcode = 20401 + iota
	StorageDisabledError
	StorageMkdirError
	StorageTouchError
	StorageRemoveError
	StorageRenameError
	StorageListError
	StorageDetailError
	StorageDownloadError
)

const (
	OperationRoleHasBindError Errcode = 20501 + iota
	OperationRbacTypeError
)

const (
	SettingServerUrlEmptyError Errcode = 20601 + iota
)

const (
	SdkAlreadyUpToDateError Errcode = 20701 + iota
)
