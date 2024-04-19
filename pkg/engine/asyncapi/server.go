package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/jsonpointer"
	json "github.com/json-iterator/go"
)

type (
	Servers map[string]*Server
	Server  struct {
		// REQUIRED. The server host name. It MAY include the port. This field supports Server Variables. Variable substitutions will be made when a variable is named in {braces}.
		Host string `json:"host" yaml:"host"`
		// REQUIRED. The protocol this server supports for connection.
		Protocol BindingKey `json:"protocol" yaml:"protocol"`
		// The version of the protocol used for connection. For instance: AMQP 0.9.1, HTTP 2.0, Kafka 1.0.0, etc.
		ProtocolVersion string `json:"protocolVersion" yaml:"protocolVersion"`
		// The path to a resource in the host. This field supports Server Variables. Variable substitutions will be made when a variable is named in {braces}.
		Pathname string `json:"pathname" yaml:"pathname"`
		// An optional string describing the server. CommonMark syntax MAY be used for rich text representation.
		Description string `json:"description" yaml:"description"`
		// A human-friendly title for the server.
		Title string `json:"title" yaml:"title"`
		// A short summary of the server.
		Summary string `json:"summary" yaml:"summary"`
		// A map between a variable name and its value. The value is used for substitution in the server's host and pathname template.
		Variables ServerVariables `json:"variables" yaml:"variables"`
		// A declaration of which security schemes can be used with this server. The list of values includes alternative security scheme objects that can be used. Only one of the security scheme objects need to be satisfied to authorize a connection or operation.
		Security *SecuritySchemeRef `json:"security" yaml:"security"`
		// A list of tags for logical grouping and categorization of servers.
		Tags Tags `json:"tags" yaml:"tags"`
		// Additional external documentation for this server.
		ExternalDocs *ExtensionDocsRef `json:"externalDocs" yaml:"externalDocs"`
		// A map where the keys describe the name of the protocol and the values describe protocol-specific definitions for the server.
		Bindings ServerBindings `json:"bindings" yaml:"bindings"`
	}
)

type (
	ServerVariables   map[string]*ServerVariableRef
	ServerVariableRef struct {
		Ref   string
		Value *openapi3.ServerVariable
	}
)

// MarshalYAML returns the YAML encoding of ServerVariableRef.
func (x ServerVariableRef) MarshalYAML() (interface{}, error) {
	if ref := x.Ref; ref != "" {
		return &openapi3.Ref{Ref: ref}, nil
	}
	return x.Value, nil
}

// MarshalJSON returns the JSON encoding of ServerVariableRef.
func (x ServerVariableRef) MarshalJSON() ([]byte, error) {
	if ref := x.Ref; ref != "" {
		return json.Marshal(openapi3.Ref{Ref: ref})
	}
	return json.Marshal(x.Value)
}

// UnmarshalJSON sets ServerVariableRef to a copy of data.
func (x *ServerVariableRef) UnmarshalJSON(data []byte) error {
	var refOnly openapi3.Ref
	if err := json.Unmarshal(data, &refOnly); err == nil && refOnly.Ref != "" {
		x.Ref = refOnly.Ref
		return nil
	}
	return json.Unmarshal(data, &x.Value)
}

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (x *ServerVariableRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return x.Ref, nil
	}
	ptr, _, err := jsonpointer.GetForToken(x.Value, token)
	return ptr, err
}
