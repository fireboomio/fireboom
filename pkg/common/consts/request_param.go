// Package consts
/*
 请求参数常量
*/
package consts

const (
	PathParamDataName = "dataName"

	QueryParamDataNames      = "dataNames"
	QueryParamWatchAction    = "watchAction"
	QueryParamFilename       = "filename"
	QueryParamDirname        = "dirname"
	QueryParamUrl            = "url"
	QueryParamAuthentication = "auth-key"
	QueryParamCrud           = "crud"
	QueryParamOverwrite      = "overwrite"

	FormParamFile = "file"

	HeaderParamAuthentication = "X-FB-Authentication"
	HeaderParamLocale         = "X-FB-Locale"
	HeaderParamTag            = "X-FB-Tag"
	HeaderParamUser           = "X-FB-User"

	AttachmentFilenameFormat = `attachment;filename="%s"`
)
