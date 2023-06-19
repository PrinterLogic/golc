package llm

import (
	"context"

	"github.com/cohere-ai/cohere-go"
	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/tokenizer"
)

// Compile time check to ensure Cohere satisfies the LLM interface.
var _ schema.LLM = (*Cohere)(nil)

type CohereOptions struct {
	*schema.CallbackOptions
	Model      string
	Temperatur float32
}

type Cohere struct {
	schema.Tokenizer
	client *cohere.Client
	opts   CohereOptions
}

func NewCohere(apiKey string, optFns ...func(o *CohereOptions)) (*Cohere, error) {
	opts := CohereOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		Model: "medium",
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	client, err := cohere.CreateClient(apiKey)
	if err != nil {
		return nil, err
	}

	return &Cohere{
		Tokenizer: tokenizer.NewSimple(),
		client:    client,
		opts:      opts,
	}, nil
}

func (l *Cohere) Generate(ctx context.Context, prompts []string, stop []string) (*schema.LLMResult, error) {
	res, err := l.client.Generate(cohere.GenerateOptions{
		Model:         l.opts.Model,
		Prompt:        prompts[0],
		StopSequences: stop,
	})
	if err != nil {
		return nil, err
	}

	return &schema.LLMResult{
		Generations: [][]*schema.Generation{{&schema.Generation{Text: res.Generations[0].Text}}},
		LLMOutput:   map[string]any{},
	}, nil
}

func (l *Cohere) Type() string {
	return "Cohere"
}

func (l *Cohere) Verbose() bool {
	return l.opts.CallbackOptions.Verbose
}

func (l *Cohere) Callbacks() []schema.Callback {
	return l.opts.CallbackOptions.Callbacks
}