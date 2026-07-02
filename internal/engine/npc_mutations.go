package engine

import (
	"database/sql"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

type npcStateStore interface {
	GetNPCByName(storyID, name string) (*storage.NPC, error)
	CreateNPC(npc *storage.NPC) error
	UpdateNPC(npc *storage.NPC) error
}

type directNPCStore struct {
	db *storage.DB
}

func (s directNPCStore) GetNPCByName(storyID, name string) (*storage.NPC, error) {
	if s.db == nil {
		return nil, nil
	}
	return s.db.GetNPCByName(storyID, name)
}

func (s directNPCStore) CreateNPC(npc *storage.NPC) error {
	if s.db == nil {
		return nil
	}
	return s.db.CreateNPC(npc)
}

func (s directNPCStore) UpdateNPC(npc *storage.NPC) error {
	if s.db == nil {
		return nil
	}
	return s.db.UpdateNPC(npc)
}

type txNPCStore struct {
	db *storage.DB
	tx *sql.Tx
}

func (s txNPCStore) GetNPCByName(storyID, name string) (*storage.NPC, error) {
	if s.db == nil || s.tx == nil {
		return nil, nil
	}
	return s.db.GetNPCByNameTx(s.tx, storyID, name)
}

func (s txNPCStore) CreateNPC(npc *storage.NPC) error {
	if s.db == nil || s.tx == nil {
		return nil
	}
	return s.db.CreateNPCTx(s.tx, npc)
}

func (s txNPCStore) UpdateNPC(npc *storage.NPC) error {
	if s.db == nil || s.tx == nil {
		return nil
	}
	return s.db.UpdateNPCTx(s.tx, npc)
}

type npcMutationRecorder struct {
	base    npcStateStore
	creates map[string]*storage.NPC
	updates map[string]*storage.NPC
}

func newNPCMutationRecorder(base npcStateStore) *npcMutationRecorder {
	return &npcMutationRecorder{
		base:    base,
		creates: make(map[string]*storage.NPC),
		updates: make(map[string]*storage.NPC),
	}
}

func (r *npcMutationRecorder) GetNPCByName(storyID, name string) (*storage.NPC, error) {
	if r == nil {
		return nil, nil
	}
	for _, npc := range r.creates {
		if npcNameMatches(npc, storyID, name) {
			return cloneNPC(npc), nil
		}
	}
	for _, npc := range r.updates {
		if npcNameMatches(npc, storyID, name) {
			return cloneNPC(npc), nil
		}
	}
	if r.base == nil {
		return nil, nil
	}
	npc, err := r.base.GetNPCByName(storyID, name)
	if err != nil || npc == nil {
		return npc, err
	}
	return cloneNPC(npc), nil
}

func (r *npcMutationRecorder) CreateNPC(npc *storage.NPC) error {
	if r == nil || npc == nil {
		return nil
	}
	r.creates[npc.ID] = cloneNPC(npc)
	delete(r.updates, npc.ID)
	return nil
}

func (r *npcMutationRecorder) UpdateNPC(npc *storage.NPC) error {
	if r == nil || npc == nil {
		return nil
	}
	if _, created := r.creates[npc.ID]; created {
		r.creates[npc.ID] = cloneNPC(npc)
		return nil
	}
	r.updates[npc.ID] = cloneNPC(npc)
	return nil
}

func (r *npcMutationRecorder) Commit(store npcStateStore) error {
	if r == nil || store == nil {
		return nil
	}
	for _, id := range sortedNPCIDs(r.creates) {
		if err := store.CreateNPC(cloneNPC(r.creates[id])); err != nil {
			return err
		}
	}
	for _, id := range sortedNPCIDs(r.updates) {
		if _, created := r.creates[id]; created {
			continue
		}
		if err := store.UpdateNPC(cloneNPC(r.updates[id])); err != nil {
			return err
		}
	}
	return nil
}

func sortedNPCIDs(m map[string]*storage.NPC) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func cloneNPC(npc *storage.NPC) *storage.NPC {
	if npc == nil {
		return nil
	}
	cloned := *npc
	return &cloned
}

func npcNameMatches(npc *storage.NPC, storyID, name string) bool {
	return npc != nil &&
		npc.IsAlive &&
		strings.EqualFold(strings.TrimSpace(npc.StoryID), strings.TrimSpace(storyID)) &&
		strings.EqualFold(strings.TrimSpace(npc.Name), strings.TrimSpace(name))
}
