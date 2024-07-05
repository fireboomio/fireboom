// Package build
/*
 将数据源名称添加到graphql的命名上
*/
package build

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/datasource"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"regexp"
)

type dataSourceRename struct {
	name         string
	doc          *ast.SchemaDocument
	engineConfig *engineConfiguration
	action       datasource.Action

	rootNodes          []*wgpb.TypeField
	childNodes         []*wgpb.TypeField
	fieldQuoteTypesMap map[string][]string

	rootOperationTypeNameMap   map[string]string
	ignoreRenameTypeNames      []string
	joinFieldRequiredTypeNames []string
}

func newDataSourceRename(name string, doc *ast.SchemaDocument, engineConfig *engineConfiguration, action datasource.Action) *dataSourceRename {
	return &dataSourceRename{
		name:               name,
		doc:                doc,
		engineConfig:       engineConfig,
		action:             action,
		fieldQuoteTypesMap: make(map[string][]string),
	}
}

func (d *dataSourceRename) resolve() {
	d.rootOperationTypeNameMap = make(map[string]string)
	for _, itemSchema := range d.doc.Schema {
		for _, itemType := range itemSchema.OperationTypes {
			d.rootOperationTypeNameMap[itemType.Type] = utils.UppercaseFirst(string(itemType.Operation))
			if datasource.ContainsRootDefinition(itemType.Type) {
				d.ignoreRenameTypeNames = append(d.ignoreRenameTypeNames, itemType.Type)
			}
		}
	}
	if len(d.rootOperationTypeNameMap) == 0 {
		for _, name := range datasource.RootObjectNames {
			d.rootOperationTypeNameMap[name] = name
		}
	}
	d.renameDefinitions()
	d.renameDirectives()
	d.setRootNodeQuotes()
}

// 将根字段所引用的所有子字段通过索引方式记录
// 留作后续运行期间再转换成真实的子字段
func (d *dataSourceRename) setRootNodeQuotes() {
	for _, node := range d.rootNodes {
		node.Quotes = make(map[int32]*wgpb.QuoteField)
		for index, rootFieldName := range node.FieldNames {
			var quotes []int32
			d.searchNodeQuotes(rootFieldName, &quotes)
			if len(quotes) > 0 {
				node.Quotes[int32(index)] = &wgpb.QuoteField{Indexes: quotes}
			}
		}
	}
}

// 递归搜索字段引用
func (d *dataSourceRename) searchNodeQuotes(fieldName string, quotes *[]int32) {
	names, ok := d.fieldQuoteTypesMap[fieldName]
	if !ok {
		return
	}

	for _, name := range names {
		quoteIndex := slices.IndexFunc(d.childNodes, func(item *wgpb.TypeField) bool { return item.TypeName == name })
		if quoteIndex == -1 || slices.Contains(*quotes, int32(quoteIndex)) {
			continue
		}

		*quotes = append(*quotes, int32(quoteIndex))
		d.searchNodeQuotes(name, quotes)
	}
}

// 根据类型不同实现定义不同方式的重命名
func (d *dataSourceRename) renameDefinitions() {
	if len(d.doc.Definitions) == 0 {
		return
	}

	for _, definition := range d.doc.Definitions {
		switch definition.Kind {
		case ast.Scalar:
			d.ignoreRenameTypeNames = append(d.ignoreRenameTypeNames, definition.Name)
		case ast.Object:
			d.joinFieldRequiredTypeNames = append(d.joinFieldRequiredTypeNames, d.applyNamespace(definition.Name, false))
		}
	}

	for _, definition := range d.doc.Definitions {
		switch definition.Kind {
		case ast.Scalar:
			if datasource.IsScalarJsonName(definition.Name) {
				definition.Name = consts.ScalarJSON
			}
		case ast.Interface:
			definition.Name = d.applyNamespace(definition.Name, true)
		case ast.Union:
			definition.Name = d.applyNamespace(definition.Name, true)
			d.renameTypes(&definition.Types)
		case ast.Enum:
			definition.Name = d.applyNamespace(definition.Name, true)
		case ast.InputObject:
			definition.Name = d.applyNamespace(definition.Name, true)
			d.renameInterfaces(&definition.Interfaces)
			d.renameFields(definition, false)
		case ast.Object:
			_, isRootNode := d.rootOperationTypeNameMap[definition.Name]
			if !isRootNode {
				definition.Name = d.applyNamespace(definition.Name, true)
			}

			d.renameInterfaces(&definition.Interfaces)
			d.renameFields(definition, isRootNode)
		}
	}
	return
}

// 重命名字段，包含根类型和普通字段
func (d *dataSourceRename) renameFields(definition *ast.Definition, isRootNode bool) {
	if len(definition.Fields) == 0 {
		return
	}

	joinFieldRequired := slices.Contains(d.joinFieldRequiredTypeNames, definition.Name)
	node := &wgpb.TypeField{TypeName: definition.Name}
	for _, field := range definition.Fields {
		if isRootNode {
			// 根类型时重命名所有子类型即Query/Mutation/Subscription下的graphql
			node.TypeName = d.rootOperationTypeNameMap[definition.Name]
			fieldRename := d.applyNamespace(field.Name, false)
			d.engineConfig.resolveFieldConfiguration(field, fieldRename, node.TypeName, d.action)
			field.Name = fieldRename
			// 在描述中添加数据源特殊标识用作引用数据源的筛选
			field.Description += fmt.Sprintf(datasourceFormat, d.name)
		} else if len(field.Arguments) > 0 {
			// 非根类型时若有入参也需要处理
			d.engineConfig.resolveFieldConfiguration(field, field.Name, node.TypeName, d.action)
		}
		node.FieldNames = append(node.FieldNames, field.Name)

		// 将所有字段的引用关系保存，留作后续递归搜索引用
		if d.renameType(field.Type) && !slices.Contains(d.fieldQuoteTypesMap[definition.Name], field.Type.Name()) {
			if isRootNode {
				d.fieldQuoteTypesMap[field.Name] = append(d.fieldQuoteTypesMap[field.Name], field.Type.Name())
			} else if joinFieldRequired {
				d.fieldQuoteTypesMap[definition.Name] = append(d.fieldQuoteTypesMap[definition.Name], field.Type.Name())
			}
		}
		d.engineConfig.saveFieldArgumentTypeNames(field.Name, node.TypeName, d.renameArguments(field.Arguments))
	}

	if isRootNode {
		d.rootNodes = append(d.rootNodes, node)
		return
	}

	if joinFieldRequired {
		definition.Fields = append(definition.Fields, joinFieldDefinition, joinMutationFieldDefinition)
		node.FieldNames = append(node.FieldNames, datasource.JoinFieldName, datasource.JoinMutationFieldName)
		d.childNodes = append(d.childNodes, node)
	}
}

func (d *dataSourceRename) renameInterfaces(interfaces *[]string) {
	if interfaces == nil || len(*interfaces) == 0 {
		return
	}

	for index, item := range *interfaces {
		(*interfaces)[index] = d.applyNamespace(item, false)
	}
}

func (d *dataSourceRename) renameTypes(types *[]string) {
	if types == nil || len(*types) == 0 {
		return
	}

	for index, item := range *types {
		(*types)[index] = d.applyNamespace(item, false)
	}
}

// 重命名类型，忽略scalar类型但对json类型特殊处理成统一的大写JSON
// 忽略根类型的重命名
func (d *dataSourceRename) renameType(t *ast.Type) bool {
	realType := fetchRealType(t)
	originName := realType.NamedType
	if datasource.IsScalarJsonName(originName) {
		realType.NamedType = consts.ScalarJSON
		if originName != consts.ScalarJSON {
			d.engineConfig.resolveTypeConfigurations(consts.ScalarJSON, originName)
		}
		return false
	}

	if datasource.IsBaseScalarName(originName) || slices.Contains(d.ignoreRenameTypeNames, originName) {
		return false
	}

	realType.NamedType = d.applyNamespace(originName, false)
	return true
}

// 重命名指令中的参数类型名称
func (d *dataSourceRename) renameDirectives() {
	if len(d.doc.Directives) == 0 {
		return
	}

	for _, directive := range d.doc.Directives {
		d.renameArguments(directive.Arguments)
	}
}

// 重命名入参中的参数类型名称
func (d *dataSourceRename) renameArguments(arguments ast.ArgumentDefinitionList) []string {
	if len(arguments) == 0 {
		return nil
	}

	renamedTypes := make(map[string]bool)
	for _, arg := range arguments {
		if d.renameType(arg.Type) {
			renamedTypes[arg.Type.Name()] = true
		}
	}
	return maps.Keys(renamedTypes)
}

// 将数据源名称添加前面
func (d *dataSourceRename) applyNamespace(name string, resolved bool) string {
	renameTo := utils.JoinString("_", d.name, name)
	if resolved {
		d.engineConfig.resolveTypeConfigurations(renameTo, name)
	}
	return renameTo
}

// 获取真实的类型
// 例如：![Age] => Age
func fetchRealType(t *ast.Type) *ast.Type {
	if t.NamedType != "" {
		return t
	}

	return fetchRealType(t.Elem)
}

const datasourceFormat = `<#datasource#>%s<#datasource#>`

var (
	datasourceRegexp    = regexp.MustCompile(`<#datasource#>([^}]+)<#datasource#>`)
	joinFieldDefinition = &ast.FieldDefinition{
		Name: datasource.JoinFieldName,
		Type: ast.NonNullNamedType(consts.TypeQuery, nil),
	}
	joinMutationFieldDefinition = &ast.FieldDefinition{
		Name: datasource.JoinMutationFieldName,
		Type: ast.NonNullNamedType(consts.TypeMutation, nil),
	}
)

// 通过在description中添加的特殊标识匹配出数据源名称
func matchDatasource(description string) (string, string) {
	return utils.MatchNameWithRegexp(description, datasourceRegexp)
}
