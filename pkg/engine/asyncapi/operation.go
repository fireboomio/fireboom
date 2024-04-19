package asyncapi

type (
	Operations map[string]*Operation
	Operation  struct {
		// Required. Use send when it's expected that the application will send a message to the given channel, and receive when the application should expect receiving messages from the given channel.
		Action OperationAction `json:"action" yaml:"action"`
		// Required. A $ref pointer to the definition of the channel in which this operation is performed. If the operation is located in the root Operations Object, it MUST point to a channel definition located in the root Channels Object, and MUST NOT point to a channel definition located in the Components Object or anywhere else. If the operation is located in the Components Object, it MAY point to a Channel Object in any location. Please note the channel property value MUST be a Reference Object and, therefore, MUST NOT contain a Channel Object. However, it is RECOMMENDED that parsers (or other software) dereference this property for a better development experience.
		Channel *ChannelRef `json:"channel" yaml:"channel"`
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
		// A list of traits to apply to the operation object. Traits MUST be merged using traits merge mechanism. The resulting object MUST be a valid Operation Object.
		Traits OperationTraits `json:"traits" yaml:"traits"`
		// A list of $ref pointers pointing to the supported Message Objects that can be processed by this operation. It MUST contain a subset of the messages defined in the channel referenced in this operation, and MUST NOT point to a subset of message definitions located in the Messages Object in the Components Object or anywhere else. Every message processed by this operation MUST be valid against one, and only one, of the message objects referenced in this list. Please note the messages property value MUST be a list of Reference Objects and, therefore, MUST NOT contain Message Objects. However, it is RECOMMENDED that parsers (or other software) dereference this property for a better development experience.
		Messages Messages `json:"messages" yaml:"messages"`
		// The definition of the reply in a request-reply operation.
		Reply *OperationReplyRef `json:"reply" yaml:"reply"`
	}
)

type OperationAction string

const (
	OperationAction_receive   OperationAction = "receive"
	OperationAction_send      OperationAction = "send"
	OperationAction_Publish   OperationAction = "Publish"
	OperationAction_Subscribe OperationAction = "Subscribe"
)
