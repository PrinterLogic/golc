package chain

import (
	"context"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/model"
	"github.com/hupe1980/golc/prompt"
	"github.com/hupe1980/golc/schema"
)

type LLMChainOptions struct {
	*schema.CallbackOptions
	Memory       schema.Memory
	OutputKey    string
	OutputParser schema.OutputParser[any]
}

type LLMChain struct {
	llm    schema.LLM
	prompt *prompt.Template
	opts   LLMChainOptions
}

func NewLLMChain(llm schema.LLM, prompt *prompt.Template, optFns ...func(o *LLMChainOptions)) (*LLMChain, error) {
	opts := LLMChainOptions{
		OutputKey: "text",
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	return &LLMChain{
		prompt: prompt,
		llm:    llm,
		opts:   opts,
	}, nil
}

func (c *LLMChain) Call(ctx context.Context, inputs schema.ChainValues) (schema.ChainValues, error) {
	promptValue, err := c.prompt.FormatPrompt(inputs)
	if err != nil {
		return nil, err
	}

	res, err := model.GeneratePrompt(ctx, c.llm, []schema.PromptValue{promptValue}, func(o *schema.GenerateOptions) {
		o.Callbacks = c.opts.Callbacks
	})
	if err != nil {
		return nil, err
	}

	return schema.ChainValues{
		c.opts.OutputKey: c.getFinalOutput(res.Generations),
	}, nil
}

func (c *LLMChain) Prompt() *prompt.Template {
	return c.prompt
}

func (c *LLMChain) Memory() schema.Memory {
	return c.opts.Memory
}

func (c *LLMChain) Type() string {
	return "LLM"
}

func (c *LLMChain) Verbose() bool {
	return c.opts.CallbackOptions.Verbose
}

func (c *LLMChain) Callbacks() []schema.Callback {
	return c.opts.CallbackOptions.Callbacks
}

// InputKeys returns the expected input keys.
func (c *LLMChain) InputKeys() []string {
	return c.prompt.InputVariables()
}

// OutputKeys returns the output keys the chain will return.
func (c *LLMChain) OutputKeys() []string {
	return []string{c.opts.OutputKey}
}

func (c *LLMChain) getFinalOutput(generations [][]*schema.Generation) string {
	output := []string{}
	for _, generation := range generations {
		// Get the text of the top generated string.
		output = append(output, strings.TrimSpace(generation[0].Text))
	}

	return output[0]
}