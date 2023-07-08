// Package chatmodel provides functionalities for working with Large Language Models (LLMs).
package chatmodel

import "github.com/hupe1980/golc/schema"

func newChatGeneraton(text string) schema.Generation {
	return schema.Generation{
		Text:    text,
		Message: schema.NewAIChatMessage(text),
	}
}
