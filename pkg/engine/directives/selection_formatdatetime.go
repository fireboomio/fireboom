// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve 按照引擎需要的配置定义PostResolveTransformations，留作后续处理graphql响应时格式化日期格式
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
)

const (
	formatDateTimeName               = "formatDateTime"
	formatDateTimeNotSupportedFormat = "expected scalar %v, but found [%s:%s]"
)

var formatDateTimeScalars = []string{consts.ScalarDate, consts.ScalarDateTime}

type formatDateTime struct{}

func (t *formatDateTime) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.FormatDateTimeDesc.String()),
		Name:        formatDateTimeName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments:   dateTimeFormatArguments(),
	}
}

func (t *formatDateTime) Definitions() ast.DefinitionList {
	return dateTimeFormatDefinitions()
}

func (t *formatDateTime) Resolve(resolver *SelectionResolver) (err error) {
	if format := resolver.Schema.Value.Format; !slices.Contains(formatDateTimeScalars, format) {
		return fmt.Errorf(formatDateTimeNotSupportedFormat, formatDateTimeScalars, resolver.Schema.Value.Type, format)
	}

	postTransformation := &wgpb.PostResolveGetTransformation{
		From:           resolver.Path,
		To:             resolver.Path,
		DateTimeFormat: dateFormatArgValue(resolver.Arguments),
	}
	resolver.Operation.PostResolveTransformations = append(resolver.Operation.PostResolveTransformations, &wgpb.PostResolveTransformation{
		Kind:  wgpb.PostResolveTransformationKind_GET_POST_RESOLVE_TRANSFORMATION,
		Depth: int32(len(postTransformation.From)),
		Get:   postTransformation,
	})
	return
}

func init() {
	registerDirective(formatDateTimeName, &formatDateTime{})
}
