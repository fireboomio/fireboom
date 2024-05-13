// Package directives
/*
 实现VariableDirective接口，只能定义在LocationVariableDefinition上
 Resolve 按照引擎需要的配置定义VariablesConfiguration.InjectVariables，留作后续发送graphql前填充当前时间参数
*/
package directives

import (
	"fireboom-server/pkg/common/consts"
	"fireboom-server/pkg/plugins/i18n"
	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wundergraph/wundergraph/pkg/wgpb"
	"time"
)

const (
	dateTimeArgFormat         = "format"
	dateTimeArgFormatEnumName = "EngineDateTimeFormat"
	dateTimeArgCustomFormat   = "customFormat"
	injectCurrentDateTimeName = "injectCurrentDateTime"
	dateTimeArgOffsetName     = "offset"
	dateTimeArgOffsetType     = "DateOffset"
	dateTimeArgOffsetUnitType = "DateOffsetUnit"
	datetimeArgOffsetUnitName = "unit"
)

type injectCurrentDateTime struct{}

func (v *injectCurrentDateTime) Directive() *ast.DirectiveDefinition {
	formatArgs := dateTimeFormatArguments()
	formatArgs = append(formatArgs, &ast.ArgumentDefinition{
		Name: dateTimeArgOffsetName,
		Type: ast.NamedType(dateTimeArgOffsetType, nil),
	})
	return &ast.DirectiveDefinition{
		Description: appendIfExistExampleGraphql(i18n.InjectCurrentDateTimeDesc.String()),
		Name:        injectCurrentDateTimeName,
		Locations:   []ast.DirectiveLocation{ast.LocationVariableDefinition},
		Arguments:   formatArgs,
	}
}

func (v *injectCurrentDateTime) Definitions() ast.DefinitionList {
	definitions := dateTimeFormatDefinitions()
	var unitEnumValues ast.EnumValueList
	for k := range wgpb.DateOffsetUnit_value {
		unitEnumValues = append(unitEnumValues, &ast.EnumValueDefinition{Name: k})
	}
	definitions = append(definitions,
		&ast.Definition{
			Kind: ast.InputObject,
			Name: dateTimeArgOffsetType,
			Fields: ast.FieldList{
				{Name: "previous", Type: ast.NamedType(consts.ScalarBoolean, nil)},
				{Name: "value", Type: ast.NonNullNamedType(consts.ScalarInt, nil)},
				{Name: datetimeArgOffsetUnitName, Type: ast.NonNullNamedType(dateTimeArgOffsetUnitType, nil)},
			},
		}, &ast.Definition{
			Kind:       ast.Enum,
			Name:       dateTimeArgOffsetUnitType,
			EnumValues: unitEnumValues,
		})
	return definitions
}

func (v *injectCurrentDateTime) Resolve(resolver *VariableResolver) (_, skip bool, err error) {
	dateFormat := dateFormatArgValue(resolver.Arguments)
	if len(dateFormat) == 0 {
		dateFormat = dateFormatMap[ISO8601]
	}
	variableConfig := &wgpb.VariableInjectionConfiguration{
		VariablePathComponents: resolver.Path,
		DateFormat:             dateFormat,
		VariableKind:           wgpb.InjectVariableKind_DATE_TIME,
	}
	if dateOffset, ok := resolver.Arguments[dateTimeArgOffsetName]; ok {
		unit := gjson.Get(dateOffset, datetimeArgOffsetUnitName).String()
		dateOffset, _ = sjson.Set(dateOffset, datetimeArgOffsetUnitName, wgpb.DateOffsetUnit_value[unit])
		if err = json.Unmarshal([]byte(dateOffset), &variableConfig.DateOffset); err != nil {
			return
		}
	}

	resolver.Operation.VariablesConfiguration.InjectVariables = append(resolver.Operation.VariablesConfiguration.InjectVariables, variableConfig)
	skip = true
	return
}

func dateFormatArgValue(argMap map[string]string) (dateFormat string) {
	if value, ok := argMap[dateTimeArgFormat]; ok {
		dateFormat = dateFormatMap[value]
	}
	if value, ok := argMap[dateTimeArgCustomFormat]; ok {
		dateFormat = value
	}
	return
}

func dateTimeFormatArguments() ast.ArgumentDefinitionList {
	return ast.ArgumentDefinitionList{
		{
			Description:  i18n.FormatDateTimeArgFormatDesc.String(),
			Name:         dateTimeArgFormat,
			Type:         ast.NamedType(dateTimeArgFormatEnumName, nil),
			DefaultValue: &ast.Value{Kind: ast.EnumValue, Raw: ISO8601},
		},
		{
			Description: i18n.FormatDateTimeArgCustomFormatDesc.String(),
			Name:        dateTimeArgCustomFormat,
			Type:        ast.NamedType(consts.ScalarString, nil),
		},
	}
}

func dateTimeFormatDefinitions() ast.DefinitionList {
	var enumValues ast.EnumValueList
	for k := range dateFormatMap {
		enumValues = append(enumValues, &ast.EnumValueDefinition{Name: k})
	}
	return ast.DefinitionList{{
		Kind:       ast.Enum,
		Name:       dateTimeArgFormatEnumName,
		EnumValues: enumValues,
	}}
}

const (
	ISO8601     = "ISO8601"
	ANSIC       = "ANSIC"
	UnixDate    = "UnixDate"
	RubyDate    = "RubyDate"
	RFC822      = "RFC822"
	RFC822Z     = "RFC822Z"
	RFC850      = "RFC850"
	RFC1123     = "RFC1123"
	RFC1123Z    = "RFC1123Z"
	RFC3339     = "RFC3339"
	RFC3339Nano = "RFC3339Nano"
	Kitchen     = "Kitchen"
	Stamp       = "Stamp"
	StampMilli  = "StampMilli"
	StampMicro  = "StampMicro"
	StampNano   = "StampNano"
	DateTime    = "DateTime"
	DateOnly    = "DateOnly"
	TimeOnly    = "TimeOnly"
)

var dateFormatMap map[string]string

func init() {
	registerDirective(injectCurrentDateTimeName, &injectCurrentDateTime{})

	dateFormatMap = make(map[string]string)
	dateFormatMap[ISO8601] = time.RFC3339
	dateFormatMap[ANSIC] = time.ANSIC
	dateFormatMap[UnixDate] = time.UnixDate
	dateFormatMap[RubyDate] = time.RubyDate
	dateFormatMap[RFC822] = time.RFC822
	dateFormatMap[RFC822Z] = time.RFC822Z
	dateFormatMap[RFC850] = time.RFC850
	dateFormatMap[RFC1123] = time.RFC1123
	dateFormatMap[RFC1123Z] = time.RFC1123Z
	dateFormatMap[RFC3339] = time.RFC3339
	dateFormatMap[RFC3339Nano] = time.RFC3339Nano
	dateFormatMap[Kitchen] = time.Kitchen
	dateFormatMap[Stamp] = time.Stamp
	dateFormatMap[StampMilli] = time.StampMilli
	dateFormatMap[StampMicro] = time.StampMicro
	dateFormatMap[StampNano] = time.StampNano
	dateFormatMap[DateTime] = time.DateTime
	dateFormatMap[DateOnly] = time.DateOnly
	dateFormatMap[TimeOnly] = time.TimeOnly
}
