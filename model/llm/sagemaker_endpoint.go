package llm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"
	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/tokenizer"
)

// Compile time check to ensure SagemakerEndpoint satisfies the LLM interface.
var _ schema.LLM = (*SagemakerEndpoint)(nil)

type Transformer interface {
	// Transforms the input to a format that model can accept
	// as the request Body. Should return bytes or seekable file
	// like object in the format specified in the content_type
	// request header.
	TransformInput(prompt string) ([]byte, error)

	// Transforms the output from the model to string that
	// the LLM class expects.
	TransformOutput(output []byte) (string, error)
}

type LLMContentHandler struct {
	// The MIME type of the input data passed to endpoint.
	contentType string

	// The MIME type of the response data returned from endpoint
	accept string

	transformer Transformer
}

func NewLLMContentHandler(contentType, accept string, transformer Transformer) *LLMContentHandler {
	return &LLMContentHandler{
		contentType: contentType,
		accept:      accept,
		transformer: transformer,
	}
}

func (ch *LLMContentHandler) ContentType() string {
	return ch.contentType
}

func (ch *LLMContentHandler) Accept() string {
	return ch.accept
}

func (ch *LLMContentHandler) TransformInput(prompt string) ([]byte, error) {
	return ch.transformer.TransformInput(prompt)
}

func (ch *LLMContentHandler) TransformOutput(output []byte) (string, error) {
	return ch.transformer.TransformOutput(output)
}

type SagemakerEndpointOptions struct {
	*schema.CallbackOptions
}

type SagemakerEndpoint struct {
	schema.Tokenizer
	client        *sagemakerruntime.Client
	endpointName  string
	contenHandler *LLMContentHandler
	opts          SagemakerEndpointOptions
}

func NewSagemakerEndpoint(client *sagemakerruntime.Client, endpointName string, contenHandler *LLMContentHandler) (*SagemakerEndpoint, error) {
	opts := SagemakerEndpointOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
	}

	return &SagemakerEndpoint{
		Tokenizer:     tokenizer.NewSimple(),
		client:        client,
		endpointName:  endpointName,
		contenHandler: contenHandler,
		opts:          opts,
	}, nil
}

func (l *SagemakerEndpoint) Generate(ctx context.Context, prompts []string, stop []string) (*schema.LLMResult, error) {
	generations := [][]*schema.Generation{}

	for _, prompt := range prompts {
		body, err := l.contenHandler.TransformInput(prompt)
		if err != nil {
			return nil, err
		}

		out, err := l.client.InvokeEndpoint(ctx, &sagemakerruntime.InvokeEndpointInput{
			EndpointName: aws.String(l.endpointName),
			ContentType:  aws.String(l.contenHandler.ContentType()),
			Accept:       aws.String(l.contenHandler.Accept()),
			Body:         body,
		})
		if err != nil {
			return nil, err
		}

		text, err := l.contenHandler.TransformOutput(out.Body)
		if err != nil {
			return nil, err
		}

		generations = append(generations, []*schema.Generation{{
			Text: text,
		}})
	}

	return &schema.LLMResult{
		Generations: generations,
		LLMOutput:   map[string]any{},
	}, nil
}

func (l *SagemakerEndpoint) Type() string {
	return "SagemakerEndpoint"
}

func (l *SagemakerEndpoint) Verbose() bool {
	return l.opts.CallbackOptions.Verbose
}

func (l *SagemakerEndpoint) Callbacks() []schema.Callback {
	return l.opts.CallbackOptions.Callbacks
}