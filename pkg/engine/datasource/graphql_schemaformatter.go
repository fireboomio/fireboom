// Package datasource
/*
 对graphql内省请求返回的文本进行处理，格式化成合法的graphql文档
 重写了开源库的实现，添加了对此内省结果的支持
*/
package datasource

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/common/utils"
	"fireboom-server/pkg/engine/directives"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/wundergraph/wundergraph/pkg/pool"
	"golang.org/x/exp/slices"
	"io"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

func newFormatter(w io.Writer) *schemaFormatter {
	return &schemaFormatter{
		indent: "\t",
		writer: w,
	}
}

const (
	KindNonNull    = "NON_NULL"
	kindList       = "LIST"
	kindListFormat = `[%s]`
)

var (
	RootObjectNames       = []string{consts.TypeQuery, consts.TypeMutation, consts.TypeSubscription}
	DefinitionNameSorts   = []ast.DefinitionKind{ast.Object, ast.InputObject, ast.Enum, ast.Union, ast.Interface, ast.Scalar}
	ignoreDefinitionNames = []string{consts.ScalarBoolean, consts.ScalarInt, consts.ScalarFloat, consts.ScalarString, consts.ScalarID}
)

type schemaFormatter struct {
	writer io.Writer

	indent      string
	indentSize  int
	emitBuiltin bool

	padNext  bool
	lineHead bool
}

type (
	schema struct {
		QueryType        *baseDefinition
		MutationType     *baseDefinition
		SubscriptionType *baseDefinition
		Types            []*definition
		Directives       []*directiveDefinition
	}
	baseDefinition struct {
		Name        string
		originName  string
		Description string
	}
	definition struct {
		baseDefinition
		Kind          ast.DefinitionKind
		Fields        []*fieldDefinition
		InputFields   []*argumentDefinition
		EnumValues    []*enumValueDefinition
		PossibleTypes []*valueDefinition
		Interfaces    []*definition
	}
	directiveDefinition struct {
		baseDefinition
		Locations []ast.DirectiveLocation
		Args      []*argumentDefinition
	}
	enumValueDefinition struct {
		baseDefinition
		IsDeprecated      bool
		DeprecationReason *string
	}
	fieldDefinition struct {
		baseDefinition
		IsDeprecated      bool
		DeprecationReason *string
		Args              []*argumentDefinition
		Type              *valueDefinition
	}
	argumentDefinition struct {
		baseDefinition
		Type         *valueDefinition
		DefaultValue any
	}
	valueDefinition struct {
		Kind   ast.DefinitionKind
		Name   *string
		OfType *valueDefinition
	}
)

func formatSchemaString(schema *schema) string {
	buf := pool.GetBytesBuffer()
	defer pool.PutBytesBuffer(buf)
	newFormatter(buf).formatSchema(schema)
	return buf.String()
}

func (f *schemaFormatter) formatSchema(schema *schema) {
	if schema == nil {
		return
	}

	var inSchema bool
	var rootNames []string
	startSchema := func(word string, base *baseDefinition) {
		if !inSchema {
			inSchema = true

			f.writeWord("schema").WriteString("{").writeNewline()
			f.incrementIndent()
		}

		f.writeWord(word).noPadding().WriteString(":").needPadding()
		f.writeWord(utils.UppercaseFirst(base.Name)).writeNewline()
		rootNames = append(rootNames, base.Name)
	}

	if query := schema.QueryType; query != nil {
		startSchema("query", query)
	}
	if mutation := schema.MutationType; mutation != nil {
		startSchema("mutation", mutation)
	}
	if subscription := schema.SubscriptionType; subscription != nil {
		startSchema("subscription", subscription)
	}
	if inSchema {
		f.decrementIndent()
		f.WriteString("}").writeNewline()
	}

	slices.SortFunc(schema.Types, func(a, b *definition) bool {
		aIndex := slices.Index(DefinitionNameSorts, a.Kind)
		bIndex := slices.Index(DefinitionNameSorts, b.Kind)
		return aIndex < bIndex || aIndex == bIndex && a.Name < b.Name
	})
	for _, item := range schema.Types {
		f.formatDefinition(item, rootNames)
	}

	slices.SortFunc(schema.Directives, func(a, b *directiveDefinition) bool {
		return a.Name < b.Name
	})
	for _, item := range schema.Directives {
		f.formatDirectiveDefinition(item)
	}
}

func (f *schemaFormatter) formatInputFieldList(inputFieldList []*argumentDefinition) {
	if len(inputFieldList) == 0 {
		return
	}

	f.WriteString("{").writeNewline()
	f.incrementIndent()

	for _, field := range inputFieldList {
		f.formatArgumentDefinition(field)
		f.writeNewline()
	}

	f.decrementIndent()
	f.WriteString("}")
}

func (f *schemaFormatter) formatFieldList(fieldList []*fieldDefinition) {
	if len(fieldList) == 0 {
		return
	}

	f.WriteString("{").writeNewline()
	f.incrementIndent()

	for _, field := range fieldList {
		f.formatFieldDefinition(field)
	}

	f.decrementIndent()
	f.WriteString("}")
}

func (f *schemaFormatter) formatFieldDefinition(field *fieldDefinition) {
	if !f.emitBuiltin && strings.HasPrefix(field.Name, "__") {
		return
	}

	f.writeDescription(field.Description)

	f.writeWord(field.Name).noPadding()
	f.formatArgumentDefinitionList(field.Args)
	f.noPadding().WriteString(":").needPadding()
	f.formatType(field.Type)

	f.writeNewline()
}

func (f *schemaFormatter) formatArgumentDefinitionList(lists []*argumentDefinition) {
	if len(lists) == 0 {
		return
	}

	f.WriteString("(")
	indented := slices.ContainsFunc(lists, func(item *argumentDefinition) bool {
		return item.Description != ""
	})
	if indented {
		f.incrementIndent()
	}
	for idx, arg := range lists {
		if arg.Description != "" {
			f.writeNewline()
		}
		f.formatArgumentDefinition(arg)
		if idx != len(lists)-1 {
			f.noPadding().writeWord(utils.StringComma)
		}
		if arg.Description != "" {
			f.writeNewline()
		}
	}
	if indented {
		f.decrementIndent()
	}
	f.noPadding().WriteString(")").needPadding()
}

func (f *schemaFormatter) formatArgumentDefinition(def *argumentDefinition) {
	if def.Description != "" {
		f.writeDescription(def.Description)
	}

	f.writeWord(def.Name).noPadding().WriteString(":").needPadding()
	f.formatType(def.Type)

	if def.DefaultValue != nil {
		f.writeWord("=")
		f.formatValue(def.DefaultValue)
	}

	f.needPadding()
}

func (f *schemaFormatter) formatDirectiveLocation(location ast.DirectiveLocation) {
	f.writeWord(string(location))
}

func (f *schemaFormatter) formatDirectiveDefinition(def *directiveDefinition) {
	if directives.IsBaseDirective(def.Name) {
		return
	}

	f.writeDescription(def.Description)
	f.writeWord("directive").WriteString("@").writeWord(def.Name)

	if len(def.Args) != 0 {
		f.noPadding()
		f.formatArgumentDefinitionList(def.Args)
	}

	if len(def.Locations) != 0 {
		f.writeWord("on")

		for idx, dirLoc := range def.Locations {
			f.formatDirectiveLocation(dirLoc)

			if idx != len(def.Locations)-1 {
				f.writeWord("|")
			}
		}
	}

	f.writeNewline()
}

func (f *schemaFormatter) formatDefinition(def *definition, rootNames []string) {
	if !f.emitBuiltin && (IsBaseScalarName(def.Name) || strings.HasPrefix(def.Name, "__")) {
		return
	}

	if slices.Contains(rootNames, def.Name) {
		def.Name = utils.UppercaseFirst(def.Name)
	}

	f.writeDescription(def.Description)

	switch def.Kind {
	case ast.Scalar:
		f.writeWord("scalar").writeWord(def.Name)
	case ast.Object:
		f.writeWord("type").writeWord(def.Name)
	case ast.Interface:
		f.writeWord("interface").writeWord(def.Name)
	case ast.Union:
		f.writeWord("union").writeWord(def.Name)
	case ast.Enum:
		f.writeWord("enum").writeWord(def.Name)
	case ast.InputObject:
		f.writeWord("input").writeWord(def.Name)
	}

	if len(def.Interfaces) != 0 {
		var interfaceNames []string
		for _, item := range def.Interfaces {
			interfaceNames = append(interfaceNames, item.Name)
		}
		f.writeWord("implements").writeWord(strings.Join(interfaceNames, " & "))
	}

	if len(def.PossibleTypes) != 0 {
		var typeNames []string
		for _, item := range def.PossibleTypes {
			if item.Name == nil {
				continue
			}

			typeNames = append(typeNames, item.prototype().string())
		}
		f.writeWord("=").writeWord(strings.Join(typeNames, " | "))
	}

	f.formatFieldList(def.Fields)

	f.formatInputFieldList(def.InputFields)

	f.formatEnumValueList(def.EnumValues)

	f.writeNewline()
}

func (f *schemaFormatter) formatEnumValueList(lists []*enumValueDefinition) {
	if len(lists) == 0 {
		return
	}

	f.WriteString("{").writeNewline()
	f.incrementIndent()

	for _, v := range lists {
		f.formatEnumValueDefinition(v)
	}

	f.decrementIndent()
	f.WriteString("}")
}

func (f *schemaFormatter) formatEnumValueDefinition(def *enumValueDefinition) {
	f.writeDescription(def.Description)

	f.writeWord(def.Name)

	f.writeNewline()
}

func (f *schemaFormatter) formatValue(value any) {
	f.WriteString(fmt.Sprintf("%v", value))
}

func (f *schemaFormatter) formatType(t *valueDefinition) {
	f.writeWord(t.string())
}

func (v *valueDefinition) prototype() *valueDefinition {
	if v.OfType == nil {
		return v
	}

	return v.OfType.prototype()
}

func (v *valueDefinition) depth() (depth int) {
	if v == nil {
		return
	}

	depth++
	if v.OfType == nil {
		return
	}

	depth += v.OfType.depth()
	return
}

func (v *valueDefinition) string() (result string) {
	if v == nil {
		return
	}

	if v.OfType != nil {
		result += v.OfType.string()
	}
	switch v.Kind {
	case kindList:
		result = fmt.Sprintf(kindListFormat, result)
	case KindNonNull:
		result += "!"
	default:
		result += *v.Name
	}

	return
}

func (v *valueDefinition) valueType() jsonparser.ValueType {
	if v == nil {
		return jsonparser.Null
	}
	if v.Kind == kindList {
		return jsonparser.Array
	}
	if v.OfType != nil {
		return v.OfType.valueType()
	}

	switch *v.Name {
	case consts.ScalarBoolean:
		return jsonparser.Boolean
	case consts.ScalarInt, consts.ScalarFloat:
		return jsonparser.Number
	case consts.ScalarString, consts.ScalarID, consts.ScalarDate, consts.ScalarDateTime, consts.ScalarUUID,
		consts.ScalarBytes, consts.ScalarBinary, consts.ScalarBigInt, consts.ScalarDecimal, consts.ScalarGeometry:
		return jsonparser.String
	default:
		return jsonparser.Object
	}
}

func (f *schemaFormatter) writeString(s string) {
	_, _ = f.writer.Write([]byte(s))
}

func (f *schemaFormatter) writeIndent() *schemaFormatter {
	if f.lineHead {
		f.writeString(strings.Repeat(f.indent, f.indentSize))
	}
	f.lineHead = false
	f.padNext = false

	return f
}

func (f *schemaFormatter) writeNewline() *schemaFormatter {
	f.writeString("\n")
	f.lineHead = true
	f.padNext = false

	return f
}

func (f *schemaFormatter) writeWord(word string) *schemaFormatter {
	if f.lineHead {
		f.writeIndent()
	}
	if f.padNext {
		f.writeString(" ")
	}
	f.writeString(strings.TrimSpace(word))
	f.padNext = true

	return f
}

func (f *schemaFormatter) WriteString(s string) *schemaFormatter {
	if f.lineHead {
		f.writeIndent()
	}
	if f.padNext {
		f.writeString(" ")
	}
	f.writeString(s)
	f.padNext = false

	return f
}

func (f *schemaFormatter) writeDescription(s string) *schemaFormatter {
	if s == "" {
		return f
	}

	f.WriteString(`"""`)
	if ss := strings.Split(s, "\n"); len(ss) > 1 {
		f.writeNewline()
		for _, s := range ss {
			f.WriteString(s).writeNewline()
		}
	} else {
		f.WriteString(s)
	}

	f.WriteString(`"""`).writeNewline()

	return f
}

func (f *schemaFormatter) incrementIndent() {
	f.indentSize++
}

func (f *schemaFormatter) decrementIndent() {
	f.indentSize--
}

func (f *schemaFormatter) noPadding() *schemaFormatter {
	f.padNext = false

	return f
}

func (f *schemaFormatter) needPadding() *schemaFormatter {
	f.padNext = true

	return f
}

func IsRootDefinition(name string) bool {
	return slices.Contains(RootObjectNames, name)
}

func IsBaseScalarName(name string) bool {
	return slices.Contains(ignoreDefinitionNames, name)
}

func IsScalarJsonName(name string) bool {
	return strings.EqualFold(name, consts.ScalarJSON)
}
