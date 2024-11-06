// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve 按照引擎需要的配置定义PostResolveTransformations，留作后续处理graphql响应时格式化日期格式
*/
package directives

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/pool"
	"golang.org/x/exp/slices"
)

const (
	formatDateTimeName             = "formatDateTime"
	formatDateTimeNotAllowedFormat = "expected format %v, but found [%s:%s]"
)

var formatDateTimeAllows = []string{"date", "date-time"}

type formatDateTime struct{}

func (s *formatDateTime) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockWorked(appendIfExistExampleGraphql(i18n.FormatDateTimeDesc.String())),
		Name:        formatDateTimeName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments:   dateTimeFormatArguments(),
	}
}

func (s *formatDateTime) Definitions() ast.DefinitionList {
	return dateTimeFormatDefinitions()
}

func (s *formatDateTime) Resolve(resolver *SelectionResolver) (err error) {
	if format := resolver.Schema.Value.Format; !slices.Contains(formatDateTimeAllows, format) {
		return fmt.Errorf(formatDateTimeNotAllowedFormat, formatDateTimeAllows, resolver.Schema.Value.Type, format)
	}

	if len(dateFormatArgValue(resolver.Arguments)) == 0 {
		return fmt.Errorf(argumentRequiredFormat, utils.JoinString("/", dateTimeArgFormat, dateTimeArgCustomFormat))
	}
	return
}

func init() {
	registerDirective(formatDateTimeName, &formatDateTime{})
	pool.DateFormatFunc = func(args map[string]string, value string) string {
		return cast.ToTime(value).Format(dateFormatArgValue(args))
	}
}
