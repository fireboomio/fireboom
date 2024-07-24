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
	"strings"
)

const (
	transformName              = "transform"
	transformArgName           = "get"
	transformInvalidPathFormat = `invalid path [%s] @transform`
)

type transform struct{}

func (s *transform) Directive() *ast.DirectiveDefinition {
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.TransformDesc.String()),
		Name:        transformName,
		Locations:   []ast.DirectiveLocation{ast.LocationField},
		Arguments: ast.ArgumentDefinitionList{{
			Description: i18n.TransformArgGetDesc.String(),
			Name:        transformArgName,
			Type:        ast.NonNullNamedType(consts.ScalarString, nil),
		}},
	}
}

func (s *transform) Definitions() ast.DefinitionList {
	return nil
}

func (s *transform) Resolve(resolver *SelectionResolver) (err error) {
	value, ok := resolver.Arguments[transformArgName]
	if !ok {
		err = fmt.Errorf(argumentRequiredFormat, transformArgName)
		return
	}

	postTransformation := &wgpb.PostResolveGetTransformation{}
	postTransformation.From = append(postTransformation.From, resolver.Path...)
	postTransformation.To = append(postTransformation.To, resolver.Path...)
	var transformPaths []string
	for _, item := range strings.Split(value, utils.StringDot) {
		if item == utils.ArrayPath {
			continue
		}

		transformPaths = append(transformPaths, item)
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

	endWithArrayPath := resolver.Path[len(resolver.Path)-1] == utils.ArrayPath
	switch schema.Value.Type {
	case openapi3.TypeArray:
		if endWithArrayPath {
			postTransformation.From = append(postTransformation.From, utils.ArrayPath)
		}
	default:
		if arrayVisited {
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
	})

	return
}

func init() {
	registerDirective(transformName, &transform{})
}
