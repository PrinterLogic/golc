package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/callback"
	"github.com/hupe1980/golc/chain"
	"github.com/hupe1980/golc/documentloader"
	"github.com/hupe1980/golc/llm"
	"github.com/hupe1980/golc/schema"
)

func main() {
	ctx := context.Background()

	golc.Verbose = true

	openai, err := llm.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	info := callback.NewOpenAIHandler()

	llmSummarizationChain, err := chain.NewStuffSummarizationChain(openai, func(o *chain.StuffSummarizationChainOptions) {
		o.Callbacks = []schema.Callback{callback.NewStdOutHandler(), info}
	})
	if err != nil {
		log.Fatal(err)
	}

	doc := `Large Language Models (LLMs) refer to advanced artificial intelligence models, 
	such as OpenAI's GPT-3.5, that are designed to process and generate human-like text 
	based on vast amounts of pre-existing data. LLMs utilize deep learning techniques and 
	natural language processing algorithms to understand and respond to a wide range of 
	prompts and queries. These models are trained on diverse sources of information, 
	including books, articles, websites, and other textual data, enabling them to provide 
	comprehensive and contextually relevant information on various topics. LLMs have the 
	ability to generate coherent and coherent text, engage in conversations, answer questions, 
	provide suggestions, and assist in various language-related tasks. They are used in 
	applications like chatbots, language translation, content generation, and personalized 
	assistance, among others, to enhance human-computer interactions and support language-based tasks.`

	docs, err := documentloader.NewTextLoader(strings.NewReader(doc)).Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	completion, err := llmSummarizationChain.Run(ctx, docs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(completion)
	fmt.Println("---")
	fmt.Println(info)
}
