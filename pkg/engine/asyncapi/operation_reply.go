package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	OperationReplyRef struct {
		Ref   string
		Value *OperationReply
	}
	OperationReply struct {
		// Definition of the address that implementations MUST use for the reply.
		Address *OperationReplyAddressRef `json:"address" yaml:"address"`
		// A $ref pointer to the definition of the channel in which this operation is performed. When address is specified, the address property of the channel referenced by this property MUST be either null or not defined. If the operation reply is located inside a root Operation Object, it MUST point to a channel definition located in the root Channels Object, and MUST NOT point to a channel definition located in the Components Object or anywhere else. If the operation reply is located inside an [Operation Object] in the Components Object or in the Replies Object in the Components Object, it MAY point to a Channel Object in any location. Please note the channel property value MUST be a Reference Object and, therefore, MUST NOT contain a Channel Object. However, it is RECOMMENDED that parsers (or other software) dereference this property for a better development experience.
		Channel *ChannelRef `json:"channel" yaml:"channel"`
		// A list of $ref pointers pointing to the supported Message Objects that can be processed by this operation as reply. It MUST contain a subset of the messages defined in the channel referenced in this operation reply, and MUST NOT point to a subset of message definitions located in the Components Object or anywhere else. Every message processed by this operation MUST be valid against one, and only one, of the message objects referenced in this list. Please note the messages property value MUST be a list of Reference Objects and, therefore, MUST NOT contain Message Objects. However, it is RECOMMENDED that parsers (or other software) dereference this property for a better development experience.
		Messages Messages `json:"messages" yaml:"messages"`
	}
)

// MarshalYAML returns the YAML encoding of OperationReplyRef.
func (x OperationReplyRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of OperationReplyRef.
func (x OperationReplyRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets OperationReplyRef to a copy of data.
func (x *OperationReplyRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *OperationReplyRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}

type (
	OperationReplyAddressRef struct {
		Ref   string
		Value *OperationReplyAddress
	}
	OperationReplyAddress struct {
		// An optional description of the address. CommonMark syntax can be used for rich text representation.
		Location string `json:"location" yaml:"location"`
		// REQUIRED. A runtime expression that specifies the location of the reply address.
		Description string `json:"description" yaml:"description"`
	}
)

// MarshalYAML returns the YAML encoding of OperationReplyAddressRef.
func (x OperationReplyAddressRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of OperationReplyAddressRef.
func (x OperationReplyAddressRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets OperationReplyAddressRef to a copy of data.
func (x *OperationReplyAddressRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *OperationReplyAddressRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
