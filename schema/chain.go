package schema

import (
	"context"
	"fmt"

	"github.com/hupe1980/golc/util"
)

// ChainValues represents a map of key-value pairs used for passing inputs and outputs between chain components.
type ChainValues map[string]any

// GetString retrieves the value associated with the given name as a string from ChainValues.
// If the name does not exist or the value is not a string, an error is returned.
func (cv ChainValues) GetString(name string) (string, error) {
	input, ok := cv[name]
	if !ok {
		return "", fmt.Errorf("%w: no value for inputKey %s", ErrInvalidInputValues, name)
	}

	value, ok := input.(string)
	if !ok {
		return "", ErrInputValuesWrongType
	}

	return value, nil
}

// GetDocuments retrieves the value associated with the given name as a slice of documents from ChainValues.
// If the name does not exist, the value is not a slice of documents, or the slice is empty, an error is returned.
func (cv ChainValues) GetDocuments(name string) ([]Document, error) {
	input, ok := cv[name]
	if !ok {
		return nil, fmt.Errorf("%w: no value for inputKey %s", ErrInvalidInputValues, name)
	}

	docs, ok := input.([]Document)
	if !ok {
		return nil, ErrInputValuesWrongType
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("%w: no documents", ErrInvalidInputValues)
	}

	return docs, nil
}

// Clone creates a shallow copy of the ChainValues map.
func (cv ChainValues) Clone() ChainValues {
	return util.CopyMap(cv)
}

// CallOptions contains general options for executing a chain.
type CallOptions struct {
	CallbackManger CallbackManagerForChainRun
	Stop           []string
}

// Chain represents a sequence of calls to llms oder other utilities.
type Chain interface {
	// Call executes the chain with the given context and inputs.
	// It returns the outputs of the chain or an error, if any.
	Call(ctx context.Context, inputs ChainValues, optFns ...func(o *CallOptions)) (ChainValues, error)
	// Type returns the type of the chain.
	Type() string
	// Verbose returns the verbosity setting of the chain.
	Verbose() bool
	// Callbacks returns the callbacks associated with the chain.
	Callbacks() []Callback
	// Memory returns the memory associated with the chain.
	Memory() Memory
	// InputKeys returns the expected input keys.
	InputKeys() []string
	// OutputKeys returns the output keys the chain will return.
	OutputKeys() []string
}
