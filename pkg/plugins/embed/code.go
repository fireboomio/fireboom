// Package embed
/*
 嵌入资源管理
 DefaultFs 系统默认配置，包括env, globalOperation, globalSetting, license
 TemplateFs 模板配置，用作渲染默认配置的初始化模板
 ResourceFs 资源配置, 包括banner图，application设置，内省查询参数
 DatasourceFs 内置数据源
 RoleFs 内置角色
 DirectiveExampleFs graphql指令示例
*/
package embed

import (
	"embed"
)

var (
	//go:embed default/*
	DefaultFs embed.FS
	//go:embed resource/*
	ResourceFs embed.FS
	//go:embed datasource/*
	DatasourceFs embed.FS
	//go:embed role/*
	RoleFs embed.FS
	//go:embed directive_example/*
	DirectiveExampleFs embed.FS
)

const (
	DefaultRoot          = "default"
	ResourceRoot         = "resource"
	DatasourceRoot       = "datasource"
	RoleRoot             = "role"
	DirectiveExampleRoot = "directive_example"
)
