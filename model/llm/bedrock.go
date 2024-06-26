package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockruntimeTypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/callback"
	"github.com/hupe1980/golc/integration/ai21"
	"github.com/hupe1980/golc/internal/util"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/tokenizer"
)

// Compile time check to ensure Bedrock satisfies the LLM interface.
var _ schema.LLM = (*Bedrock)(nil)

// providerStopSequenceKeyMap is a mapping between language model (LLM) providers
// and the corresponding key names used for stop sequences. Stop sequences are sets
// of words that, when encountered in the generated text, signal the language model
// to stop generating further content. Different LLM providers might use different
// key names to specify these stop sequences in the input parameters.
var providerStopSequenceKeyMap = map[string]string{
	"anthropic": "stop_sequences",
	"amazon":    "stopSequences",
	"ai21":      "stop_sequences",
	"cohere":    "stop_sequences",
	"mistral":   "stop",
}

// BedrockInputOutputAdapter is a helper struct for preparing input and handling output for Bedrock model.
type BedrockInputOutputAdapter struct {
	provider string
}

// NewBedrockInputOutputAdpter creates a new instance of BedrockInputOutputAdpter.
func NewBedrockInputOutputAdapter(provider string) *BedrockInputOutputAdapter {
	return &BedrockInputOutputAdapter{
		provider: provider,
	}
}

// PrepareInput prepares the input for the Bedrock model based on the specified provider.
func (bioa *BedrockInputOutputAdapter) PrepareInput(prompt string, modelParams map[string]any) ([]byte, error) {
	var body map[string]any

	switch bioa.provider {
	case "ai21":
		body = modelParams
		body["prompt"] = prompt
	case "amazon":
		body = make(map[string]any)
		body["inputText"] = prompt
		body["textGenerationConfig"] = modelParams
	case "anthropic":
		body = modelParams
		messages := []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: prompt,
			},
		}
		body["messages"] = messages
	case "cohere":
		body = modelParams
		body["prompt"] = prompt
	case "cohere-r":
		body = modelParams
		body["message"] = prompt
	case "meta":
		body = modelParams
		body["prompt"] = prompt
	case "mistral":
		body = modelParams
		body["prompt"] = fmt.Sprintf("<s>[INST] %s [/INST]", prompt)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", bioa.provider)
	}

	return json.Marshal(body)
}

// ai21Output represents the structure of the output from the AI21 language model.
// It is used for unmarshaling JSON responses from the language model's API.
type ai21Output struct {
	Completions []struct {
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	} `json:"completions"`
}

// amazonOutput represents the structure of the output from the Amazon language model.
// It is used for unmarshaling JSON responses from the language model's API.
type amazonOutput struct {
	InputTextTokenCount int `json:"inputTextTokenCount"`
	Results             []struct {
		OutputText       string `json:"outputText"`
		TokenCount       int    `json:"tokenCount"`
		CompletionReason string `json:"completionReason"`
	} `json:"results"`
}

// anthropicOutput is a struct representing the output structure for the "anthropic" provider.
type anthropicOutput struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// cohereOutput is a struct representing the output structure for the "cohere" provider.
type cohereOutput struct {
	Generations []struct {
		Text string `json:"text"`
	} `json:"generations"`
}

// cohereCommandROutput is a struct representing the output structure for the "cohere" provider command r model family.
type cohereCommandROutput struct {
	ResponseID   string `json:"response_id"`
	Text         string `json:"text"`
	GenerationID string `json:"generation_id"`
	FinishReason string `json:"finish_reason"`
}

// metaOutput is a struct representing the output structure for the "meta" provider.
type metaOutput struct {
	Generation string `json:"generation"`
}

// mistralOutput is a struct representing the output structure for the "mistral" provider.
type mistralOutput struct {
	Outputs []struct {
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"outputs"`
}

// PrepareOutput prepares the output for the Bedrock model based on the specified provider.
func (bioa *BedrockInputOutputAdapter) PrepareOutput(response []byte) (string, error) {
	switch bioa.provider {
	case "ai21":
		output := &ai21Output{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Completions[0].Data.Text, nil
	case "amazon":
		output := &amazonOutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Results[0].OutputText, nil
	case "anthropic":
		output := &anthropicOutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Content[0].Text, nil
	case "cohere":
		output := &cohereOutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Generations[0].Text, nil
	case "cohere-r":
		output := &cohereCommandROutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Text, nil
	case "meta":
		output := &metaOutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Generation, nil
	case "mistral":
		output := &mistralOutput{}
		if err := json.Unmarshal(response, output); err != nil {
			return "", err
		}

		return output.Outputs[0].Text, nil
	}

	return "", fmt.Errorf("unsupported provider: %s", bioa.provider)
}

// BedrockInvocationMetrics represents the structure of the invocation metrics for the model invoked by Bedrock.
type BedrockInvocationMetrics struct {
	InputTokenCount  int32 `json:"inputTokenCount"`
	OutputTokenCount int32 `json:"outputTokenCount"`
}

// amazonStreamOutput represents the structure of the stream output from the Amazon language model.
// It is used for unmarshaling JSON responses from the language model's API.
type amazonStreamOutput struct {
	Index                     int                      `json:"index"`
	InputTextTokenCount       int                      `json:"inputTextTokenCount"`
	TotalOutputTextTokenCount int                      `json:"totalOutputTextTokenCount"`
	OutputText                string                   `json:"outputText"`
	CompletionReason          string                   `json:"completionReason"`
	Metrics                   BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

// anthropicStreamOutput is a struct representing the stream output structure for the "anthropic" provider.
type anthropicStreamOutput struct {
	Type  string `json:"type"`
	Delta struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Metrics BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

// cohereStreamOutput is a struct representing the stream output structure for the "cohere" provider.
type cohereStreamOutput struct {
	Generations []struct {
		Text string `json:"text"`
	} `json:"generations"`
	Metrics BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

// cohereStreamCommandROutput is a struct representing the stream output structure for the "cohere" provider command r model family.
type cohereStreamCommandROutput struct {
	Text         string                   `json:"text"`
	IsFinished   bool                     `json:"is_finished"`
	FinishReason string                   `json:"finish_reason"`
	Metrics      BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

// metaStreamOutput is a struct representing the stream output structure for the "meta" provider.
type metaStreamOutput struct {
	Generation string                   `json:"generation"`
	Metrics    BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

// mistralStreamOutput is a struct representing the stream output structure for the "mistral" provider.
type mistralStreamOutput struct {
	Outputs []struct {
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"outputs"`
	Metrics BedrockInvocationMetrics `json:"amazon-bedrock-invocationMetrics"`
}

type streamOutput struct {
	token        string
	inputTokens  int32
	outputTokens int32
}

// PrepareStreamOutput prepares the output for the Bedrock model based on the specified provider.
func (bioa *BedrockInputOutputAdapter) PrepareStreamOutput(response []byte) (streamOutput, error) {
	output := streamOutput{}

	switch bioa.provider {
	case "amazon":
		o := &amazonStreamOutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.OutputText
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	case "anthropic":
		o := &anthropicStreamOutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.Delta.Text
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	case "cohere":
		o := &cohereStreamOutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.Generations[0].Text
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	case "cohere-r":
		o := &cohereStreamCommandROutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.Text
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	case "meta":
		o := &metaStreamOutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.Generation
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	case "mistral":
		o := &mistralStreamOutput{}
		if err := json.Unmarshal(response, o); err != nil {
			return output, err
		}

		output.token = o.Outputs[0].Text
		output.inputTokens = o.Metrics.InputTokenCount
		output.outputTokens = o.Metrics.OutputTokenCount
	default:
		return output, fmt.Errorf("unsupported provider: %s", bioa.provider)
	}

	return output, nil
}

// BedrockRuntimeClient is an interface for the Bedrock model runtime client.
type BedrockRuntimeClient interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

type BedrockAI21Options struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float64 `map:"temperature"`

	// TopP sets the nucleus sampling probability. Higher values result in more diverse text.
	TopP float64 `map:"topP"`

	// MaxTokens sets the maximum number of tokens in the generated text.
	MaxTokens int `map:"maxTokens"`

	// PresencePenalty specifies the penalty for repeating words in generated text.
	PresencePenalty ai21.Penalty `map:"presencePenalty"`

	// CountPenalty specifies the penalty for repeating tokens in generated text.
	CountPenalty ai21.Penalty `map:"countPenalty"`

	// FrequencyPenalty specifies the penalty for generating frequent words.
	FrequencyPenalty ai21.Penalty `map:"frequencyPenalty"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

func NewBedrockAI21(client BedrockRuntimeClient, optFns ...func(o *BedrockAI21Options)) (*Bedrock, error) {
	opts := BedrockAI21Options{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:          "ai21.j2-ultra-v1", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature:      0.5,
		TopP:             0.5,
		MaxTokens:        200,
		PresencePenalty:  DefaultPenalty,
		CountPenalty:     DefaultPenalty,
		FrequencyPenalty: DefaultPenalty,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"temperature":      opts.Temperature,
			"topP":             opts.TopP,
			"maxTokens":        opts.MaxTokens,
			"presencePenalty":  opts.PresencePenalty,
			"countPenalty":     opts.CountPenalty,
			"frequencyPenalty": opts.FrequencyPenalty,
		}
		o.Stream = opts.Stream
	})
}

type BedrockAnthropicOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// MaxTokensToSmaple sets the maximum number of tokens in the generated text.
	MaxTokensToSample int `map:"max_tokens_to_sample"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float32 `map:"temperature"`

	// TopP is the total probability mass of tokens to consider at each step.
	TopP float32 `map:"top_p,omitempty"`

	// TopK determines how the model selects tokens for output.
	TopK int `map:"top_k"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

func NewBedrockAnthropic(client BedrockRuntimeClient, optFns ...func(o *BedrockAnthropicOptions)) (*Bedrock, error) {
	opts := BedrockAnthropicOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:           "anthropic.claude-v2", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature:       0.5,
		MaxTokensToSample: 256,
		TopP:              1,
		TopK:              250,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewClaude()
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"max_tokens":        opts.MaxTokensToSample,
			"temperature":       opts.Temperature,
			"top_p":             opts.TopP,
			"top_k":             opts.TopK,
			"anthropic_version": AnthropicVersion,
		}
		o.Stream = opts.Stream
	})
}

type BedrockAmazonOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float64 `json:"temperature"`

	// TopP is the total probability mass of tokens to consider at each step.
	TopP float64 `json:"topP"`

	// MaxTokenCount sets the maximum number of tokens in the generated text.
	MaxTokenCount int `json:"maxTokenCount"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

func NewBedrockAmazon(client BedrockRuntimeClient, optFns ...func(o *BedrockAmazonOptions)) (*Bedrock, error) {
	opts := BedrockAmazonOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:       "amazon.titan-text-lite-v1", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature:   0,
		TopP:          1,
		MaxTokenCount: 512,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"temperature":   opts.Temperature,
			"topP":          opts.TopP,
			"maxTokenCount": opts.MaxTokenCount,
		}
		o.Stream = opts.Stream
	})
}

type ReturnLikelihood string

const (
	ReturnLikelihoodGeneration ReturnLikelihood = "GENERATION"
	ReturnLikelihoodAll        ReturnLikelihood = "ALL"
	ReturnLikelihoodNone       ReturnLikelihood = "NONE"
)

type BedrockCohereOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float64 `json:"temperature,omitempty"`

	// P is the total probability mass of tokens to consider at each step.
	P float64 `json:"p,omitempty"`

	// K determines how the model selects tokens for output.
	K float64 `json:"k,omitempty"`

	// MaxTokens sets the maximum number of tokens in the generated text.
	MaxTokens int `json:"max_tokens,omitempty"`

	// ReturnLikelihoods specifies how and if the token likelihoods are returned with the response.
	ReturnLikelihoods ReturnLikelihood `json:"return_likelihoods,omitempty"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

func NewBedrockCohere(client BedrockRuntimeClient, optFns ...func(o *BedrockCohereOptions)) (*Bedrock, error) {
	opts := BedrockCohereOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:           "cohere.command-text-v14", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature:       0.9,
		P:                 0.75,
		K:                 0,
		MaxTokens:         20,
		ReturnLikelihoods: ReturnLikelihoodNone,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewCohere(opts.ModelID)
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"temperature":        opts.Temperature,
			"p":                  opts.P,
			"k":                  opts.K,
			"max_tokens":         opts.MaxTokens,
			"return_likelihoods": opts.ReturnLikelihoods,
			"stream":             opts.Stream,
		}
		o.Stream = opts.Stream
	})
}

// BedrockMetaOptions contains options for configuring the Bedrock model with the "meta" provider.
type BedrockMetaOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float32 `map:"temperature"`

	// TopP is the total probability mass of tokens to consider at each step.
	TopP float32 `map:"top_p,omitempty"`

	// MaxGenLen specify the maximum number of tokens to use in the generated response.
	MaxGenLen int `map:"max_gen_len"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

// NewBedrockMeta creates a new instance of Bedrock for the "meta" provider.
func NewBedrockMeta(client BedrockRuntimeClient, optFns ...func(o *BedrockMetaOptions)) (*Bedrock, error) {
	opts := BedrockMetaOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:     "meta.llama2-70b-v1", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature: 0.5,
		TopP:        0.9,
		MaxGenLen:   512,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"max_gen_len": opts.MaxGenLen,
		}
		o.Stream = opts.Stream
	})
}

type BedrockMistralOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// Model id to use.
	ModelID string `map:"model_id,omitempty"`

	// Temperature controls the randomness of text generation. Higher values make it more random.
	Temperature float32 `map:"temperature"`

	// TopP is the total probability mass of tokens to consider at each step.
	TopP float32 `map:"top_p,omitempty"`

	// TopK determines how the model selects tokens for output.
	TopK int `map:"top_k"`

	// MaxTokens sets the maximum number of tokens in the generated text.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

func NewBedrockMistral(client BedrockRuntimeClient, optFns ...func(o *BedrockMistralOptions)) (*Bedrock, error) {
	opts := BedrockMistralOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelID:     "mistral.mistral-7b-instruct-v0:2", //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
		Temperature: 0.5,
		TopP:        0.9,
		TopK:        200,
		MaxTokens:   512,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	return NewBedrock(client, opts.ModelID, func(o *BedrockOptions) {
		o.CallbackOptions = opts.CallbackOptions
		o.Tokenizer = opts.Tokenizer
		o.ModelParams = map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"top_k":       opts.TopK,
			"max_tokens":  opts.MaxTokens,
		}
		o.Stream = opts.Stream
	})
}

func prepareAI21InferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["maxTokens"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["topP"] = opts.TopP

	return params
}

func prepareAmazonInferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["maxTokenCount"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["topP"] = opts.TopP

	return params
}

const AnthropicVersion string = "bedrock-2023-05-31"

func prepareAnthropicInferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["max_tokens"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["top_p"] = opts.TopP
	params["anthropic_version"] = AnthropicVersion

	return params
}

func prepareCohereInferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["max_tokens"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["p"] = opts.TopP

	return params
}

func prepareMetaInferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["max_gen_len"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["top_p"] = opts.TopP

	return params
}

func prepareMistralInferenceParams(opts *BedrockOptions) map[string]any {
	params := opts.ModelParams
	params["max_tokens"] = opts.MaxTokens
	params["temperature"] = opts.Temperature
	params["top_p"] = opts.TopP

	return params
}

func prepareModelInferenceParams(opts *BedrockOptions, modelID string) map[string]any {
	if opts == nil || (opts.MaxTokens == nil && len(opts.StopSequences) == 0 && opts.Temperature == nil && opts.TopP == nil) {
		return opts.ModelParams
	}

	provider := strings.Split(modelID, ".")[0]

	switch provider {
	case "ai21":
		return prepareAI21InferenceParams(opts)
	case "anthropic":
		return prepareAnthropicInferenceParams(opts)
	case "amazon":
		return prepareAmazonInferenceParams(opts)
	case "cohere":
		return prepareCohereInferenceParams(opts)
	case "meta":
		return prepareMetaInferenceParams(opts)
	case "mistral":
		return prepareMistralInferenceParams(opts)
	default:
		return opts.ModelParams
	}
}

// BedrockOptions contains options for configuring the Bedrock LLM model.
type BedrockOptions struct {
	*schema.CallbackOptions `map:"-"`
	schema.Tokenizer        `map:"-"`

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens *int32

	// Stop is a list of sequences to stop the generation at.
	StopSequences []string

	// Temperature
	Temperature *float32

	// TopP
	TopP *float32

	// Model params to use.
	ModelParams map[string]any `map:"model_params,omitempty"`

	// Stream indicates whether to stream the results or not.
	Stream bool `map:"stream,omitempty"`
}

// Bedrock is a Bedrock LLM model that generates text based on a provided response function.
type Bedrock struct {
	schema.Tokenizer
	client  BedrockRuntimeClient
	modelID string
	opts    BedrockOptions
}

// NewBedrock creates a new instance of the Bedrock LLM model with the provided response function and options.
func NewBedrock(client BedrockRuntimeClient, modelID string, optFns ...func(o *BedrockOptions)) (*Bedrock, error) {
	opts := BedrockOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelParams: make(map[string]any),
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	opts.ModelParams = prepareModelInferenceParams(&opts, modelID)

	return &Bedrock{
		Tokenizer: opts.Tokenizer,
		client:    client,
		modelID:   modelID,
		opts:      opts,
	}, nil
}

// Generate generates text based on the provided prompt and options.
func (l *Bedrock) Generate(ctx context.Context, prompt string, optFns ...func(o *schema.GenerateOptions)) (*schema.ModelResult, error) {
	opts := schema.GenerateOptions{
		CallbackManger: &callback.NoopManager{},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	provider := l.getProvider()

	params := util.CopyMap(l.opts.ModelParams)

	if len(opts.Stop) > 0 {
		key, ok := providerStopSequenceKeyMap[provider]
		if !ok {
			return nil, fmt.Errorf("stop sequence key name for provider %s is not supported", provider)
		}

		params[key] = opts.Stop
	}

	bioa := NewBedrockInputOutputAdapter(provider)

	body, err := bioa.PrepareInput(prompt, params)
	if err != nil {
		return nil, err
	}

	var completion string

	llmOutput := make(map[string]any)

	if l.opts.Stream {
		res, err := l.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
			ModelId:     aws.String(l.modelID),
			Body:        body,
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			return nil, err
		}

		stream := res.GetStream()

		defer stream.Close()

		tokens := []string{}
		llmOutput["input_tokens"] = int32(0)
		llmOutput["output_tokens"] = int32(0)

		for event := range stream.Events() {
			switch v := event.(type) {
			case *bedrockruntimeTypes.ResponseStreamMemberChunk:
				output, err := bioa.PrepareStreamOutput(v.Value.Bytes)
				if err != nil {
					return nil, err
				}

				if err := opts.CallbackManger.OnModelNewToken(ctx, &schema.ModelNewTokenManagerInput{
					Token: output.token,
				}); err != nil {
					return nil, err
				}

				tokens = append(tokens, output.token)
				llmOutput["input_tokens"] = llmOutput["input_tokens"].(int32) + output.inputTokens
				llmOutput["output_tokens"] = llmOutput["output_tokens"].(int32) + output.outputTokens
			}
		}

		completion = strings.Join(tokens, "")
	} else {
		res, err := l.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(l.modelID),
			Body:        body,
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			return nil, err
		}

		output, err := bioa.PrepareOutput(res.Body)
		if err != nil {
			return nil, err
		}

		completion = output
	}

	return &schema.ModelResult{
		Generations: []schema.Generation{{Text: completion}},
		LLMOutput:   llmOutput,
	}, nil
}

// Type returns the type of the model.
func (l *Bedrock) Type() string {
	return "llm.Bedrock"
}

// Verbose returns the verbosity setting of the model.
func (l *Bedrock) Verbose() bool {
	return l.opts.Verbose
}

// Callbacks returns the registered callbacks of the model.
func (l *Bedrock) Callbacks() []schema.Callback {
	return l.opts.Callbacks
}

// InvocationParams returns the parameters used in the model invocation.
func (l *Bedrock) InvocationParams() map[string]any {
	params := util.StructToMap(l.opts)
	params["model_id"] = l.modelID

	return params
}

// getProvider returns the provider of the model based on the model ID.
func (l *Bedrock) getProvider() string {
	provider := strings.Split(l.modelID, ".")[0]

	if provider == "cohere" && strings.Contains(l.modelID, "command-r") {
		provider = provider + "-r"
	}

	return provider
}
