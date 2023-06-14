package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hupe1980/golc/chain"
	"github.com/hupe1980/golc/llm"
)

func main() {
	openai, err := llm.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	bashChain, err := chain.NewLLMBashChainFromLLM(openai)
	if err != nil {
		log.Fatal(err)
	}

	result, err := chain.Run(context.Background(), bashChain, "Please write a bash script that prints 'Hello World' to the console.")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}