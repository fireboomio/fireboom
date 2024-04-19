package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	ServerBindings   map[BindingKey]*ServerBindingRef
	ServerBindingRef struct {
		Ref   string
		Value *ServerBinding
	}
	ServerBinding struct {
		ServerBindingKafka
		ServerBindingMqtt
	}
)

// MarshalYAML returns the YAML encoding of ServerBindingRef.
func (x ServerBindingRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of ServerBindingRef.
func (x ServerBindingRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets ServerBindingRef to a copy of data.
func (x *ServerBindingRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ServerBindingRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type ServerBindingKafka struct {
	// API URL for the Schema Registry used when producing Kafka messages (if a Schema Registry was used)
	SchemaRegistryUrl string `json:"schemaRegistryUrl" yaml:"schemaRegistryUrl"`
	// The vendor of Schema Registry and Kafka serdes library that should be used (e.g. apicurio, confluent, ibm, or karapace)
	SchemaRegistryVendor string `json:"schemaRegistryVendor" yaml:"schemaRegistryVendor"`
	// The version of this binding.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}

type (
	ServerBindingMqtt struct {
		// The client identifier.
		ClientId string `json:"clientId" yaml:"clientId"`
		// Whether to create a persistent connection or not. When false, the connection will be persistent. This is called clean start in MQTTv5.
		CleanSession bool `json:"cleanSession" yaml:"cleanSession"`
		// Last Will and Testament configuration. topic, qos, message and retain are properties of this object as shown below.
		LastWill *ServerBindingMqttLastWill `json:"lastWill" yaml:"lastWill"`
		// Interval in seconds of the longest period of time the broker and the client can endure without sending a message.
		KeepAlive int `json:"keepAlive" yaml:"keepAlive"`
		// Interval in seconds or a Schema Object containing the definition of the interval. The broker maintains a session for a disconnected client until this interval expires.
		SessionExpiryInterval int `json:"sessionExpiryInterval" yaml:"sessionExpiryInterval"`
		// Number of bytes or a Schema Object representing the maximum packet size the client is willing to accept.
		MaximumPacketSize int `json:"maximumPacketSize" yaml:"maximumPacketSize"`
		// The version of this binding. If omitted, "latest" MUST be assumed.
		BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
	}
	ServerBindingMqttLastWill struct {
		// The topic where the Last Will and Testament message will be sent.
		Topic string `json:"topic" yaml:"topic"`
		// Defines how hard the broker/client will try to ensure that the Last Will and Testament message is received. Its value MUST be either 0, 1 or 2.
		Qos int `json:"qos" yaml:"qos"`
		// Last Will message.
		Message string `json:"message" yaml:"message"`
		// Whether the broker should retain the Last Will and Testament message or not.
		Retain bool `json:"retain" yaml:"retain"`
	}
)
