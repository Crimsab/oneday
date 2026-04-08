package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// ChapterManager handles the chapter lifecycle: creation, transitions, and summaries.
type ChapterManager struct {
	db      *storage.DB
	storyID string
	rag     *rag.RAG // optional — for embedding chapter summaries
	router  *ai.Router
}

// NewChapterManager creates a new ChapterManager.
func NewChapterManager(db *storage.DB, storyID string, ragPipeline *rag.RAG, router *ai.Router) *ChapterManager {
	return &ChapterManager{
		db:      db,
		storyID: storyID,
		rag:     ragPipeline,
		router:  router,
	}
}

// EnsureCurrentChapter creates chapter 1 if no chapter exists for the story.
// Called once on narrator initialization.
func (cm *ChapterManager) EnsureCurrentChapter(startTurn int) error {
	existing, err := cm.db.GetCurrentChapter(cm.storyID)
	if err != nil {
		return fmt.Errorf("checking current chapter: %w", err)
	}
	if existing != nil {
		return nil // already has an open chapter
	}

	// Check if any chapter exists at all (in case all are closed somehow).
	chapters, err := cm.db.ListChapters(cm.storyID)
	if err != nil {
		return fmt.Errorf("listing chapters: %w", err)
	}
	if len(chapters) > 0 {
		return nil // chapters exist, even if closed
	}

	ch := &storage.Chapter{
		StoryID:       cm.storyID,
		ChapterNumber: 1,
		Title:         "Chapter 1",
		Summary:       "",
		StartTurn:     startTurn,
		CreatedAt:     time.Now(),
	}
	if err := cm.db.CreateChapter(ch); err != nil {
		return fmt.Errorf("creating initial chapter: %w", err)
	}
	return nil
}

// HandleChapterEnd closes the current chapter (generating an AI summary) and opens the next one.
// chapterTitle is provided by the AI in the response. currentTurn is the turn number at chapter end.
func (cm *ChapterManager) HandleChapterEnd(ctx context.Context, currentTurn int, chapterTitle string) error {
	// Get the current open chapter.
	current, err := cm.db.GetCurrentChapter(cm.storyID)
	if err != nil {
		return fmt.Errorf("getting current chapter: %w", err)
	}
	if current == nil {
		// No open chapter — create one and close it immediately.
		current = &storage.Chapter{
			StoryID:       cm.storyID,
			ChapterNumber: 1,
			Title:         chapterTitle,
			StartTurn:     0,
			CreatedAt:     time.Now(),
		}
		if err := cm.db.CreateChapter(current); err != nil {
			return fmt.Errorf("creating chapter for end: %w", err)
		}
	}

	// Update title if provided.
	if chapterTitle != "" && chapterTitle != current.Title {
		current.Title = chapterTitle
		if err := cm.db.UpdateChapterTitle(cm.storyID, current.ChapterNumber, chapterTitle); err != nil {
			_ = err // non-fatal
		}
	}

	// Fetch messages for this chapter to generate a summary.
	summary, err := cm.generateChapterSummary(ctx, current.StartTurn, currentTurn)
	if err != nil {
		// Non-fatal: store empty summary.
		summary = ""
	}

	// Close the current chapter.
	if err := cm.db.UpdateChapterEnd(cm.storyID, current.ChapterNumber, currentTurn, summary); err != nil {
		return fmt.Errorf("closing chapter %d: %w", current.ChapterNumber, err)
	}

	// Embed the chapter summary into RAG for long-term memory.
	if cm.rag != nil && summary != "" {
		chunkText := fmt.Sprintf("[Chapter %d: %s]\n%s", current.ChapterNumber, current.Title, summary)
		go func() {
			bgCtx := context.Background()
			_ = cm.rag.StoreChunk(bgCtx, cm.storyID, chunkText, "chapter", current.StartTurn, currentTurn)
		}()
	}

	// Open the next chapter.
	nextNum := current.ChapterNumber + 1
	nextChapter := &storage.Chapter{
		StoryID:       cm.storyID,
		ChapterNumber: nextNum,
		Title:         fmt.Sprintf("Chapter %d", nextNum),
		Summary:       "",
		StartTurn:     currentTurn + 1,
		CreatedAt:     time.Now(),
	}
	if err := cm.db.CreateChapter(nextChapter); err != nil {
		return fmt.Errorf("creating chapter %d: %w", nextNum, err)
	}

	return nil
}

// GetChapterSummaries returns all chapter summaries formatted for display (e.g., /journal).
func (cm *ChapterManager) GetChapterSummaries() (string, error) {
	chapters, err := cm.db.ListChapters(cm.storyID)
	if err != nil {
		return "", fmt.Errorf("listing chapters: %w", err)
	}
	if len(chapters) == 0 {
		return "No chapters yet.", nil
	}

	var sb strings.Builder
	for _, ch := range chapters {
		status := ""
		if ch.EndTurn == nil {
			status = " (current)"
		}
		sb.WriteString(fmt.Sprintf("## Chapter %d: %s%s\n", ch.ChapterNumber, ch.Title, status))
		if ch.Summary != "" {
			sb.WriteString(ch.Summary)
			sb.WriteString("\n")
		} else {
			sb.WriteString("_(in progress)_\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// chapterSummaryResponse is the JSON structure returned by the chapter summary AI call.
type chapterSummaryResponse struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// generateChapterSummary fetches chapter messages and calls AI to produce a summary.
func (cm *ChapterManager) generateChapterSummary(ctx context.Context, startTurn, endTurn int) (string, error) {
	if cm.router == nil {
		return "", nil
	}

	// Fetch messages for this chapter's turn range.
	msgs, err := cm.db.GetStoryMessagesByTurnRange(cm.storyID, startTurn, endTurn)
	if err != nil {
		return "", fmt.Errorf("fetching chapter messages: %w", err)
	}
	if len(msgs) == 0 {
		return "", nil
	}

	// Build a transcript from the messages.
	var transcript strings.Builder
	for _, m := range msgs {
		role := "Player"
		if m.Role == "assistant" {
			role = "Narrator"
		}
		transcript.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, m.Content))
	}

	story, err := cm.db.GetStory(cm.storyID)
	if err != nil {
		return "", fmt.Errorf("getting story for chapter summary: %w", err)
	}

	// Call AI with the chapter summary prompt.
	req := ai.Request{
		Messages: []ai.Message{
			{
				Role: ai.RoleSystem,
				Content: prompts.ChapterSummarySystem(
					story.Language,
					story.WritingStyle,
					story.PromptDirectives,
				),
			},
			{Role: ai.RoleUser, Content: prompts.ChapterSummaryUser(transcript.String())},
		},
		Temperature:    0.5,
		MaxTokens:      1024,
		ResponseFormat: ai.ChapterSummaryResponseFormat(),
	}

	resp, err := cm.router.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("AI chapter summary call: %w", err)
	}

	// Parse the JSON response.
	var summaryResp chapterSummaryResponse
	raw := extractJSONFromResponse(resp.Content)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &summaryResp); err == nil && summaryResp.Summary != "" {
			return summaryResp.Summary, nil
		}
	}

	// Fallback: return raw content if JSON parsing fails.
	return resp.Content, nil
}

// extractJSONFromResponse extracts a ```json ... ``` block from an AI response.
func extractJSONFromResponse(text string) string {
	raw, err := ai.ExtractJSONPayload(text)
	if err == nil {
		return raw
	}
	return ""
}
