// Package directives
/*
 graphql指令通用代码，需要实现接口CustomDirective
 OperationDirective 继承CustomDirective，额外实现自定义解析operation上指令
 SelectionDirective 继承CustomDirective，额外实现自定义解析查询字段上的指令
 VariableDirective 继承CustomDirective，额外实现自定义解析传递参数上的指令
 指令和其参数的描述通过i18n实现多语言支持
 指令的示例代码通过embed实现去除硬编码
*/
package directives

import (
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/plugins/embed"
	"fireboom-server/pkg/plugins/fileloader"
	"fireboom-server/pkg/plugins/i18n"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"io/fs"
	"strconv"
	"strings"
)

type (
	OperationResolver struct {
		Operation *wgpb.Operation
		Arguments map[string]string
	}
	SelectionResolver struct {
		OperationResolver
		Path                []string
		Schema              *openapi3.SchemaRef
		VariableSchemas     openapi3.Schemas
		VariableExported    map[string]bool
		OperationDefinition *ast.OperationDefinition
	}
	VariableResolver struct {
		SelectionResolver
		ArgumentDefinitions openapi3.Schemas
	}

	CustomDirective interface {
		Directive() *ast.DirectiveDefinition
		Definitions() ast.DefinitionList
	}
	OperationDirective interface {
		CustomDirective
		Resolve(*OperationResolver) error
	}
	SelectionDirective interface {
		CustomDirective
		Resolve(*SelectionResolver) error
	}
	VariableDirective interface {
		CustomDirective
		Resolve(*VariableResolver) (bool, bool, error)
	}
	variableDirectiveRemoved interface{ variableRemoved() }
	selectionFieldCustomized interface{ fieldCustomized() }
)

const (
	commonArgName                   = "name"
	argumentRequiredFormat          = `argument [%s] required`
	argumentValueNotSupportedFormat = `value [%s] in argument [%s] not supported`
	exampleFlag                     = `
@example `
)

var (
	logger                = zap.L()
	operationDirectiveMap map[string]OperationDirective
	selectionDirectiveMap map[string]SelectionDirective
	variableDirectiveMap  map[string]VariableDirective
	baseDirectives        = []string{"removeNullVariables", "deprecated", "specifiedBy"}
	examples              = make(map[string]string)
)

func IsBaseDirective(name string) bool {
	return slices.Contains(baseDirectives, name)
}

func GetDirectiveSchemas() (result []CustomDirective) {
	for _, directive := range operationDirectiveMap {
		result = append(result, directive)
	}
	for _, directive := range selectionDirectiveMap {
		result = append(result, directive)
	}
	for _, directive := range variableDirectiveMap {
		result = append(result, directive)
	}
	return
}

func GetOperationDirectiveMapByName(name string) OperationDirective {
	return operationDirectiveMap[name]
}

func GetSelectionDirectiveByName(name string) SelectionDirective {
	return selectionDirectiveMap[name]
}

func GetVariableDirectiveByName(name string) VariableDirective {
	return variableDirectiveMap[name]
}

func VariableDefinitionRemoveRequired(directives ast.DirectiveList) bool {
	return slices.ContainsFunc(directives, func(directive *ast.Directive) bool {
		_, ok := variableDirectiveMap[directive.Name].(variableDirectiveRemoved)
		return ok
	})
}

func FieldCustomizedDefinition(directives ast.DirectiveList, resolver *SelectionResolver) (string, error) {
	for _, item := range directives {
		directive := selectionDirectiveMap[item.Name]
		if _, ok := directive.(selectionFieldCustomized); ok {
			resolver.Arguments = ResolveDirectiveArguments(item.Arguments)
			return item.Name, directive.Resolve(resolver)
		}
	}
	return "", nil
}

func registerDirective(name string, directive CustomDirective) {
	switch convert := directive.(type) {
	case OperationDirective:
		operationDirectiveMap[name] = convert
	case SelectionDirective:
		selectionDirectiveMap[name] = convert
	case VariableDirective:
		variableDirectiveMap[name] = convert
	default:
		logger.Warn("not implement directive", zap.String("name", name))
	}
}

func appendIfExistExampleGraphql(desc string) string {
	content, ok := examples[i18n.GetCallerMode()]
	if !ok {
		return desc
	}

	return desc + exampleFlag + content
}

// ResolveDirectiveArguments 解析指令参数并返回map
func ResolveDirectiveArguments(arguments ast.ArgumentList) (argMap map[string]string) {
	if len(arguments) == 0 {
		return
	}

	argMap = make(map[string]string)
	for _, item := range arguments {
		if item.Value == nil {
			continue
		}

		argMap[item.Name] = argumentValueString(item.Value, false)
	}
	return
}

func argumentValueString(v *ast.Value, quote bool) string {
	if v == nil {
		return ""
	}

	switch v.Kind {
	case ast.Variable:
		return VariablePrefix + v.Raw
	case ast.StringValue, ast.BlockValue, ast.EnumValue:
		if quote {
			return strconv.Quote(v.Raw)
		}
		return v.Raw
	case ast.ListValue:
		var val []string
		for _, elem := range v.Children {
			val = append(val, argumentValueString(elem.Value, true))
		}
		return "[" + strings.Join(val, utils.StringComma) + "]"
	case ast.ObjectValue:
		var val []string
		for _, elem := range v.Children {
			val = append(val, strconv.Quote(elem.Name)+":"+argumentValueString(elem.Value, true))
		}
		return "{" + strings.Join(val, utils.StringComma) + "}"
	default:
		return v.Raw
	}
}

func init() {
	operationDirectiveMap = make(map[string]OperationDirective)
	selectionDirectiveMap = make(map[string]SelectionDirective)
	variableDirectiveMap = make(map[string]VariableDirective)

	sources, _ := fs.ReadDir(embed.DirectiveExampleFs, embed.DirectiveExampleRoot)
	for _, item := range sources {
		itemName := strings.TrimSuffix(item.Name(), string(fileloader.ExtGraphql))
		itemPath := utils.NormalizePath(embed.DirectiveExampleRoot, item.Name())
		itemBytes, _ := fs.ReadFile(embed.DirectiveExampleFs, itemPath)
		examples[itemName] = string(itemBytes)
	}
}
