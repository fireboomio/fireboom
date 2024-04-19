// Package consts
/*
 graphql常量
*/
package consts

// graphql role指令
const (
	RequireMatchAll = "requireMatchAll"
	RequireMatchAny = "requireMatchAny"
	DenyMatchAll    = "denyMatchAll"
	DenyMatchAny    = "denyMatchAny"
)

// graphql 根字段所属类型
const (
	TypeQuery        = "Query"
	TypeMutation     = "Mutation"
	TypeSubscription = "Subscription"
)

// graphql 普通类型定义
const (
	ScalarBoolean = "Boolean"
	ScalarInt     = "Int"
	ScalarFloat   = "Float"
	ScalarString  = "String"
	ScalarID      = "ID"
	ScalarJSON    = "JSON"
)

// graphql 复杂类型定义(通过stringFormat标识)
const (
	ScalarDate     = "Date"
	ScalarDateTime = "DateTime"
	ScalarBytes    = "Bytes"
	ScalarBinary   = "Binary"
	ScalarUUID     = "UUID"
	ScalarBigInt   = "BigInt"
	ScalarDecimal  = "Decimal"
	ScalarGeometry = "Geometry"
)
