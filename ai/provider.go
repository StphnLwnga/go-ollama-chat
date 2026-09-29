package ai

// Provider is what a chat backend must offer. handler never see this
// they keep calling the package functions below
type Provider interface {
	Chat(model string, messages []Message) (string, error)
	ChatStream(model string, messages []Message, onChunk func(string) error) error
}

var current Provider = Ollama{} // default until Init says otherwise

// Init picks the AI provider=ollama|groq
func Init(provider, groqKey string) {
	if provider == "groq" {
		current = Groq{APIKey: groqKey}
	}
}

// Chat uses whatever provider was selected in Init
func Chat(model string, messages []Message) (string, error) {
	return current.Chat(model, messages)
}

// ChatStream uses whatever provider was selected in Init
func ChatStream(model string, messages []Message, onChunk func(string) error) error {
	return current.ChatStream(model, messages, onChunk)
}
