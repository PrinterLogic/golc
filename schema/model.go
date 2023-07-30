package schema

import (
	"context"

	"github.com/hupe1980/golc/integration/jsonschema"
)

// Generation represents a generated text along with its corresponding chat message and additional information.
type Generation struct {
	Text    string
	Message ChatMessage
	Info    map[string]any
}

// ModelResult represents the result of a model generation.
type ModelResult struct {
	Generations []Generation
	LLMOutput   map[string]any
}

// PromptValue is an interface representing a prompt value for LLMs and chat models.
type PromptValue interface {
	String() string
	Messages() ChatMessages
}

// Tokenizer is an interface for tokenizing text.
type Tokenizer interface {
	// GetTokenIDs returns the token IDs corresponding to the provided text.
	GetTokenIDs(text string) ([]uint, error)
	// GetNumTokens returns the number of tokens in the provided text.
	GetNumTokens(text string) (uint, error)
	// GetNumTokensFromMessage returns the number of tokens in the provided chat messages.
	GetNumTokensFromMessage(messages ChatMessages) (uint, error)
}

type FunctionDefinitionParameters struct {
	Type       string                        `json:"type"`
	Properties map[string]*jsonschema.Schema `json:"properties"`
	Required   []string                      `json:"required"`
}

type FunctionDefinition struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Parameters  FunctionDefinitionParameters `json:"parameters"`
}

type GenerateOptions struct {
	CallbackManger CallbackManagerForModelRun
	Stop           []string
	Functions      []FunctionDefinition
}

// LLM is the interface for language models.
type LLM interface {
	Model
	// Generate generates text based on the provided prompt and options.
	Generate(ctx context.Context, prompt string, optFns ...func(o *GenerateOptions)) (*ModelResult, error)
}

// ChatModel is the interface for chat models.
type ChatModel interface {
	Model
	// Generate generates text based on the provided chat messages and options.
	Generate(ctx context.Context, messages ChatMessages, optFns ...func(o *GenerateOptions)) (*ModelResult, error)
}

// Model is the interface for language models and chat models.
type Model interface {
	Tokenizer
	// Type returns the type of the model.
	Type() string
	// Verbose returns the verbosity setting of the model.
	Verbose() bool
	// Callbacks returns the registered callbacks of the model.
	Callbacks() []Callback
	// InvocationParams returns the parameters used in the model invocation.
	InvocationParams() map[string]any
}

// Embedder is the interface for creating vector embeddings from texts.
type Embedder interface {
	// EmbedDocuments embeds a list of documents and returns their embeddings.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error)
	// EmbedQuery embeds a single query and returns its embedding.
	EmbedQuery(ctx context.Context, text string) ([]float64, error)
}

// OutputParser is an interface for parsing the output of an LLM call.
type OutputParser[T any] interface {
	// Parse parses the output of an LLM call.
	ParseResult(result Generation) (any, error)
	// Parse parses the output of an LLM call.
	Parse(text string) (T, error)
	// ParseWithPrompt parses the output of an LLM call with the prompt used.
	ParseWithPrompt(text string, prompt PromptValue) (T, error)
	// GetFormatInstructions returns a string describing the format of the output.
	GetFormatInstructions() string
	// Type returns the string type key uniquely identifying this class of parser
	Type() string
}
