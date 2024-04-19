package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	MessageBindings   map[BindingKey]*MessageBindingRef
	MessageBindingRef struct {
		Ref   string
		Value *MessageBinding
	}
	MessageBinding struct {
		MessageBindingAmqp
		MessageBindingKafka
		MessageBindingMqtt
		MessageBindingHttp
	}
)

// MarshalYAML returns the YAML encoding of MessageBindingRef.
func (x MessageBindingRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of MessageBindingRef.
func (x MessageBindingRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets MessageBindingRef to a copy of data.
func (x *MessageBindingRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *MessageBindingRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type MessageBindingAmqp struct {
	// A MIME encoding for the message content.
	ContentEncoding string `json:"contentEncoding" yaml:"contentEncoding"`
	// Application-specific message type.
	MessageType string `json:"messageType" yaml:"messageType"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type MessageBindingKafka struct {
	// The message key. NOTE: You can also use the reference object way.
	Key *openapi3.SchemaRef `json:"key" yaml:"key"`
	// If a Schema Registry is used when performing this operation, tells where the id of schema is stored (e.g. header or payload).
	SchemaIdLocation string `json:"schemaIdLocation" yaml:"schemaIdLocation"`
	// Number of bytes or vendor specific values when schema id is encoded in payload (e.g confluent/ apicurio-legacy / apicurio-new).
	SchemaIdPayloadEncoding string `json:"schemaIdPayloadEncoding" yaml:"schemaIdPayloadEncoding"`
	// Freeform string for any naming strategy class to use. Clients should default to the vendor default if not supplied.
	SchemaLookupStrategy string `json:"schemaLookupStrategy" yaml:"schemaLookupStrategy"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type MessageBindingMqtt struct {
	// Either: 0 (zero): Indicates that the payload is unspecified bytes, or 1: Indicates that the payload is UTF-8 encoded character data.
	PayloadFormatIndicator int `json:"payloadFormatIndicator" yaml:"payloadFormatIndicator"`
	// Correlation Data is used by the sender of the request message to identify which request the response message is for when it is received.
	CorrelationData *openapi3.SchemaRef `json:"correlationData" yaml:"correlationData"`
	// String describing the content type of the message payload. This should not conflict with the contentType field of the associated AsyncAPI Message object.
	ContentType string `json:"contentType" yaml:"contentType"`
	// The topic (channel URI) for a response message.
	ResponseTopic string `json:"responseTopic" yaml:"responseTopic"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type MessageBindingHttp struct {
	// A Schema object containing the definitions of the HTTP headers to use when establishing the connection. This schema MUST be of type object and have a properties key.
	Headers *openapi3.SchemaRef `json:"headers" yaml:"headers"`
	// The HTTP response status code according to RFC 9110. statusCode is only relevant for messages referenced by the Operation Reply Object, as it defines the status code for the response. In all other cases, this value can be safely ignored.
	StatusCode int `json:"statusCode" yaml:"statusCode"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}
