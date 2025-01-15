// Package consts
/*
 文件路径/文件名常量
*/
package consts

// 工作目录下的根目录
const (
	RootExported = "exported"
	RootStore    = "store"
	RootUpload   = "upload"
	RootTemplate = "template"
)

// exported目录下的子目录和文件
const (
	ExportedGeneratedParent     = "generated"
	ExportedIntrospectionParent = "introspection"
	ExportedMigrationParent     = "migration"

	ExportedGeneratedSwaggerFilename            = "swagger"
	ExportedGeneratedHookSwaggerFilename        = "hook.swagger"
	ExportedGeneratedFireboomConfigFilename     = "fireboom.config"
	ExportedGeneratedFireboomOperationsFilename = "fireboom.operations"
	ExportedGeneratedGraphqlSchemaFilename      = "fireboom.app.schema"
)

// store目录下的子目录
const (
	StoreDatasourceParent     = "datasource"
	StoreAuthenticationParent = "authentication"
	StoreOperationParent      = "operation"
	StoreStorageParent        = "storage"
	StoreSdkParent            = "sdk"
	StoreConfigParent         = "config"
	StoreRoleParent           = "role"
	StoreFragmentParent       = "fragment"
)

// upload目录下的子目录
const (
	UploadOasParent      = "oas"
	UploadAsyncapiParent = "asyncapi"
	UploadPrismaParent   = "prisma"
	UploadSqliteParent   = "sqlite"
	UploadGraphqlParent  = "graphql"
)

// 服务端钩子工作目录下的子目录
const (
	HookGeneratedParent      = "generated"
	HookGlobalParent         = "global"
	HookAuthenticationParent = "authentication"
	HookOperationParent      = "operation"
	HookStorageProfileParent = "upload"
	HookCustomizeParent      = "customize"
	HookProxyParent          = "proxy"
	HookFunctionParent       = "function"
	HookFragmentsParent      = "fragment"
)

// 内嵌的文件名/当前工作目录的文件名
const (
	GlobalOperation = "global.operation"
	GlobalSetting   = "global.setting"

	JaegerConfig = "jaeger.config"

	DefaultEnv = ".env"

	ResourceIntrospect  = "introspect"
	ResourceApplication = "application"
	ResourceBanner      = "banner"

	KeyAuthentication = "authentication"
	KeyLicense        = "license"
)
