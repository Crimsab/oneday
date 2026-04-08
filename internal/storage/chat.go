package storage

import "fmt"

// AppendChatMessage inserts a chat message into the DB.
func (db *DB) AppendChatMessage(m *ChatMessage) error {
	result, err := db.conn.Exec(
		`INSERT INTO chat_messages (session_id, story_id, turn, role, content, message_type, metadata_json, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.SessionID, m.StoryID, m.Turn, m.Role, m.Content, m.MessageType, m.MetadataJSON, m.CreatedAt,
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

// GetRecentMessages returns the last N messages for a session, ordered chronologically (ASC).
func (db *DB) GetRecentMessages(sessionID string, limit int) ([]ChatMessage, error) {
	// Fetch last N in DESC order, then reverse to get chronological order.
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at
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
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at
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
		); err != nil {
			return nil, fmt.Errorf("scanning chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetStoryMessagesByTurnRange returns all messages for a story within a turn range [turnStart, turnEnd],
// ordered chronologically. Used by the RAG summarizer to fetch unsummarized turns.
func (db *DB) GetStoryMessagesByTurnRange(storyID string, turnStart, turnEnd int) ([]ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at
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
		`SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at
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
