package asyncapi

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type Spec struct {
	// REQUIRED. Specifies the AsyncAPI Specification version being used. It can be used by tooling Specifications and clients to interpret the version. The structure shall be major.minor.patch, where patch versions must be compatible with the existing major.minor tooling. Typically patch versions will be introduced to address errors in the documentation, and tooling should typically be compatible with the corresponding major.minor (1.0.*). Patch versions will correspond to patches of this document.
	Asyncapi string `json:"asyncapi" yaml:"asyncapi"`
	// Identifier of the application the AsyncAPI document is defining.
	Id string `json:"id" yaml:"id"`
	// REQUIRED. Provides metadata about the API. The metadata can be used by the clients if needed.
	Info openapi3.Info `json:"info" yaml:"info"`
	// Provides connection details of servers.
	Servers Servers `json:"servers" yaml:"servers"`
	// Default content type to use when encoding/decoding a message's payload.
	DefaultContentType string `json:"defaultContentType" yaml:"defaultContentType"`
	// The channels used by this application.
	Channels Channels `json:"channels" yaml:"channels"`
	// The operations this application MUST implement.
	Operations Operations `json:"operations" yaml:"operations"`
	// An element to hold various reusable objects for the specification. Everything that is defined inside this object represents a resource that MAY or MAY NOT be used in the rest of the document and MAY or MAY NOT be used by the implemented Application.
	Components *Components `json:"components" yaml:"components"`
}

func (s *Spec) Validate() error {
	return nil
}

func (s *Spec) ResolveRefsIn() (err error) {
	if components := s.Components; components != nil {
		for _, component := range components.Schemas {
			if err = s.resolveMultiFormatSchemaRef(component); err != nil {
				return
			}
		}
		for _, component := range components.Channels {
			if err = s.resolveChannelRef(component); err != nil {
				return
			}
		}
		for _, component := range components.Messages {
			if err = s.resolveMessageRef(component); err != nil {
				return
			}
		}
		for _, component := range components.SecuritySchemes {
			if err = s.resolveSecuritySchemeRef(component); err != nil {
				return
			}
		}
		for _, component := range components.ServerVariables {
			if err = s.resolveServerVariableRef(component); err != nil {
				return
			}
		}
		for _, component := range components.Parameters {
			if err = s.resolveParameterRef(component); err != nil {
				return
			}
		}
		for _, component := range components.CorrelationIds {
			if err = s.resolveCorrelationIdRef(component); err != nil {
				return
			}
		}
		for _, component := range components.Replies {
			if err = s.resolveOperationReplyRef(component); err != nil {
				return
			}
		}
		for _, component := range components.ReplyAddresses {
			if err = s.resolveOperationReplyAddressRef(component); err != nil {
				return
			}
		}
		for _, component := range components.ExternalDocs {
			if err = s.resolveExtensionDocsRef(component); err != nil {
				return
			}
		}
		for _, component := range components.Tags {
			if err = s.resolveTagRef(component); err != nil {
				return
			}
		}
		for _, component := range components.OperationTraits {
			if err = s.resolveOperationTraitRef(component); err != nil {
				return
			}
		}
		for _, component := range components.MessageTraits {
			if err = s.resolveMessageTraitRef(component); err != nil {
				return
			}
		}
		for _, component := range components.ServerBindings {
			if err = s.resolveServerBindingRef(component); err != nil {
				return
			}
		}
		for _, component := range components.ChannelBindings {
			if err = s.resolveChannelBindingRef(component); err != nil {
				return
			}
		}
		for _, component := range components.OperationBindings {
			if err = s.resolveOperationBindingRef(component); err != nil {
				return
			}
		}
		for _, component := range components.MessageBindings {
			if err = s.resolveMessageBindingRef(component); err != nil {
				return
			}
		}
	}

	for _, server := range s.Servers {
		if err = s.resolveServerRef(server); err != nil {
			return
		}
	}
	for _, channel := range s.Channels {
		if err = s.resolveChannelRef(channel); err != nil {
			return
		}
	}
	for _, operation := range s.Operations {
		if err = s.resolveOperationRef(operation); err != nil {
			return
		}
	}
	return
}

func (s *Spec) resolveServerRef(server *Server) error {
	return nil
}
func (s *Spec) resolveOperationRef(operation *Operation) error {
	return nil
}

func (s *Spec) resolveMultiFormatSchemaRef(ref *MultiFormatSchemaRef) error {
	return nil
}
func (s *Spec) resolveChannelRef(ref *ChannelRef) error {
	return nil
}
func (s *Spec) resolveMessageRef(ref *MessageRef) error {
	return nil
}
func (s *Spec) resolveSecuritySchemeRef(ref *SecuritySchemeRef) error {
	return nil
}
func (s *Spec) resolveServerVariableRef(ref *ServerVariableRef) error {
	return nil
}
func (s *Spec) resolveParameterRef(ref *ParameterRef) error {
	return nil
}
func (s *Spec) resolveCorrelationIdRef(ref *CorrelationIdRef) error {
	return nil
}
func (s *Spec) resolveOperationReplyRef(ref *OperationReplyRef) error {
	return nil
}
func (s *Spec) resolveOperationReplyAddressRef(ref *OperationReplyAddressRef) error {
	return nil
}
func (s *Spec) resolveExtensionDocsRef(ref *ExtensionDocsRef) error {
	return nil
}
func (s *Spec) resolveTagRef(ref *TagRef) error {
	return nil
}
func (s *Spec) resolveOperationTraitRef(ref *OperationTraitRef) error {
	return nil
}
func (s *Spec) resolveMessageTraitRef(ref *MessageTraitRef) error {
	return nil
}
func (s *Spec) resolveServerBindingRef(ref *ServerBindingRef) error {
	return nil
}
func (s *Spec) resolveChannelBindingRef(ref *ChannelBindingRef) error {
	return nil
}
func (s *Spec) resolveOperationBindingRef(ref *OperationBindingRef) error {
	return nil
}
func (s *Spec) resolveMessageBindingRef(ref *MessageBindingRef) error {
	return nil
}
