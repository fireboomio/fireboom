// Package directives
/*
 实现OperationDirective接口，只能定义在LocationMutation上
 Resolve 开启EnableTransaction，引擎在执行graphql请求时合并多个并按定义顺序排列graphql达到顺序执行且保证事务
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/datasources/database"
	"github.com/wundergraph/wundergraph/pkg/engineconfigloader"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
)

const (
	transactionName                  = "transaction"
	transactionArgMaxWaitSecondsName = "maxWaitSeconds"
	transactionArgTimeoutSecondsName = "timeoutSeconds"
	transactionArgIsolationLevelName = "isolationLevel"
	transactionArgIsolationLevelType = "TransactionIsolationLevel"
)

type transaction struct{}

func (e *transaction) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.TransactionDesc.String()),
		Name:        transactionName,
		Locations:   []ast.DirectiveLocation{ast.LocationQuery, ast.LocationMutation},
		Arguments: ast.ArgumentDefinitionList{
			{
				Description:  i18n.TransactionArgMaxWaitSecondsDesc.String(),
				Name:         transactionArgMaxWaitSecondsName,
				Type:         ast.NamedType(consts.ScalarInt, nil),
				DefaultValue: &ast.Value{Kind: ast.IntValue, Raw: cast.ToString(database.DefaultTransaction.MaxWaitSeconds)},
			},
			{
				Description:  i18n.TransactionArgTimeoutSecondsDesc.String(),
				Name:         transactionArgTimeoutSecondsName,
				Type:         ast.NamedType(consts.ScalarInt, nil),
				DefaultValue: &ast.Value{Kind: ast.IntValue, Raw: cast.ToString(database.DefaultTransaction.TimeoutSeconds)},
			},
			{
				Description:  i18n.TransactionArgIsolationLevelDesc.String(),
				Name:         transactionArgIsolationLevelName,
				Type:         ast.NamedType(transactionArgIsolationLevelType, nil),
				DefaultValue: &ast.Value{Kind: ast.EnumValue, Raw: database.DefaultTransaction.IsolationLevel.String()},
			},
		},
	}
}

func (e *transaction) Definitions() ast.DefinitionList {
	var isolationLevelEnumValues ast.EnumValueList
	for k := range wgpb.OperationTransactionIsolationLevel_value {
		isolationLevelEnumValues = append(isolationLevelEnumValues, &ast.EnumValueDefinition{Name: k})
	}

	return ast.DefinitionList{
		{
			Kind:       ast.Enum,
			Name:       transactionArgIsolationLevelType,
			EnumValues: isolationLevelEnumValues,
		},
	}
}

func (e *transaction) Resolve(resolver *OperationResolver) error {
	transactionInfo := &wgpb.OperationTransaction{}
	resolver.Operation.Transaction = transactionInfo
	if value, ok := resolver.Arguments[transactionArgMaxWaitSecondsName]; ok {
		transactionInfo.MaxWaitSeconds = cast.ToInt64(value)
	}
	if transactionInfo.MaxWaitSeconds == 0 {
		transactionInfo.MaxWaitSeconds = database.DefaultTransaction.MaxWaitSeconds
	}
	if value, ok := resolver.Arguments[transactionArgTimeoutSecondsName]; ok {
		transactionInfo.TimeoutSeconds = cast.ToInt64(value)
	}
	if transactionInfo.TimeoutSeconds == 0 {
		transactionInfo.TimeoutSeconds = database.DefaultTransaction.TimeoutSeconds
	}
	if value, ok := resolver.Arguments[transactionArgIsolationLevelName]; ok {
		transactionInfo.IsolationLevel = wgpb.OperationTransactionIsolationLevel(wgpb.OperationTransactionIsolationLevel_value[value])
	}
	return nil
}

func init() {
	registerDirective(transactionName, &transaction{})
	engineconfigloader.AddDisallowParallelFetchDirective(transactionName)
}
