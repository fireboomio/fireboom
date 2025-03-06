// Package consts
/*
 环境变量常量
*/
package consts

// command params
const (
	WebPort    = "web-port"
	ActiveMode = "active"

	DevMode                = "dev"
	EnableAuth             = "enable-auth"
	EnableRebuild          = "enable-rebuild"
	EnableSwagger          = "enable-swagger"
	EnableHookReport       = "enable-hook-report"
	EnableWebConsole       = "enable-web-console"
	EnableDebugPprof       = "enable-debug-pprof"
	EnableLogicDelete      = "enable-logic-delete"
	RegenerateKey          = "regenerate-key"
	IgnoreMergeEnvironment = "ignore-merge-environment"
)

// command params default value
const (
	DefaultWebPort    = "9123"
	DefaultProdActive = "prod"
)

// env file param
const (
	GithubRawProxyUrl = "GITHUB_RAW_PROXY_URL"
	GithubProxyUrl    = "GITHUB_PROXY_URL"
	FbRepoUrlMirror   = "FB_REPO_URL_MIRROR"
	FbRawUrlMirror    = "FB_RAW_URL_MIRROR"
)

// runtime temp param
const (
	FbVersion = "FbVersion"
	FbCommit  = "FbCommit"
)

// engine status param name
const (
	GlobalStartTime   = "globalStartTime"
	EngineStartTime   = "engineStartTime"
	EnginePrepareTime = "enginePrepareTime"
	EngineStatusField = "engineStatus"
	EngineFirstStatus = "engineFirstStatus"
)

// engine status param value
const (
	EngineBuilding       = "building"
	EngineIncrementBuild = "incrementBuild"
	EngineBuildSucceed   = "buildSucceed"
	EngineBuildFailed    = "buildFailed"
	EngineStarting       = "starting"
	EngineIncrementStart = "incrementStart"
	EngineStartSucceed   = "startSucceed"
	EngineStartFailed    = "startFailed"
)

const (
	HookReportTime   = "hookReportTime"
	HookReportStatus = "hookReportStatus"
)

const (
	DatabaseExecuteTimeout = "FB_DATABASE_EXECUTE_TIMEOUT"
	DatabaseCloseTimeout   = "FB_DATABASE_CLOSE_TIMEOUT"
)

const JaegerWithSpanInOut = "JAEGER_WITH_SPAN_IN_OUT"
