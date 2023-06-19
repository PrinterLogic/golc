package chatmodel

import (
	"context"
	"fmt"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/tokenizer"
	"github.com/sashabaranov/go-openai"
)

// Compile time check to ensure OpenAI satisfies the ChatModel interface.
var _ schema.ChatModel = (*OpenAI)(nil)

type OpenAIOptions struct {
	*schema.CallbackOptions
	// Model name to use.
	ModelName string
	// Sampling temperature to use.
	Temperatur float32
	// The maximum number of tokens to generate in the completion.
	// -1 returns as many tokens as possible given the prompt and
	//the models maximal context size.
	MaxTokens int
	// Total probability mass of tokens to consider at each step.
	TopP float32
	// Penalizes repeated tokens.
	PresencePenalty float32
	// Penalizes repeated tokens according to frequency.
	FrequencyPenalty float32
	// How many completions to generate for each prompt.
	N int
	// Batch size to use when passing multiple documents to generate.
	BatchSize int
}

type OpenAI struct {
	schema.Tokenizer
	client *openai.Client
	opts   OpenAIOptions
}

func NewOpenAI(apiKey string) (*OpenAI, error) {
	opts := OpenAIOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelName:        "gpt-3.5-turbo",
		Temperatur:       1,
		TopP:             1,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
	}

	return &OpenAI{
		Tokenizer: tokenizer.NewOpenAI(opts.ModelName),
		client:    openai.NewClient(apiKey),
		opts:      opts,
	}, nil
}

func (cm *OpenAI) Generate(ctx context.Context, messages schema.ChatMessages) (*schema.LLMResult, error) {
	openAIMessages := []openai.ChatCompletionMessage{}

	for _, message := range messages {
		role, err := messageTypeToOpenAIRole(message.Type())
		if err != nil {
			return nil, err
		}

		openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: message.Text(),
		})
	}

	res, err := cm.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    cm.opts.ModelName,
		Messages: openAIMessages,
	})
	if err != nil {
		return nil, err
	}

	text := res.Choices[0].Message.Content
	role := res.Choices[0].Message.Role

	return &schema.LLMResult{
		Generations: [][]*schema.Generation{{&schema.Generation{
			Text:    text,
			Message: openAIResponseToChatMessage(role, text),
		}}},
		LLMOutput: map[string]any{},
	}, nil
}

func messageTypeToOpenAIRole(mType schema.ChatMessageType) (string, error) {
	switch mType { // nolint exhaustive
	case schema.ChatMessageTypeSystem:
		return "system", nil
	case schema.ChatMessageTypeAI:
		return "assistant", nil
	case schema.ChatMessageTypeHuman:
		return "user", nil
	default:
		return "", fmt.Errorf("unknown message type: %s", mType)
	}
}

func openAIResponseToChatMessage(role, text string) schema.ChatMessage {
	switch role {
	case "user":
		return schema.NewHumanChatMessage(text)
	case "assistant":
		return schema.NewAIChatMessage(text)
	case "system":
		return schema.NewSystemChatMessage(text)
	}

	return schema.NewGenericChatMessage(text, "unknown")
}

func (cm *OpenAI) Type() string {
	return "OpenAI"
}

func (cm *OpenAI) Verbose() bool {
	return cm.opts.CallbackOptions.Verbose
}

func (cm *OpenAI) Callbacks() []schema.Callback {
	return cm.opts.CallbackOptions.Callbacks
}
