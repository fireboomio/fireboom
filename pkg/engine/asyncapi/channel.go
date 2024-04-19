package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	Channels   map[string]*ChannelRef
	ChannelRef struct {
		Ref   string
		Value *Channel
	}
	Channel struct {
		// An optional string representation of this channel's address. The address is typically the "topic name", "routing key", "event type", or "path". When null or absent, it MUST be interpreted as unknown. This is useful when the address is generated dynamically at runtime or can't be known upfront. It MAY contain Channel Address Expressions. Query parameters and fragments SHALL NOT be used, instead use bindings to define them.
		Address string `json:"address" yaml:"address"`
		// A map of the messages that will be sent to this channel by any application at any time. Every message sent to this channel MUST be valid against one, and only one, of the message objects defined in this map.
		Messages map[string]*MessageRef `json:"messages" yaml:"messages"`
		// A human-friendly title for the channel.
		Title string `json:"title" yaml:"title"`
		// A short summary of the channel.
		Summary string `json:"summary" yaml:"summary"`
		// An optional description of this channel. CommonMark syntax can be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// An array of $ref pointers to the definition of the servers in which this channel is available. If the channel is located in the root Channels Object, it MUST point to a subset of server definitions located in the root Servers Object, and MUST NOT point to a subset of server definitions located in the Components Object or anywhere else. If the channel is located in the Components Object, it MAY point to a Server Objects in any location. If servers is absent or empty, this channel MUST be available on all the servers defined in the Servers Object. Please note the servers property value MUST be an array of Reference Objects and, therefore, MUST NOT contain an array of Server Objects. However, it is RECOMMENDED that parsers (or other software) dereference this property for a better development experience.
		Servers Servers `json:"servers" yaml:"servers"`
		// A map of the parameters included in the channel address. It MUST be present only when the address contains Channel Address Expressions.
		Parameters Parameters `json:"parameters" yaml:"parameters"`
		// A list of tags for logical grouping of channels.
		Tags Tags `json:"tags" yaml:"tags"`
		// Additional external documentation for this channel.
		ExternalDocs *ExtensionDocsRef `json:"externalDocs" yaml:"externalDocs"`
		// A map where the keys describe the name of the protocol and the values describe protocol-specific definitions for the channel.
		Bindings ChannelBindings `json:"bindings" yaml:"bindings"`
	}
)

// MarshalYAML returns the YAML encoding of ChannelRef.
func (x ChannelRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of ChannelRef.
func (x ChannelRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets ChannelRef to a copy of data.
func (x *ChannelRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ChannelRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
