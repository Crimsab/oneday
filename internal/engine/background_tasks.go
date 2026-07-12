package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	ragBackgroundWorkers  = 2
	ragBackgroundCapacity = 128
	ragBackgroundTimeout  = 90 * time.Second
)

type storyTask struct {
	key string
	run func(context.Context) error
}

type storyTaskState struct {
	queue     []storyTask
	pending   map[string]int
	running   bool
	scheduled bool
}

// storyTaskExecutor bounds background work globally, serializes work for each
// story, and replaces queued tasks with the same key with their newest version.
type storyTaskExecutor struct {
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	capacity int
	ready    chan string

	mu     sync.Mutex
	total  int
	states map[string]*storyTaskState
}

func newStoryTaskExecutor(parent context.Context, workers, capacity int, timeout time.Duration) *storyTaskExecutor {
	if parent == nil {
		parent = context.Background()
	}
	if workers <= 0 {
		workers = 1
	}
	if capacity <= 0 {
		capacity = 1
	}
	if timeout <= 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithCancel(parent)
	e := &storyTaskExecutor{
		ctx: ctx, cancel: cancel, timeout: timeout, capacity: capacity,
		ready: make(chan string, capacity), states: make(map[string]*storyTaskState),
	}
	for i := 0; i < workers; i++ {
		go e.worker()
	}
	return e
}

func (e *storyTaskExecutor) Submit(storyID, key string, run func(context.Context) error) bool {
	if e == nil || storyID == "" || key == "" || run == nil || e.ctx.Err() != nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.states[storyID]
	if state == nil {
		state = &storyTaskState{pending: make(map[string]int)}
		e.states[storyID] = state
	}
	if index, ok := state.pending[key]; ok {
		state.queue[index].run = run
		return true
	}
	if e.total >= e.capacity {
		return false
	}
	state.pending[key] = len(state.queue)
	state.queue = append(state.queue, storyTask{key: key, run: run})
	e.total++
	if !state.running && !state.scheduled {
		state.scheduled = true
		e.ready <- storyID
	}
	return true
}

func (e *storyTaskExecutor) Close() {
	if e != nil {
		e.cancel()
	}
}

func (e *storyTaskExecutor) worker() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case storyID := <-e.ready:
			task, ok := e.take(storyID)
			if !ok {
				continue
			}
			ctx, cancel := context.WithTimeout(e.ctx, e.timeout)
			err := runStoryTask(ctx, task)
			cancel()
			if err != nil && e.ctx.Err() == nil {
				log.Printf("oneday: background task %s for story %s failed: %v", task.key, storyID, err)
			}
			e.finish(storyID)
		}
	}
}

func runStoryTask(ctx context.Context, task storyTask) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return task.run(ctx)
}

func (e *storyTaskExecutor) take(storyID string) (storyTask, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.states[storyID]
	if state == nil || state.running || len(state.queue) == 0 {
		return storyTask{}, false
	}
	state.scheduled = false
	task := state.queue[0]
	state.queue = state.queue[1:]
	delete(state.pending, task.key)
	for key, index := range state.pending {
		state.pending[key] = index - 1
	}
	state.running = true
	e.total--
	return task, true
}

func (e *storyTaskExecutor) finish(storyID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.states[storyID]
	if state == nil {
		return
	}
	state.running = false
	if len(state.queue) == 0 {
		delete(e.states, storyID)
		return
	}
	if !state.scheduled {
		state.scheduled = true
		e.ready <- storyID
	}
}

var ragTasks = newStoryTaskExecutor(
	context.Background(), ragBackgroundWorkers, ragBackgroundCapacity, ragBackgroundTimeout,
)

func submitRAGTask(storyID, key string, run func(context.Context) error) {
	if !ragTasks.Submit(storyID, key, run) {
		log.Printf("oneday: background RAG queue full; skipped %s for story %s", key, storyID)
	}
}

func ragTaskKey(kind, payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%s:%x", kind, sum[:8])
}
