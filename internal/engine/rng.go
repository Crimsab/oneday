package engine

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// RollRecord captures one mechanical random event for replay/debug surfaces.
type RollRecord struct {
	ID        string     `json:"id"`
	Source    string     `json:"source"`
	Die       string     `json:"die"`
	Sides     int        `json:"sides"`
	Raw       int        `json:"raw"`
	Modifiers []Modifier `json:"modifiers,omitempty"`
	Total     int        `json:"total,omitempty"`
	Target    int        `json:"target,omitempty"`
	Outcome   string     `json:"outcome,omitempty"`
	Sequence  int        `json:"sequence"`
	Seed      int64      `json:"seed"`
}

// RNGService owns a deterministic random stream and keeps a compact roll log.
type RNGService struct {
	mu       sync.Mutex
	seed     int64
	rng      *rand.Rand
	sequence int
	log      []RollRecord
}

type rngSnapshot struct {
	seed     int64
	sequence int
	log      []RollRecord
}

func (r *RNGService) snapshot() rngSnapshot {
	if r == nil {
		return rngSnapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	logCopy := append([]RollRecord(nil), r.log...)
	return rngSnapshot{seed: r.seed, sequence: r.sequence, log: logCopy}
}

func (r *RNGService) restore(snapshot rngSnapshot) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seed = snapshot.seed
	r.rng = rand.New(rand.NewSource(snapshot.seed))
	for _, record := range snapshot.log {
		sides := record.Sides
		if sides <= 0 {
			sides = 1
		}
		r.rng.Intn(sides)
	}
	r.sequence = snapshot.sequence
	r.log = append([]RollRecord(nil), snapshot.log...)
}

func NewRNGService(seed int64) *RNGService {
	return &RNGService{
		seed: seed,
		rng:  rand.New(rand.NewSource(seed)),
	}
}

func NewDefaultRNGService() *RNGService {
	return NewRNGService(time.Now().UnixNano())
}

func (r *RNGService) Roll(source string, sides int) RollRecord {
	if r == nil {
		r = defaultRNGService()
	}
	if sides <= 0 {
		sides = 1
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.sequence++
	raw := r.rng.Intn(sides) + 1
	record := RollRecord{
		ID:       fmt.Sprintf("roll:%d", r.sequence),
		Source:   source,
		Die:      fmt.Sprintf("d%d", sides),
		Sides:    sides,
		Raw:      raw,
		Total:    raw,
		Sequence: r.sequence,
		Seed:     r.seed,
	}
	r.log = append(r.log, record)
	return record
}

func (r *RNGService) RollLog() []RollRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]RollRecord, len(r.log))
	copy(out, r.log)
	return out
}

func (r *RNGService) Seed() int64 {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seed
}

var (
	defaultRNGMu sync.Mutex
	defaultRNG   = NewDefaultRNGService()
)

func defaultRNGService() *RNGService {
	defaultRNGMu.Lock()
	defer defaultRNGMu.Unlock()
	return defaultRNG
}

// RollD100 rolls a 100-sided die. Returns a value in [1, 100].
func RollD100() int {
	return defaultRNGService().Roll("legacy.d100", 100).Raw
}

// RollD20 rolls a 20-sided die. Returns a value in [1, 20].
func RollD20() int {
	return defaultRNGService().Roll("legacy.d20", 20).Raw
}
