package golc

import (
	"context"
	"fmt"
	"strings"
)

type Generation struct {
	Text    string
	Message ChatMessage
	Info    map[string]any
}

type LLMResult struct {
	Generations [][]Generation
	LLMOutput   map[string]any
}

type ChainValues map[string]any

type Memory interface {
	// Input keys this memory class will load dynamically.
	MemoryVariables() []string
	// Return key-value pairs given the text input to the chain.
	// If None, return all memories
	LoadMemoryVariables(inputs map[string]any) (map[string]any, error)
	// Save the context of this model run to memory.
	SaveContext(inputs map[string]any, outputs map[string]any) error
	// Clear memory contents.
	Clear() error
}

type PromptValue interface {
	String() string
	Messages() []ChatMessage
}

type LLM interface {
	GeneratePrompt(ctx context.Context, promptValues []PromptValue) (*LLMResult, error)
	Predict(ctx context.Context, text string) (string, error)
	PredictMessages(ctx context.Context, messages []ChatMessage) (ChatMessage, error)
}

type Document struct {
	PageContent string
	Metadata    map[string]any
}

type Retriever interface {
	GetRelevantDocuments(ctx context.Context, query string) ([]Document, error)
}

type ChatMessageType string

const (
	ChatMessageTypeHuman   ChatMessageType = "human"
	ChatMessageTypeAI      ChatMessageType = "ai"
	ChatMessageTypeSystem  ChatMessageType = "system"
	ChatMessageTypeGeneric ChatMessageType = "generic"
)

type ChatMessage interface {
	Text() string
	Type() ChatMessageType
}

type HumanChatMessage struct {
	text string
}

func NewHumanChatMessage(text string) *HumanChatMessage {
	return &HumanChatMessage{
		text: text,
	}
}

func (m HumanChatMessage) Type() ChatMessageType { return ChatMessageTypeHuman }
func (m HumanChatMessage) Text() string          { return m.text }

type AIChatMessage struct {
	text string
}

func NewAIChatMessage(text string) *AIChatMessage {
	return &AIChatMessage{
		text: text,
	}
}

func (m AIChatMessage) Type() ChatMessageType { return ChatMessageTypeAI }
func (m AIChatMessage) Text() string          { return m.text }

type SystemChatMessage struct {
	text string
}

func NewSystemChatMessage(text string) *SystemChatMessage {
	return &SystemChatMessage{
		text: text,
	}
}

func (m SystemChatMessage) Type() ChatMessageType { return ChatMessageTypeSystem }
func (m SystemChatMessage) Text() string          { return m.text }

type GenericChatMessage struct {
	text string
	role string
}

func NewGenericChatMessage(text, role string) *GenericChatMessage {
	return &GenericChatMessage{
		text: text,
		role: role,
	}
}

func (m GenericChatMessage) Type() ChatMessageType { return ChatMessageTypeGeneric }
func (m GenericChatMessage) Text() string          { return m.text }
func (m GenericChatMessage) Role() string          { return m.role }

type StringifyChatMessagesOptions struct {
	HumanPrefix  string
	AIPrefix     string
	SystemPrefix string
}

func StringifyChatMessages(messages []ChatMessage, optFns ...func(o *StringifyChatMessagesOptions)) (string, error) {
	opts := StringifyChatMessagesOptions{
		HumanPrefix:  "Human",
		AIPrefix:     "AI",
		SystemPrefix: "System",
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	result := []string{}

	for _, message := range messages {
		var role string

		switch message.Type() {
		case ChatMessageTypeHuman:
			role = opts.HumanPrefix
		case ChatMessageTypeAI:
			role = opts.AIPrefix
		case ChatMessageTypeSystem:
			role = opts.SystemPrefix
		case ChatMessageTypeGeneric:
			role = message.(GenericChatMessage).Role()
		default:
			return "", fmt.Errorf("unknown chat message type: %s", message.Type())
		}

		result = append(result, fmt.Sprintf("%s: %s", role, message.Text()))
	}

	return strings.Join(result, "\n"), nil
}