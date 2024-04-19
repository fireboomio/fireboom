package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	OperationTraits   []*OperationTraitRef
	OperationTraitRef struct {
		Ref   string
		Value *OperationTrait
	}
	OperationTrait struct {
		// A human-friendly title for the operation.
		Title string `json:"title" yaml:"title"`
		// A short summary of what the operation is about.
		Summary string `json:"summary" yaml:"summary"`
		// A verbose explanation of the operation. CommonMark syntax can be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// A declaration of which security schemes are associated with this operation. Only one of the security scheme objects MUST be satisfied to authorize an operation. In cases where Server Security also applies, it MUST also be satisfied.
		Security *SecuritySchemeRef `json:"security" yaml:"security"`
		// A list of tags for logical grouping and categorization of operations.
		Tags Tags `json:"tags" yaml:"tags"`
		// Additional external documentation for this operation.
		ExternalDocs *ExtensionDocsRef `json:"externalDocs" yaml:"externalDocs"`
		// A map where the keys describe the name of the protocol and the values describe protocol-specific definitions for the operation.
		Bindings OperationBindings `json:"bindings" yaml:"bindings"`
	}
)

// MarshalYAML returns the YAML encoding of OperationTraitRef.
func (x OperationTraitRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of OperationTraitRef.
func (x OperationTraitRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets OperationTraitRef to a copy of data.
func (x *OperationTraitRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *OperationTraitRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
