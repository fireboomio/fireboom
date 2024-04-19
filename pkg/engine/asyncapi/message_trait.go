package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	MessageTraits   []*MessageTraitRef
	MessageTraitRef struct {
		Ref   string
		Value *MessageTrait
	}
	MessageTrait struct {
		// Schema definition of the application headers. Schema MUST be a map of key-value pairs. It MUST NOT define the protocol headers. If this is a Schema Object, then the schemaFormat will be assumed to be "application/vnd.aai.asyncapi+json;version=asyncapi" where the version is equal to the AsyncAPI Version String.
		Headers *MultiFormatSchemaRef `json:"headers" yaml:"headers"`
		// Definition of the correlation ID used for message tracing or matching.
		CorrelationId *CorrelationIdRef `json:"correlationId" yaml:"correlationId"`
		// The content type to use when encoding/decoding a message's payload. The value MUST be a specific media type (e.g. application/json). When omitted, the value MUST be the one specified on the defaultContentType field.
		ContentType string `json:"contentType" yaml:"contentType"`
		// A machine-friendly name for the message.
		Name string `json:"name" yaml:"name"`
		// A human-friendly title for the message.
		Title string `json:"title" yaml:"title"`
		// A short summary of what the message is about.
		Summary string `json:"summary" yaml:"summary"`
		// A verbose explanation of the message. CommonMark syntax can be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// A list of tags for logical grouping and categorization of messages.
		Tags Tags `json:"tags" yaml:"tags"`
		// Additional external documentation for this message.
		ExternalDocs *ExtensionDocsRef `json:"externalDocs" yaml:"externalDocs"`
		// A map where the keys describe the name of the protocol and the values describe protocol-specific definitions for the message.
		Bindings MessageBindings `json:"bindings" yaml:"bindings"`
	}
)

// MarshalYAML returns the YAML encoding of MessageTraitRef.
func (x MessageTraitRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of MessageTraitRef.
func (x MessageTraitRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets MessageTraitRef to a copy of data.
func (x *MessageTraitRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *MessageTraitRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
