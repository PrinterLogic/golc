package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/chain"
	"github.com/hupe1980/golc/llm"
)

type mockRetriever struct{}

func (r *mockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]golc.Document, error) {
	return []golc.Document{
		{PageContent: "Why don't scientists trust atoms? Because they make up everything!"},
		{PageContent: "Why did the bicycle fall over? Because it was two-tired!"},
	}, nil
}

func main() {
	openai, err := llm.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	retrievalQAChain, err := chain.NewRetrievalQAFromLLM(openai, &mockRetriever{})
	if err != nil {
		log.Fatal(err)
	}

	result, err := chain.Run(context.Background(), retrievalQAChain, "Why don't scientists trust atoms?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
