// Package consts
/*
 引擎钩子常量
*/
package consts

type MiddlewareHook string

// operation/authentication/global钩子
const (
	PreResolve          MiddlewareHook = "preResolve"
	MutatingPreResolve  MiddlewareHook = "mutatingPreResolve"
	MockResolve         MiddlewareHook = "mockResolve"
	CustomResolve       MiddlewareHook = "customResolve"
	PostResolve         MiddlewareHook = "postResolve"
	MutatingPostResolve MiddlewareHook = "mutatingPostResolve"

	PostAuthentication         MiddlewareHook = "postAuthentication"
	MutatingPostAuthentication MiddlewareHook = "mutatingPostAuthentication"
	RevalidateAuthentication   MiddlewareHook = "revalidateAuthentication"
	PostLogout                 MiddlewareHook = "postLogout"

	HttpTransportBeforeRequest MiddlewareHook = "beforeOriginRequest"
	HttpTransportAfterResponse MiddlewareHook = "afterOriginResponse"
	HttpTransportOnRequest     MiddlewareHook = "onOriginRequest"
	HttpTransportOnResponse    MiddlewareHook = "onOriginResponse"

	WsTransportOnConnectionInit MiddlewareHook = "onConnectionInit"
)

type UploadHook string

// upload钩子
const (
	PreUpload  UploadHook = "preUpload"
	PostUpload UploadHook = "postUpload"
)
