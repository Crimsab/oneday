package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/game/contracts"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/storage"
)

type Server struct {
	db    *storage.DB
	turns gameservice.TurnService
}

func New(db *storage.DB) *Server {
	return &Server{db: db}
}

func NewWithTurnService(db *storage.DB, turns gameservice.TurnService) *Server {
	return &Server{db: db, turns: turns}
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:8787"
	}
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/stories", s.handleStories)
	mux.HandleFunc("/api/stories/", s.handleStoryRoute)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	stories, err := s.db.ListStories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	type storyView struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description,omitempty"`
		Genre       string    `json:"genre,omitempty"`
		Tone        string    `json:"tone,omitempty"`
		Language    string    `json:"language,omitempty"`
		IsArchived  bool      `json:"is_archived"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	out := make([]storyView, 0, len(stories))
	for _, story := range stories {
		out = append(out, storyView{
			ID:          story.ID,
			Name:        story.Name,
			Description: story.Description,
			Genre:       story.Genre,
			Tone:        story.Tone,
			Language:    story.Language,
			IsArchived:  story.IsArchived,
			UpdatedAt:   story.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleStoryRoute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/stories/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "snapshot":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleSnapshot(w, r, parts[0])
	case "turns":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleSubmitTurn(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleSnapshot(w http.ResponseWriter, _ *http.Request, storyID string) {
	if s.turns != nil {
		snapshot, err := s.turns.Snapshot(context.Background(), storyID)
		if err == nil {
			writeJSON(w, http.StatusOK, snapshot)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	story, err := s.db.GetStory(storyID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	world, err := s.db.GetWorldState(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	character, err := s.db.GetCharacterByStory(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, struct {
		contracts.GameSnapshot
		StoryName     string `json:"story_name"`
		CharacterName string `json:"character_name"`
	}{
		GameSnapshot: contracts.GameSnapshot{
			StoryID:  story.ID,
			Turn:     world.CurrentTurn,
			Location: world.CurrentLocation,
		},
		StoryName:     story.Name,
		CharacterName: character.Name,
	})
}

func (s *Server) handleSubmitTurn(w http.ResponseWriter, r *http.Request, storyID string) {
	if s.turns == nil {
		writeError(w, http.StatusNotImplemented, errors.New("turn service is not configured"))
		return
	}
	defer r.Body.Close()

	var req contracts.SubmitActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err))
		return
	}
	req.StoryID = storyID
	events, err := s.turns.SubmitAction(r.Context(), req)
	if err != nil {
		writeError(w, statusForTurnError(err), err)
		return
	}

	out := make([]contracts.TurnEvent, 0, 8)
	for event := range events {
		out = append(out, event)
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

func statusForTurnError(err error) int {
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "stale client_turn"),
		strings.Contains(message, "stale session_id"),
		strings.Contains(message, "is required"),
		strings.Contains(message, "action text or choice_id is required"):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OneDay</title>
  <style>
    :root { color-scheme: dark; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #101113; color: #f4f0e8; }
    body { margin: 0; min-height: 100vh; display: grid; grid-template-columns: minmax(240px, 340px) 1fr; }
    aside { border-right: 1px solid #33363a; padding: 18px; background: #181a1d; }
    main { padding: 24px; max-width: 900px; display: grid; gap: 14px; align-content: start; }
    h1 { font-size: 18px; margin: 0 0 16px; }
    button { width: 100%; text-align: left; border: 1px solid #3b3f45; background: #202329; color: inherit; padding: 10px 12px; margin: 0 0 8px; border-radius: 6px; cursor: pointer; }
    button:hover { border-color: #9b8f6a; }
    textarea { min-height: 92px; resize: vertical; border: 1px solid #3b3f45; border-radius: 6px; background: #181a1d; color: inherit; padding: 10px; font: inherit; }
    #send { width: auto; justify-self: start; }
    pre { white-space: pre-wrap; background: #181a1d; border: 1px solid #33363a; padding: 14px; border-radius: 6px; }
    .choices button { width: auto; margin-right: 8px; }
    .muted { color: #a9a39a; }
    @media (max-width: 760px) { body { grid-template-columns: 1fr; } aside { border-right: 0; border-bottom: 1px solid #33363a; } }
  </style>
</head>
<body>
  <aside>
    <h1>OneDay</h1>
    <div id="stories" class="muted">Loading stories...</div>
  </aside>
  <main>
    <h2 id="storyTitle">Select a story</h2>
    <pre id="narrative">Snapshot appears here.</pre>
    <div id="choices" class="choices"></div>
    <textarea id="action" placeholder="Write an action..."></textarea>
    <button id="send">Send action</button>
    <pre id="snapshot">No snapshot loaded.</pre>
  </main>
  <script>
    let current = null;
    async function loadStories() {
      const list = document.getElementById('stories');
      const res = await fetch('/api/stories');
      const stories = await res.json();
      list.innerHTML = '';
      if (!stories.length) { list.textContent = 'No stories yet.'; return; }
      for (const story of stories) {
        const btn = document.createElement('button');
        btn.textContent = story.name || story.id;
        btn.onclick = () => loadSnapshot(story.id);
        list.appendChild(btn);
      }
    }
    async function loadSnapshot(id) {
      const res = await fetch('/api/stories/' + encodeURIComponent(id) + '/snapshot');
      current = await res.json();
      document.getElementById('storyTitle').textContent = current.story_name || current.story_id || id;
      document.getElementById('snapshot').textContent = JSON.stringify(current, null, 2);
      renderChoices(current.choices || []);
    }
    function renderChoices(choices) {
      const node = document.getElementById('choices');
      node.innerHTML = '';
      for (const choice of choices) {
        const btn = document.createElement('button');
        btn.textContent = choice.text;
        btn.onclick = () => submitAction({ kind: 'choice', choice_id: choice.id, text: '[Choice ' + choice.id + '] ' + choice.text });
        node.appendChild(btn);
      }
    }
    async function submitAction(actionOverride) {
      if (!current) return;
      const text = document.getElementById('action').value.trim();
      const action = actionOverride || { kind: 'free_text', text };
      if (!action.text && !action.choice_id) return;
      document.getElementById('send').disabled = true;
      try {
        const res = await fetch('/api/stories/' + encodeURIComponent(current.story_id) + '/turns', {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({
            session_id: current.session_id,
            client_turn: current.turn,
            idempotency_key: crypto.randomUUID(),
            action,
            capabilities: { images: true, ascii: true, roll_log: true }
          })
        });
        const payload = await res.json();
        if (!res.ok) throw new Error(payload.error || res.statusText);
        for (const event of payload.events || []) {
          const data = event.payload ? (typeof event.payload === 'string' ? JSON.parse(event.payload) : event.payload) : {};
          if (event.type === 'narrative.final') {
            document.getElementById('narrative').textContent = data.narrative || '';
          }
          if (event.type === 'choices.updated') renderChoices(data.choices || []);
          if (event.type === 'turn.committed') {
            current = data.snapshot;
            document.getElementById('snapshot').textContent = JSON.stringify(current, null, 2);
          }
        }
        document.getElementById('action').value = '';
      } catch (err) {
        document.getElementById('narrative').textContent = String(err);
      } finally {
        document.getElementById('send').disabled = false;
      }
    }
    document.getElementById('send').onclick = () => submitAction();
    loadStories().catch(err => { document.getElementById('stories').textContent = String(err); });
  </script>
</body>
</html>`

func ServeURL(addr string) string {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:8787"
	}
	return fmt.Sprintf("http://%s", addr)
}
