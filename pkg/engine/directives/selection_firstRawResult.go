// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve 标识参数内部传递，与@export组合使用
*/
package directives

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/vektah/gqlparser/v2/ast"
)

const firstRawResultName = "firstRawResult"

type firstRawResult struct{}

func (s *firstRawResult) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.FirstRawResultDesc.String())),
		Name:        firstRawResultName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
	}
}

func (s *firstRawResult) Definitions() ast.DefinitionList {
	return nil
}

func (s *firstRawResult) Resolve(resolver *SelectionResolver) (err error) {
	return
}

func FieldQueryRawWrapArrayRequired(directives ast.DirectiveList, fieldOriginName string) bool {
	wrapArrayRequired := fieldOriginName == "queryRaw" || fieldOriginName == "optional_queryRaw"
	if wrapArrayRequired {
		for _, item := range directives {
			directive := selectionDirectiveMap[item.Name]
			if _, ok := directive.(*firstRawResult); ok {
				wrapArrayRequired = false
				break
			}
		}
	}
	return wrapArrayRequired
}

func init() {
	registerDirective(firstRawResultName, &firstRawResult{})
}
