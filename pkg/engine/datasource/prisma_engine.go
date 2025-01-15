// Package datasource
/*
 提供prisma引擎的封装
*/
package datasource

import (
	"context"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/fileloader"
	"regexp"
	"slices"
	"strings"
)

type EngineInput struct {
	PrismaSchema         string
	PrismaSchemaFilepath string
	EnvironmentRequired  bool
}

func buildEnvironments(engineInput EngineInput) (envs []string) {
	if !engineInput.EnvironmentRequired {
		return
	}
	var schemas []string
	if len(engineInput.PrismaSchema) > 0 {
		schemas = append(schemas, engineInput.PrismaSchema)
	}
	if len(engineInput.PrismaSchemaFilepath) > 0 {
		introspectSchema, _ := extractIntrospectSchema(engineInput.PrismaSchemaFilepath)
		schemas = append(schemas, introspectSchema)
	}
	for _, item := range schemas {
		if variable, ok := extractEnvironmentVariable(item); ok && !slices.ContainsFunc(envs, func(s string) bool {
			return strings.HasPrefix(s, variable+"=")
		}) {
			envs = append(envs, utils.JoinString("=", variable, utils.GetStringWithLockViper(variable)))
		}
	}
	return
}

// 内省并缓存graphqlSchema，数据库类型数据源/prisma数据源
func introspectForPrisma(actionDatabase ActionDatabase, dsName string) (graphqlSchema string, err error) {
	engineInput, skipGraphql, err := actionDatabase.fetchSchemaEngineInput()
	if err != nil {
		return
	}

	prismaSchema, err := Introspect(context.Background(), engineInput, dsName)
	if err != nil {
		return
	}
	if err = CachePrismaSchemaText.Write(dsName, fileloader.SystemUser, []byte(prismaSchema)); err != nil || skipGraphql {
		return
	}

	if len(engineInput.PrismaSchemaFilepath) == 0 {
		engineInput.PrismaSchemaFilepath = CachePrismaSchemaText.GetPath(dsName)
	}
	if graphqlSchema, err = IntrospectGraphqlSchema(engineInput); err != nil {
		return
	}

	cacheGraphqlSchema(dsName, graphqlSchema)
	return
}

var envUrlRegexp = regexp.MustCompile(`env\("([^"]+)"\)`)

// 匹配env("aaa")中变量aaa并且在命令行执行参数中追加
// 替换真实的环境变量且保证env变量不在缓存中泄漏
func extractEnvironmentVariable(prismaSchema string) (string, bool) {
	if matches := envUrlRegexp.FindStringSubmatch(prismaSchema); len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func extractIntrospectSchema(prismaSchemaFilepath string) (introspectSchema string, err error) {
	prismaSchemaFileText, err := utils.ReadWithCondition(prismaSchemaFilepath, nil,
		func(_ int, line string) bool { return strings.HasSuffix(line, "}") })
	if err != nil {
		return
	}

	introspectSchema = utils.JoinString("\n", prismaSchemaFileText...)
	return
}
