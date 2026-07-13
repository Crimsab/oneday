package rag

import (
	"container/heap"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// VectorStore manages embedding storage and retrieval in SQLite.
type VectorStore struct {
	db           *sql.DB
	lineage      bool
	durableNorms bool
}

// Chunk represents a stored text chunk with its embedding.
type Chunk struct {
	ID             int64
	StoryID        string
	Text           string
	ChunkType      string // "summary", "narrator", "chapter"
	TurnStart      int
	TurnEnd        int
	Embedding      []float32
	CreatedAt      time.Time
	BranchID       string
	SourceCommitID string
}

// SearchResult is a chunk paired with its cosine similarity score.
type SearchResult struct {
	Chunk      Chunk
	Similarity float64 // cosine similarity, 0.0 to 1.0
}

type searchResultMinHeap []SearchResult

func (h searchResultMinHeap) Len() int           { return len(h) }
func (h searchResultMinHeap) Less(i, j int) bool { return h[i].Similarity < h[j].Similarity }
func (h searchResultMinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *searchResultMinHeap) Push(value any)    { *h = append(*h, value.(SearchResult)) }
func (h *searchResultMinHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

// NewVectorStore creates a VectorStore using the given DB connection.
func NewVectorStore(db *sql.DB) *VectorStore {
	return &VectorStore{
		db:           db,
		lineage:      ragColumnExists(db, "branch_id") && ragColumnExists(db, "source_commit_id"),
		durableNorms: ragColumnExists(db, "embedding_norm"),
	}
}

func ragColumnExists(db *sql.DB, column string) bool {
	if db == nil {
		return false
	}
	rows, err := db.Query(`PRAGMA table_info(rag_chunks)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk) == nil && name == column {
			return true
		}
	}
	return false
}

// Insert stores a chunk with its embedding BLOB.
func (vs *VectorStore) Insert(ctx context.Context, chunk *Chunk) error {
	if chunk.StoryID == "" {
		return fmt.Errorf("vectorstore: chunk must have a story_id")
	}

	blob := serializeEmbedding(chunk.Embedding)
	norm := vectorNorm(chunk.Embedding)

	var result sql.Result
	var err error
	if vs.lineage && vs.durableNorms {
		result, err = vs.db.ExecContext(ctx,
			`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding, embedding_norm, branch_id, source_commit_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?,
		   COALESCE((SELECT active_branch_id FROM stories WHERE id=?), ''),
		   COALESCE((SELECT b.head_commit_id FROM story_branches b JOIN stories s ON s.active_branch_id=b.id WHERE s.id=?), ''))`,
			chunk.StoryID, chunk.Text, chunk.ChunkType,
			chunk.TurnStart, chunk.TurnEnd, blob, norm, chunk.StoryID, chunk.StoryID,
		)
	} else if vs.lineage {
		result, err = vs.db.ExecContext(ctx,
			`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding, branch_id, source_commit_id)
		 VALUES (?, ?, ?, ?, ?, ?,
		   COALESCE((SELECT active_branch_id FROM stories WHERE id=?), ''),
		   COALESCE((SELECT b.head_commit_id FROM story_branches b JOIN stories s ON s.active_branch_id=b.id WHERE s.id=?), ''))`,
			chunk.StoryID, chunk.Text, chunk.ChunkType,
			chunk.TurnStart, chunk.TurnEnd, blob, chunk.StoryID, chunk.StoryID,
		)
	} else if vs.durableNorms {
		result, err = vs.db.ExecContext(ctx,
			`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding, embedding_norm) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			chunk.StoryID, chunk.Text, chunk.ChunkType, chunk.TurnStart, chunk.TurnEnd, blob, norm)
	} else {
		result, err = vs.db.ExecContext(ctx,
			`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding) VALUES (?, ?, ?, ?, ?, ?)`,
			chunk.StoryID, chunk.Text, chunk.ChunkType, chunk.TurnStart, chunk.TurnEnd, blob)
	}
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
	queryNorm := vectorNorm(queryEmbedding)
	selectSQL := `SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at FROM rag_chunks WHERE story_id = ? ORDER BY id ASC`
	if vs.lineage {
		selectSQL = `SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at, branch_id, source_commit_id
		 FROM rag_chunks WHERE story_id = ? ORDER BY id ASC`
	}
	if vs.durableNorms {
		if vs.lineage {
			selectSQL = `SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at, branch_id, source_commit_id, embedding_norm
			 FROM rag_chunks WHERE story_id = ? ORDER BY id ASC`
		} else {
			selectSQL = `SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at, embedding_norm
			 FROM rag_chunks WHERE story_id = ? ORDER BY id ASC`
		}
	}
	rows, err := vs.db.QueryContext(ctx, selectSQL, storyID)
	if err != nil {
		return nil, fmt.Errorf("vectorstore search query: %w", err)
	}
	defer rows.Close()

	best := searchResultMinHeap{}
	results := []SearchResult{}
	missingNorms := map[int64]float64{}
	for rows.Next() {
		var chunk Chunk
		var blob []byte
		var createdAt string
		var storedNorm float64
		dest := []any{&chunk.ID, &chunk.StoryID, &chunk.Text, &chunk.ChunkType, &chunk.TurnStart, &chunk.TurnEnd, &blob, &createdAt}
		if vs.lineage {
			dest = append(dest, &chunk.BranchID, &chunk.SourceCommitID)
		}
		if vs.durableNorms {
			dest = append(dest, &storedNorm)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("vectorstore search scan: %w", err)
		}
		chunk.Embedding = deserializeEmbedding(blob)
		if parsed, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			chunk.CreatedAt = parsed
		}
		if storedNorm <= 0 {
			storedNorm = vectorNorm(chunk.Embedding)
			if vs.durableNorms && storedNorm > 0 {
				missingNorms[chunk.ID] = storedNorm
			}
		}
		result := SearchResult{Chunk: chunk, Similarity: cosineSimilarityWithNorm(queryEmbedding, queryNorm, chunk.Embedding, storedNorm)}
		if topK > 0 {
			if len(best) < topK {
				heap.Push(&best, result)
			} else if result.Similarity > best[0].Similarity {
				heap.Pop(&best)
				heap.Push(&best, result)
			}
		} else {
			results = append(results, result)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vectorstore search rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("vectorstore closing search rows: %w", err)
	}
	if len(missingNorms) > 0 {
		if err := vs.persistEmbeddingNorms(ctx, missingNorms); err != nil {
			log.Printf("[rag] persisting embedding norms failed: %v", err)
		}
	}
	if topK > 0 {
		results = []SearchResult(best)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
	return results, nil
}

func (vs *VectorStore) persistEmbeddingNorms(ctx context.Context, norms map[int64]float64) error {
	tx, err := vs.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vectorstore norm transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for id, norm := range norms {
		if _, err := tx.ExecContext(ctx, `UPDATE rag_chunks SET embedding_norm=? WHERE id=? AND embedding_norm=0`, norm, id); err != nil {
			return fmt.Errorf("vectorstore persist norm: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vectorstore commit norms: %w", err)
	}
	return nil
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

// PruneDimensionMismatches removes chunks whose serialized embedding length does
// not match the configured embedding dimensions. This keeps model switches from
// silently poisoning cosine search with incompatible vectors.
func (vs *VectorStore) PruneDimensionMismatches(ctx context.Context, storyID string, dimensions int) (int64, error) {
	if storyID == "" || dimensions <= 0 {
		return 0, nil
	}
	result, err := vs.db.ExecContext(ctx,
		`DELETE FROM rag_chunks
		 WHERE story_id = ?
		   AND embedding IS NOT NULL
		   AND length(embedding) != ?`,
		storyID, dimensions*4,
	)
	if err != nil {
		return 0, fmt.Errorf("vectorstore prune dimension mismatches: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("vectorstore prune rows affected: %w", err)
	}
	return removed, nil
}

// DeleteByStory removes all RAG chunks for a story so they can be regenerated.
func (vs *VectorStore) DeleteByStory(ctx context.Context, storyID string) (int64, error) {
	if storyID == "" {
		return 0, nil
	}
	result, err := vs.db.ExecContext(ctx, `DELETE FROM rag_chunks WHERE story_id = ?`, storyID)
	if err != nil {
		return 0, fmt.Errorf("vectorstore delete by story: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("vectorstore delete rows affected: %w", err)
	}
	return removed, nil
}

// LastSummarizedTurn returns the highest turn_end among summary chunks for a story.
// Returns -1 if no summaries exist yet so committed turn 0 remains summarizable.
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
		return -1, nil
	}
	return int(turnEnd.Int64), nil
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	return cosineSimilarityWithNorm(a, vectorNorm(a), b, vectorNorm(b))
}

func cosineSimilarityWithNorm(a []float32, normA float64, b []float32, normB float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func vectorNorm(v []float32) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, value := range v {
		sum += float64(value) * float64(value)
	}
	return sum
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
