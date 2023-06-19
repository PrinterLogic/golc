package vectorstore

import (
	"context"

	"github.com/hupe1980/golc/integration/pinecone"
	"github.com/hupe1980/golc/schema"
)

// Compile time check to ensure Pinecone satisfies the VectorStore interface.
var _ schema.VectorStore = (*Pinecone)(nil)

type PineconeOptions struct {
	Namespace string
}

type Pinecone struct {
	client   pinecone.Client
	embedder schema.Embedder
	textKey  string
	opts     PineconeOptions
}

func NewPinecone(client pinecone.Client, embedder schema.Embedder, textKey string, optFns ...func(*PineconeOptions)) (*Pinecone, error) {
	opts := PineconeOptions{}

	for _, fn := range optFns {
		fn(&opts)
	}

	return &Pinecone{
		client:   client,
		embedder: embedder,
		textKey:  textKey,
		opts:     opts,
	}, nil
}

func (vs *Pinecone) AddDocuments(ctx context.Context, docs []schema.Document) error {
	texts := make([]string, 0, len(docs))
	for _, doc := range docs {
		texts = append(texts, doc.PageContent)
	}

	vectors, err := vs.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return err
	}

	metadata := make([]map[string]any, 0, len(docs))

	for i := 0; i < len(docs); i++ {
		m := make(map[string]any, len(docs[i].Metadata))
		for key, value := range docs[i].Metadata {
			m[key] = value
		}

		m[vs.textKey] = texts[i]

		metadata = append(metadata, m)
	}

	pineconeVectors, err := pinecone.ToPineconeVectors(vectors, metadata)
	if err != nil {
		return err
	}

	req := &pinecone.UpsertRequest{
		Vectors: pineconeVectors,
	}

	if vs.opts.Namespace != "" {
		req.Namespace = vs.opts.Namespace
	}

	_, err = vs.client.Upsert(ctx, req)

	return err
}

func (vs *Pinecone) SimilaritySearch(ctx context.Context, query string) ([]schema.Document, error) {
	return nil, nil
}