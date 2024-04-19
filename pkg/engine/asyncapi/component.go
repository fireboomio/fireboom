package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type Components struct {
	// An object to hold reusable Schema Object. If this is a Schema Object, then the schemaFormat will be assumed to be "application/vnd.aai.asyncapi+json;version=asyncapi" where the version is equal to the AsyncAPI Version String.
	Schemas map[string]*MultiFormatSchemaRef `json:"schemas" yaml:"schemas"`
	// An object to hold reusable Server Objects.
	Servers Servers `json:"servers" yaml:"servers"`
	// An object to hold reusable Channel Objects.
	Channels Channels `json:"channels" yaml:"channels"`
	// An object to hold reusable Operation Objects.
	Operations Operations `json:"operations" yaml:"operations"`
	// An object to hold reusable Message Objects.
	Messages map[string]*MessageRef `json:"messages" yaml:"messages"`
	// An object to hold reusable Security Scheme Objects.
	SecuritySchemes SecuritySchemes `json:"securitySchemes" yaml:"securitySchemes"`
	// An object to hold reusable Server Variable Objects.
	ServerVariables ServerVariables `json:"serverVariables" yaml:"serverVariables"`
	// An object to hold reusable Parameter Objects.
	Parameters Parameters `json:"parameters" yaml:"parameters"`
	// An object to hold reusable Correlation ID Objects.
	CorrelationIds map[string]*CorrelationIdRef `json:"correlationIds" yaml:"correlationIds"`
	// An object to hold reusable Operation Reply Objects.
	Replies map[string]*OperationReplyRef `json:"replies" yaml:"replies"`
	// An object to hold reusable Operation Reply Address Objects.
	ReplyAddresses map[string]*OperationReplyAddressRef `json:"replyAddresses" yaml:"replyAddresses"`
	// An object to hold reusable External Documentation Objects.
	ExternalDocs map[string]*ExtensionDocsRef `json:"externalDocs" yaml:"externalDocs"`
	// An object to hold reusable Tag Objects.
	Tags map[string]*TagRef `json:"tags" yaml:"tags"`
	// An object to hold reusable Operation Trait Objects.
	OperationTraits map[string]*OperationTraitRef `json:"operationTraits" yaml:"operationTraits"`
	// An object to hold reusable Message Trait Objects.
	MessageTraits map[string]*MessageTraitRef `json:"messageTraits" yaml:"messageTraits"`
	// An object to hold reusable Server Bindings Objects.
	ServerBindings ServerBindings `json:"serverBindings" yaml:"serverBindings"`
	// An object to hold reusable Channel Bindings Objects.
	ChannelBindings ChannelBindings `json:"channelBindings" yaml:"channelBindings"`
	// An object to hold reusable Operation Bindings Objects.
	OperationBindings OperationBindings `json:"operationBindings" yaml:"operationBindings"`
	// An object to hold reusable Message Bindings Objects.
	MessageBindings MessageBindings `json:"messageBindings" yaml:"messageBindings"`
}

type (
	CorrelationIdRef struct {
		Ref   string
		Value *CorrelationId
	}
	CorrelationId struct {
		// An optional description of the identifier. CommonMark syntax can be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// REQUIRED. A runtime expression that specifies the location of the correlation ID.
		Location string `json:"location" yaml:"location"`
	}
)

// MarshalYAML returns the YAML encoding of CorrelationIdRef.
func (x CorrelationIdRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of CorrelationIdRef.
func (x CorrelationIdRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets CorrelationIdRef to a copy of data.
func (x *CorrelationIdRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *CorrelationIdRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type (
	MultiFormatSchemaRef struct {
		Ref   string
		Value *MultiFormatSchema
	}
	MultiFormatSchema struct {
		// Required. A string containing the name of the schema format that is used to define the information. If schemaFormat is missing, it MUST default to application/vnd.aai.asyncapi+json;version={{asyncapi}} where {{asyncapi}} matches the AsyncAPI Version String. In such a case, this would make the Multi Format Schema Object equivalent to the Schema Object. When using Reference Object within the schema, the schemaFormat of the resource being referenced MUST match the schemaFormat of the schema that contains the initial reference. For example, if you reference Avro schema, then schemaFormat of referencing resource and the resource being reference MUST match.
		SchemaFormat string `json:"schemaFormat" yaml:"schemaFormat"`
		// Required. Definition of the message payload. It can be of any type but defaults to Schema Object. It MUST match the schema format defined in schemaFormat, including the encoding type. E.g., Avro should be inlined as either a YAML or JSON object instead of as a string to be parsed as YAML or JSON. Non-JSON-based schemas (e.g., Protobuf or XSD) MUST be inlined as a string.
		Schema *openapi3.Schema `json:"schema" yaml:"schema"`
	}
)

// MarshalYAML returns the YAML encoding of SchemaRef.
func (x MultiFormatSchemaRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of SchemaRef.
func (x MultiFormatSchemaRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets SchemaRef to a copy of data.
func (x *MultiFormatSchemaRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}

	x.Value = &MultiFormatSchema{}
	if err := json.Unmarshal(data, &x.Value); err == nil && x.Value.Schema != nil {
		return nil
	}

	return json.Unmarshal(data, &x.Value.Schema)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *MultiFormatSchemaRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type ExtensionDocsRef struct {
	Ref   string
	Value *openapi3.ExternalDocs
}

// MarshalYAML returns the YAML encoding of SchemaRef.
func (x ExtensionDocsRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of SchemaRef.
func (x ExtensionDocsRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets SchemaRef to a copy of data.
func (x *ExtensionDocsRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ExtensionDocsRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type (
	Tags   []*TagRef
	TagRef struct {
		Ref   string
		Value *openapi3.Tag
	}
)

// MarshalYAML returns the YAML encoding of SchemaRef.
func (x TagRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of SchemaRef.
func (x TagRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets SchemaRef to a copy of data.
func (x *TagRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *TagRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type (
	Parameters   map[string]*ParameterRef
	ParameterRef struct {
		Ref   string
		Value *Parameter
	}
	Parameter struct {
		// An enumeration of string values to be used if the substitution options are from a limited set.
		Enum []string `json:"enum" yaml:"enum"`
		// The default value to use for substitution, and to send, if an alternate value is not supplied.
		Default string `json:"default" yaml:"default"`
		// An optional description for the parameter. CommonMark syntax MAY be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// An array of examples of the parameter value.
		Examples []string `json:"examples" yaml:"examples"`
		// A runtime expression that specifies the location of the parameter value.
		Location string `json:"location" yaml:"location"`
	}
)

// MarshalYAML returns the YAML encoding of SchemaRef.
func (x ParameterRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of SchemaRef.
func (x ParameterRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets SchemaRef to a copy of data.
func (x *ParameterRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ParameterRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
