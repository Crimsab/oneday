package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

// AppendChatMessage inserts a chat message into the DB.
func (db *DB) AppendChatMessage(m *ChatMessage) error {
	return appendChatMessageExec(db.conn, m)
}

// AppendChatMessageTx inserts a chat message into the DB inside an existing transaction.
func (db *DB) AppendChatMessageTx(tx *sql.Tx, m *ChatMessage) error {
	return appendChatMessageExec(tx, m)
}

func appendChatMessageExec(exec sqlExecer, m *ChatMessage) error {
	result, err := exec.Exec(
		`INSERT INTO chat_messages (session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.SessionID, m.StoryID, m.Turn, m.Role, m.Content, m.MessageType, m.MetadataJSON, m.CreatedAt, m.BranchID, m.SourceCommitID,
	)
	if err != nil {
		return fmt.Errorf("inserting chat message: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting inserted chat message id: %w", err)
	}
	m.ID = id
	return nil
}

// GetStoryTurnCursor returns the next canonical turn number for a story.
// It prefers the highest value between world_state.current_turn and the latest
// committed assistant turn in canonical chat history, while ignoring meta-only
// entries such as /narrator and combat summaries.
func (db *DB) GetStoryTurnCursor(storyID string) (int, error) {
	var worldTurn int
	err := db.conn.QueryRow(
		`SELECT current_turn FROM world_state WHERE story_id = ?`,
		storyID,
	).Scan(&worldTurn)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("loading world turn cursor for story %s: %w", storyID, err)
		}
		worldTurn = 0
	}

	var latestCommittedTurn sql.NullInt64
	if err := db.conn.QueryRow(
		`SELECT MAX(turn)
         FROM chat_messages
         WHERE story_id = ?
           AND role = 'assistant'
           AND message_type NOT IN ('narrator', 'combat_summary')`,
		storyID,
	).Scan(&latestCommittedTurn); err != nil {
		return 0, fmt.Errorf("loading chat turn cursor for story %s: %w", storyID, err)
	}

	nextHistoryTurn := 0
	if latestCommittedTurn.Valid {
		nextHistoryTurn = int(latestCommittedTurn.Int64) + 1
	}
	if nextHistoryTurn > worldTurn {
		return nextHistoryTurn, nil
	}
	return worldTurn, nil
}

// GetRecentMessages returns the last N messages for a session, ordered chronologically (ASC).
func (db *DB) GetRecentMessages(sessionID string, limit int) ([]ChatMessage, error) {
	// Fetch last N in DESC order, then reverse to get chronological order.
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE session_id = ?
         ORDER BY turn DESC, id DESC
         LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting recent messages for session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
			&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
			&m.BranchID, &m.SourceCommitID,
		); err != nil {
			return nil, fmt.Errorf("scanning chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating chat messages: %w", err)
	}

	// Reverse to return chronological (ASC) order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// GetSessionMessages returns all messages for a session, ordered chronologically (ASC).
func (db *DB) GetSessionMessages(sessionID string) ([]ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE session_id = ?
         ORDER BY turn ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting messages for session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
			&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
			&m.BranchID, &m.SourceCommitID,
		); err != nil {
			return nil, fmt.Errorf("scanning chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetStoryMessages returns all messages for a story across all sessions,
// ordered chronologically (ASC).
func (db *DB) GetStoryMessages(storyID string) ([]ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE story_id = ?
         ORDER BY created_at ASC, id ASC`,
		storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting messages for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
			&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
			&m.BranchID, &m.SourceCommitID,
		); err != nil {
			return nil, fmt.Errorf("scanning story chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetLatestAssistantMessageByStory returns the latest committed assistant
// narrative message for a story, ignoring meta-only entries.
func (db *DB) GetLatestAssistantMessageByStory(storyID string) (*ChatMessage, error) {
	row := db.conn.QueryRow(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE story_id = ?
           AND role = 'assistant'
           AND message_type NOT IN ('narrator', 'combat_summary')
         ORDER BY turn DESC, id DESC
         LIMIT 1`,
		storyID,
	)
	var m ChatMessage
	if err := row.Scan(
		&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
		&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
		&m.BranchID, &m.SourceCommitID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting latest assistant message for story %s: %w", storyID, err)
	}
	return &m, nil
}

// GetStoryMessagesByTurnRange returns all messages for a story within a turn range [turnStart, turnEnd],
// ordered chronologically. Used by the RAG summarizer to fetch unsummarized turns.
func (db *DB) GetStoryMessagesByTurnRange(storyID string, turnStart, turnEnd int) ([]ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE story_id = ? AND turn >= ? AND turn <= ?
           AND role IN ('user', 'assistant')
         ORDER BY turn ASC, id ASC`,
		storyID, turnStart, turnEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("getting story messages by turn range: %w", err)
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
			&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
			&m.BranchID, &m.SourceCommitID,
		); err != nil {
			return nil, fmt.Errorf("scanning story message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetRecentMessagesByStory returns the last N messages for a story across all
// sessions, ordered chronologically (ASC). Used by ResumeNarration to rebuild
// context when a new session is started for an already-played story.
func (db *DB) GetRecentMessagesByStory(storyID string, limit int) ([]ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
         FROM chat_messages
         WHERE story_id = ?
         ORDER BY turn DESC, id DESC
         LIMIT ?`,
		storyID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting recent messages for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.StoryID, &m.Turn, &m.Role,
			&m.Content, &m.MessageType, &m.MetadataJSON, &m.CreatedAt,
			&m.BranchID, &m.SourceCommitID,
		); err != nil {
			return nil, fmt.Errorf("scanning chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating chat messages: %w", err)
	}
	// Reverse to return chronological (ASC) order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// CountSessionMessages returns the number of messages in a session.
func (db *DB) CountSessionMessages(sessionID string) (int, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM chat_messages WHERE session_id = ?`, sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting messages for session %s: %w", sessionID, err)
	}
	return count, nil
}

// UpdateAssistantMessageMetadata replaces metadata_json for the latest assistant
// message of a given story/turn pair.
func (db *DB) UpdateAssistantMessageMetadata(storyID string, turn int, metadataJSON string) error {
	_, err := db.conn.Exec(
		`UPDATE chat_messages
         SET metadata_json = ?
         WHERE id = (
           SELECT id FROM chat_messages
           WHERE story_id = ? AND turn = ? AND role = 'assistant'
           ORDER BY id DESC
           LIMIT 1
         )`,
		metadataJSON, storyID, turn,
	)
	if err != nil {
		return fmt.Errorf("updating assistant metadata for story %s turn %d: %w", storyID, turn, err)
	}
	return nil
}
