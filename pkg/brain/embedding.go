package brain

// Embedding is a vector representation for semantic search
// In a real system, you would use a model to generate these
// and a vector DB for fast similarity search.
type Embedding []float32

// Embeddable is any object that can be embedded semantically
// (e.g., message, file, event)
type Embeddable interface {
	Text() string
}

// Embedder is an interface for generating embeddings
// You can implement this using OpenAI, Cohere, or local models
// For now, this is a stub.
type Embedder interface {
	Embed(text string) (Embedding, error)
}

// VectorMemory is a stub for a vector DB (e.g., Faiss, Milvus, Pinecone)
type VectorMemory interface {
	Add(key string, emb Embedding) error
	Search(query Embedding, topK int) ([]string, error)
}
