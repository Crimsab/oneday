package rag

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

// VectorStore manages embedding storage and retrieval in SQLite.
type VectorStore struct {
	db *sql.DB
}

// Chunk represents a stored text chunk with its embedding.
type Chunk struct {
	ID        int64
	StoryID   string
	Text      string
	ChunkType string // "summary", "narrator", "chapter"
	TurnStart int
	TurnEnd   int
	Embedding []float32
	CreatedAt time.Time
}

// SearchResult is a chunk paired with its cosine similarity score.
type SearchResult struct {
	Chunk      Chunk
	Similarity float64 // cosine similarity, 0.0 to 1.0
}

// NewVectorStore creates a VectorStore using the given DB connection.
func NewVectorStore(db *sql.DB) *VectorStore {
	return &VectorStore{db: db}
}

// Insert stores a chunk with its embedding BLOB.
func (vs *VectorStore) Insert(ctx context.Context, chunk *Chunk) error {
	if chunk.StoryID == "" {
		return fmt.Errorf("vectorstore: chunk must have a story_id")
	}

	blob := serializeEmbedding(chunk.Embedding)

	result, err := vs.db.ExecContext(ctx,
		`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		chunk.StoryID, chunk.Text, chunk.ChunkType,
		chunk.TurnStart, chunk.TurnEnd, blob,
	)
	if err != nil {
		return fmt.Errorf("vectorstore insert: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("vectorstore insert last id: %w", err)
	}
	chunk.ID = id
	return nil
}

// Search returns the top-K most similar chunks to the query embedding for a story.
// Uses brute-force cosine similarity — fast enough for <10K vectors per story.
func (vs *VectorStore) Search(ctx context.Context, storyID string, queryEmbedding []float32, topK int) ([]SearchResult, error) {
	rows, err := vs.db.QueryContext(ctx,
		`SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at
		 FROM rag_chunks
		 WHERE story_id = ?
		 ORDER BY id ASC`,
		storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("vectorstore search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var c Chunk
		var blob []byte
		var createdAt string
		if err := rows.Scan(
			&c.ID, &c.StoryID, &c.Text, &c.ChunkType,
			&c.TurnStart, &c.TurnEnd, &blob, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("vectorstore search scan: %w", err)
		}

		c.Embedding = deserializeEmbedding(blob)

		// Parse created_at (SQLite returns as string)
		if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			c.CreatedAt = t
		}

		sim := cosineSimilarity(queryEmbedding, c.Embedding)
		results = append(results, SearchResult{Chunk: c, Similarity: sim})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vectorstore search rows: %w", err)
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// CountByStory returns the number of chunks stored for a story.
func (vs *VectorStore) CountByStory(ctx context.Context, storyID string) (int, error) {
	var count int
	err := vs.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM rag_chunks WHERE story_id = ?`, storyID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("vectorstore count: %w", err)
	}
	return count, nil
}

// LastSummarizedTurn returns the highest turn_end among summary chunks for a story.
// Returns 0 if no summaries exist yet.
func (vs *VectorStore) LastSummarizedTurn(ctx context.Context, storyID string) (int, error) {
	var turnEnd sql.NullInt64
	err := vs.db.QueryRowContext(ctx,
		`SELECT MAX(turn_end) FROM rag_chunks WHERE story_id = ? AND chunk_type = 'summary'`,
		storyID,
	).Scan(&turnEnd)
	if err != nil {
		return 0, fmt.Errorf("vectorstore last summarized turn: %w", err)
	}
	if !turnEnd.Valid {
		return 0, nil
	}
	return int(turnEnd.Int64), nil
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// serializeEmbedding encodes a float32 slice to a byte slice (little-endian IEEE 754).
func serializeEmbedding(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		bits := math.Float32bits(f)
		buf[i*4] = byte(bits)
		buf[i*4+1] = byte(bits >> 8)
		buf[i*4+2] = byte(bits >> 16)
		buf[i*4+3] = byte(bits >> 24)
	}
	return buf
}

// deserializeEmbedding decodes a byte slice back to a float32 slice.
func deserializeEmbedding(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		bits := uint32(b[i*4]) |
			uint32(b[i*4+1])<<8 |
			uint32(b[i*4+2])<<16 |
			uint32(b[i*4+3])<<24
		v[i] = math.Float32frombits(bits)
	}
	return v
}
