// Package directives
/*
 实现SelectionDirective接口，只能定义在LocationField上
 Resolve 按照引擎需要的配置定义PostResolveTransformations，且修改响应的jsonschema，引擎层会转换响应
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/i18n"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/slices"
	"strings"
)

const (
	transformName                      = "transform"
	transformArgGetName                = "get"
	transformArgMathName               = "math"
	transformArgMathType               = "TransformMath"
	transformInvalidPathFormat         = `invalid path [%s] @transform`
	transformArgExpectTypeFormat       = `argument [%s] just allow apply on [%s] result`
	transformArgMathExpectNumberFormat = `invalid [%s] array, expected number array for math [%s]`
)

var (
	transformNumberMathArray = []wgpb.PostResolveTransformationMath{
		wgpb.PostResolveTransformationMath_MAX,
		wgpb.PostResolveTransformationMath_MIN,
		wgpb.PostResolveTransformationMath_AVG,
		wgpb.PostResolveTransformationMath_SUM,
	}
	transformNumberSchemaTypes = []string{
		openapi3.TypeInteger,
		openapi3.TypeNumber,
	}
)

type transform struct{}

func (s *transform) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: prependMockSwitch(appendIfExistExampleGraphql(i18n.TransformDesc.String())),
		Name:        transformName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{{
			Description: i18n.TransformArgGetDesc.String(),
			Name:        transformArgGetName,
			Type:        ast.NamedType(consts.ScalarString, nil),
		}, {
			Description: i18n.TransformArgGetDesc.String(),
			Name:        transformArgMathName,
			Type:        ast.NamedType(transformArgMathType, nil),
		}},
	}
}

func (s *transform) Definitions() ast.DefinitionList {
	var nameEnumValues ast.EnumValueList
	for k := range wgpb.PostResolveTransformationMath_value {
		nameEnumValues = append(nameEnumValues, &ast.EnumValueDefinition{Name: k})
	}

	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       transformArgMathType,
		EnumValues: nameEnumValues,
	}}
}

func (s *transform) Resolve(resolver *SelectionResolver) (err error) {
	getValue, getOk := resolver.Arguments[transformArgGetName]
	mathValue, mathOk := resolver.Arguments[transformArgMathName]
	if !getOk && !mathOk {
		err = fmt.Errorf(argumentRequiredFormat, utils.JoinString(" or ", transformArgGetName, transformArgMathName))
		return
	}

	postTransformation := &wgpb.PostResolveGetTransformation{}
	postTransformation.From = append(postTransformation.From, resolver.Path...)
	postTransformation.To = append(postTransformation.To, resolver.Path...)
	var transformPaths []string
	if getOk {
		for _, item := range strings.Split(getValue, utils.StringDot) {
			if item == utils.ArrayPath {
				continue
			}

			transformPaths = append(transformPaths, item)
		}
	}
	transformLength := len(transformPaths)
	schema := resolver.Schema
	var index int
	var arrayVisited bool
loop:
	for {
		if index == transformLength || schema.Value == nil {
			break loop
		}

		current := transformPaths[index]
		switch schema.Value.Type {
		case openapi3.TypeObject:
			if schema.Value.Properties == nil {
				break loop
			}
			nextSchema, ok := schema.Value.Properties[current]
			if !ok {
				break loop
			}

			index++
			schema.Value = nextSchema.Value
			postTransformation.From = append(postTransformation.From, current)
		case openapi3.TypeArray:
			if schema.Value.Items == nil {
				break loop
			}

			arrayVisited = true
			schema.Value = schema.Value.Items.Value
			if index > 0 || index == 0 && postTransformation.From[len(postTransformation.From)-1] != utils.ArrayPath {
				postTransformation.From = append(postTransformation.From, utils.ArrayPath)
			}
		default:
			break loop
		}
	}
	if index != transformLength {
		err = fmt.Errorf(transformInvalidPathFormat, utils.JoinStringWithDot(transformPaths[:index+1]...))
		return
	}

	var resolveMath *wgpb.PostResolveTransformationMath
	endWithArrayPath := resolver.Path[len(resolver.Path)-1] == utils.ArrayPath
	if mathOk {
		// 中间/开头/结尾均不不存在数组
		if !arrayVisited && !endWithArrayPath && schema.Value.Type != openapi3.TypeArray {
			err = fmt.Errorf(transformArgExpectTypeFormat, transformArgMathName, openapi3.TypeArray)
			return
		}
		mathType, ok := wgpb.PostResolveTransformationMath_value[mathValue]
		if !ok {
			err = fmt.Errorf(argumentValueNotSupportedFormat, mathValue, transformArgMathName)
			return
		}
		if schema.Value.Type == openapi3.TypeArray {
			schema.Value = schema.Value.Items.Value
		}
		_resolveMath := wgpb.PostResolveTransformationMath(mathType)
		if slices.Contains(transformNumberMathArray, _resolveMath) &&
			!slices.Contains(transformNumberSchemaTypes, schema.Value.Type) {
			err = fmt.Errorf(transformArgMathExpectNumberFormat, schema.Value.Type, mathValue)
			return
		}
		resolveMath = &_resolveMath
	}

	switch schema.Value.Type {
	case openapi3.TypeArray:
		if endWithArrayPath {
			postTransformation.From = append(postTransformation.From, utils.ArrayPath)
		}
	default:
		if arrayVisited && !mathOk {
			schema.Value = &openapi3.Schema{Items: &openapi3.SchemaRef{Value: schema.Value}, Type: openapi3.TypeArray}
			if !endWithArrayPath {
				arrayIndex := utils.LastIndex(postTransformation.From, utils.ArrayPath)
				resolver.Operation.PostResolveTransformations = append(resolver.Operation.PostResolveTransformations, &wgpb.PostResolveTransformation{
					Kind:  wgpb.PostResolveTransformationKind_GET_POST_RESOLVE_TRANSFORMATION,
					Depth: int32(len(postTransformation.From)),
					Get: &wgpb.PostResolveGetTransformation{
						From: postTransformation.From,
						To:   postTransformation.From[:arrayIndex+1],
					},
				})
				postTransformation.From = postTransformation.From[:arrayIndex]
			}
		}
	}

	resolver.Operation.PostResolveTransformations = append(resolver.Operation.PostResolveTransformations, &wgpb.PostResolveTransformation{
		Kind:  wgpb.PostResolveTransformationKind_GET_POST_RESOLVE_TRANSFORMATION,
		Depth: int32(len(postTransformation.From)),
		Get:   postTransformation,
		Math:  resolveMath,
	})

	return
}

func init() {
	registerDirective(transformName, &transform{})
}
