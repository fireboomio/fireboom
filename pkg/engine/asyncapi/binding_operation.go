package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	OperationBindings   map[BindingKey]*OperationBindingRef
	OperationBindingRef struct {
		Ref   string
		Value *OperationBinding
	}
	OperationBinding struct {
		OperationBindingAmqp
		OperationBindingKafka
		OperationBindingMqtt
		OperationBindingHttp
	}
)

// MarshalYAML returns the YAML encoding of OperationBindingRef.
func (x OperationBindingRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of OperationBindingRef.
func (x OperationBindingRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets OperationBindingRef to a copy of data.
func (x *OperationBindingRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *OperationBindingRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type OperationBindingAmqp struct {
	// TTL (Time-To-Live) for the message. It MUST be greater than or equal to zero.
	Expiration int `json:"expiration" yaml:"expiration"`
	// Identifies the user who has sent the message.
	UserId string `json:"userId" yaml:"userId"`
	// The routing keys the message should be routed to at the time of publishing.
	Cc []string `json:"cc" yaml:"cc"`
	// A priority for the message.
	Priority string `json:"priority" yaml:"priority"`
	// Delivery mode of the message. Its value MUST be either 1 (transient) or 2 (persistent).
	DeliveryMode OperationBindingAmqpDeliveryMode `json:"deliveryMode" yaml:"deliveryMode"`
	// Whether the message is mandatory or not.
	Mandatory bool `json:"mandatory" yaml:"mandatory"`
	// Like cc but consumers will not receive this information.
	Bcc []string `json:"bcc" yaml:"bcc"`
	// Whether the message should include a timestamp or not.
	Timestamp bool `json:"timestamp" yaml:"timestamp"`
	// Whether the consumer should ack the message or not.
	Ack bool `json:"ack" yaml:"ack"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type OperationBindingAmqpDeliveryMode int

const (
	OperationBindingAmqpDeliveryMode_transient  OperationBindingAmqpDeliveryMode = 1
	OperationBindingAmqpDeliveryMode_persistent OperationBindingAmqpDeliveryMode = 2
)

type OperationBindingKafka struct {
	// Id of the consumer group.
	GroupId *openapi3.SchemaRef `json:"groupId" yaml:"groupId"`
	// Id of the consumer inside a consumer group.
	ClientId *openapi3.SchemaRef `json:"clientId" yaml:"clientId"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion" json:"bindingVersion" yaml:"bindingVersion"`
}

type OperationBindingMqtt struct {
	// Defines the Quality of Service (QoS) levels for the message flow between client and server. Its value MUST be either 0 (At most once delivery), 1 (At least once delivery), or 2 (Exactly once delivery).
	Qos int `json:"qos" yaml:"qos"`
	// Whether the broker should retain the message or not.
	Retain bool `json:"retain" yaml:"retain"`
	// Interval in seconds or a Schema Object containing the definition of the lifetime of the message.
	MessageExpiryInterval int `json:"messageExpiryInterval" yaml:"messageExpiryInterval"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type OperationBindingHttp struct {
	// The HTTP method to use when establishing the connection. Its value MUST be either GET or POST.
	Method string `json:"method" yaml:"method"`
	// A Schema object containing the definitions for each query parameter. This schema MUST be of type object and have a properties key.
	Query *openapi3.SchemaRef `json:"query" yaml:"query"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}
