package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	ChannelBindings   map[BindingKey]*ChannelBindingRef
	ChannelBindingRef struct {
		Ref   string
		Value *ChannelBinding
	}
	ChannelBinding struct {
		ChannelBindingAmqp
		ChannelBindingKafka
		ChannelBindingWebSocket
	}
)

// MarshalYAML returns the YAML encoding of ChannelBindingRef.
func (x ChannelBindingRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of ChannelBindingRef.
func (x ChannelBindingRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets ChannelBindingRef to a copy of data.
func (x *ChannelBindingRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ChannelBindingRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type (
	ChannelBindingAmqp struct {
		// Defines what type of channel is it. Can be either queue or routingKey (default).
		Is ChannelBindingAmqpIs `json:"is" yaml:"is"`
		// When is=routingKey, this object defines the exchange properties.
		Exchange *ChannelBindingAmqpExchange `json:"exchange" yaml:"exchange"`
		// When is=queue, this object defines the queue properties.
		Queue *ChannelBindingAmqpQueue `json:"queue" yaml:"queue"`
		// The version of this binding. If omitted, "latest" MUST be assumed.
		BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
	}
	ChannelBindingAmqpExchange struct {
		// The name of the exchange. It MUST NOT exceed 255 characters long.
		Name string `json:"name" yaml:"name"`
		// The type of the exchange. Can be either topic, direct, fanout, default or headers.
		Type ChannelBindingAmqpExchangeType `json:"type" yaml:"type"`
		// Whether the exchange should survive broker restarts or not.
		Durable bool `json:"durable" yaml:"durable"`
		// Whether the exchange should be deleted when the last queue is unbound from it.
		AutoDelete bool `json:"autoDelete" yaml:"autoDelete"`
		// The virtual host of the exchange. Defaults to /.
		Vhost string `json:"vhost" yaml:"vhost"`
	}
	ChannelBindingAmqpQueue struct {
		// The name of the queue. It MUST NOT exceed 255 characters long.
		Name string `json:"name" yaml:"name"`
		// Whether the queue should survive broker restarts or not.
		Durable bool `json:"durable" yaml:"durable"`
		// Whether the queue should be used only by one connection or not.
		Exclusive bool `json:"exclusive" yaml:"exclusive"`
		// Whether the queue should be deleted when the last consumer unsubscribes.
		AutoDelete bool `json:"autoDelete" yaml:"autoDelete"`
		// The virtual host of the queue. Defaults to /.
		Vhost string `json:"vhost" yaml:"vhost"`
	}
)

type ChannelBindingAmqpIs string

const (
	ChannelBindingAmqpIs_queue      ChannelBindingAmqpIs = "queue"
	ChannelBindingAmqpIs_routingKey ChannelBindingAmqpIs = "routingKey"
)

type ChannelBindingAmqpExchangeType string

const (
	ChannelBindingAmqpExchangeType_topic   ChannelBindingAmqpExchangeType = "topic"
	ChannelBindingAmqpExchangeType_direct  ChannelBindingAmqpExchangeType = "direct"
	ChannelBindingAmqpExchangeType_fanout  ChannelBindingAmqpExchangeType = "fanout"
	ChannelBindingAmqpExchangeType_default ChannelBindingAmqpExchangeType = "default"
	ChannelBindingAmqpExchangeType_headers ChannelBindingAmqpExchangeType = "headers"
)

type (
	ChannelBindingKafka struct {
		// Kafka topic name if different from channel name.
		Topic string `json:"topic"`
		// Number of partitions configured on this topic (useful to know how many parallel consumers you may run).
		Partitions int `json:"partitions"`
		// Number of replicas configured on this topic.
		Replicas int `json:"replicas"`
		// Topic configuration properties that are relevant for the API.
		TopicConfiguration *ChannelBindingKafkaTopicConfiguration `json:"topicConfiguration"`
		// The version of this binding. If omitted, "latest" MUST be assumed.
		BindingVersion string `json:"bindingVersion"`
	}
	ChannelBindingKafkaTopicConfiguration struct {
		// The cleanup.policy configuration option.
		CleanupPolicy []ChannelBindingKafkaTopicConfigurationCleanupPolicy `json:"cleanup.policy" yaml:"cleanup.policy"`
		// The retention.ms configuration option.
		RetentionMs int64 `json:"retention.ms" yaml:"retention.ms"`
		// The retention.bytes configuration option.
		RetentionBytes int64 `json:"retention.bytes" yaml:"retention.bytes"`
		// The delete.retention.ms configuration option.
		DeleteRetentionMs int64 `json:"delete.retention.ms" yaml:"delete.retention.ms"`
		// The max.message.bytes configuration option.
		MaxMessageBytes int `json:"max.message.bytes" yaml:"max.message.bytes"`
		// It shows whether the schema validation for the message key is enabled. Vendor specific config.
		ConfluentKeySchemaValidation bool `json:"confluent.key.schema.validation" yaml:"confluent.key.schema.validation"`
		// The name of the schema lookup strategy for the message key. Vendor specific config.
		ConfluentKeySubjectNameStrategy string `json:"confluent.key.subject.name.strategy" yaml:"confluent.key.subject.name.strategy"`
		// It shows whether the schema validation for the message value is enabled. Vendor specific config.
		ConfluentValueSchemaValidation bool `json:"confluent.value.schema.validation" yaml:"confluent.value.schema.validation"`
		// The name of the schema lookup strategy for the message value. Vendor specific config.
		ConfluentValueSubjectNameStrategy string `json:"confluent.value.subject.name.strategy" yaml:"confluent.value.subject.name.strategy"`
	}
)

type ChannelBindingKafkaTopicConfigurationCleanupPolicy string

const (
	ChannelBindingKafkaTopicConfigurationCleanupPolicy_delete  ChannelBindingKafkaTopicConfigurationCleanupPolicy = "delete"
	ChannelBindingKafkaTopicConfigurationCleanupPolicy_compact ChannelBindingKafkaTopicConfigurationCleanupPolicy = "compact"
)

type ChannelBindingWebSocket struct {
	// The HTTP method to use when establishing the connection. Its value MUST be either GET or POST.
	Method string `json:"method" yaml:"method"`
	// A Schema object containing the definitions for each query parameter. This schema MUST be of type object and have a properties key.
	Query *openapi3.SchemaRef `json:"query" yaml:"query"`
	// A Schema object containing the definitions of the HTTP headers to use when establishing the connection. This schema MUST be of type object and have a properties key.
	Headers *openapi3.SchemaRef `json:"headers" yaml:"headers"`
	// The version of this binding. If omitted, "latest" MUST be assumed.
	BindingVersion string `json:"bindingVersion" yaml:"bindingVersion"`
}
