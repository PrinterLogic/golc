package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/chain"
	"github.com/hupe1980/golc/prompt"
)

// Compile time check to ensure ZeroShotReactDescriptionAgent satisfies the agent interface.
var _ golc.Agent = (*ZeroShotReactDescriptionAgent)(nil)

const (
	defaultMRKLPrefix = `Answer the following questions as best you can. You have access to the following tools:
	{{.toolDescriptions}}`

	defaultMRKLInstructions = `Use the following format:

	Question: the input question you must answer
	Thought: you should always think about what to do
	Action: the action to take, should be one of [{{.toolNames}}]
	Action Input: the input to the action
	Observation: the result of the action
	... (this Thought/Action/Action Input/Observation can repeat N times)
	Thought: I now know the final answer
	Final Answer: the final answer to the original input question`

	defaultMRKLSuffix = `Begin!

	Question: {{.input}}
	Thought: {{.agentScratchpad}}`

	finalAnswerAction = "Final Answer:"
)

type ZeroShotReactDescriptionAgentOptions struct {
	Prefix       string
	Instructions string
	Suffix       string
	OutputKey    string
}

type ZeroShotReactDescriptionAgent struct {
	chain golc.Chain
	tools []golc.Tool
	opts  ZeroShotReactDescriptionAgentOptions
}

func NewZeroShotReactDescriptionAgent(llm golc.LLM, tools []golc.Tool) (*ZeroShotReactDescriptionAgent, error) {
	opts := ZeroShotReactDescriptionAgentOptions{
		Prefix:       defaultMRKLPrefix,
		Instructions: defaultMRKLInstructions,
		Suffix:       defaultMRKLSuffix,
		OutputKey:    "output",
	}

	prompt, err := createMRKLPrompt(tools, opts.Prefix, opts.Instructions, opts.Suffix)
	if err != nil {
		return nil, err
	}

	llmChain, err := chain.NewLLMChain(llm, prompt)
	if err != nil {
		return nil, err
	}

	return &ZeroShotReactDescriptionAgent{
		chain: llmChain,
		tools: tools,
		opts:  opts,
	}, nil
}

func (a *ZeroShotReactDescriptionAgent) Plan(ctx context.Context, intermediateSteps []golc.AgentStep, inputs map[string]string) ([]golc.AgentAction, *golc.AgentFinish, error) {
	fullInputes := make(golc.ChainValues, len(inputs))
	for key, value := range inputs {
		fullInputes[key] = value
	}

	fullInputes["agentScratchpad"] = a.constructScratchPad(intermediateSteps)

	resp, err := chain.Call(ctx, a.chain, fullInputes)
	if err != nil {
		return nil, nil, err
	}

	output, ok := resp[a.chain.OutputKeys()[0]].(string)
	if !ok {
		return nil, nil, ErrInvalidChainReturnType
	}

	return a.parseOutput(output)
}

func (a *ZeroShotReactDescriptionAgent) InputKeys() []string {
	chainInputs := a.chain.InputKeys()

	agentInput := make([]string, 0, len(chainInputs))

	for _, v := range chainInputs {
		if v == "agentScratchpad" {
			continue
		}

		agentInput = append(agentInput, v)
	}

	return agentInput
}

func (a *ZeroShotReactDescriptionAgent) OutputKeys() []string {
	return []string{a.opts.OutputKey}
}

// constructScratchPad constructs the scratchpad that lets the agent
// continue its thought process.
func (a *ZeroShotReactDescriptionAgent) constructScratchPad(steps []golc.AgentStep) string {
	scratchPad := ""
	for _, step := range steps {
		scratchPad += step.Action.Log
		scratchPad += fmt.Sprintf("\nObservation: %s\nThought:", step.Observation)
	}

	return scratchPad
}

func (a *ZeroShotReactDescriptionAgent) parseOutput(output string) ([]golc.AgentAction, *golc.AgentFinish, error) {
	if strings.Contains(output, finalAnswerAction) {
		splits := strings.Split(output, finalAnswerAction)

		return nil, &golc.AgentFinish{
			ReturnValues: map[string]any{
				a.opts.OutputKey: splits[len(splits)-1],
			},
			Log: output,
		}, nil
	}

	r := regexp.MustCompile(`Action:\s*(.+)\s*Action Input:\s*(.+)`)
	matches := r.FindStringSubmatch(output)

	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("%w: %s", ErrUnableToParseOutput, output)
	}

	return []golc.AgentAction{
		{Tool: strings.TrimSpace(matches[1]), ToolInput: strings.TrimSpace(matches[2]), Log: output},
	}, nil, nil
}

func createMRKLPrompt(tools []golc.Tool, prefix, instructions, suffix string) (*prompt.Template, error) {
	return prompt.NewTemplate(strings.Join([]string{prefix, instructions, suffix}, "\n\n"), func(o *prompt.TemplateOptions) {
		o.PartialValues = prompt.PartialValues{
			"toolNames":        toolNames(tools),
			"toolDescriptions": toolDescriptions(tools),
		}
	})
}