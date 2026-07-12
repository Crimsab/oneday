package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoryTaskExecutorBoundsAndSerializesWork(t *testing.T) {
	executor := newStoryTaskExecutor(context.Background(), 2, 16, time.Second)
	defer executor.Close()
	release := make(chan struct{})
	started := make(chan struct{}, 4)
	var active atomic.Int32
	var maximum atomic.Int32
	var sameStoryActive atomic.Int32
	var overlap atomic.Bool
	for index, storyID := range []string{"story-a", "story-a", "story-b", "story-c"} {
		storyID := storyID
		if !executor.Submit(storyID, fmt.Sprintf("task-%d", index), func(ctx context.Context) error {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			if storyID == "story-a" && sameStoryActive.Add(1) > 1 {
				overlap.Store(true)
			}
			started <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
			}
			if storyID == "story-a" {
				sameStoryActive.Add(-1)
			}
			active.Add(-1)
			return nil
		}) {
			t.Fatal("task submission failed")
		}
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start")
		}
	}
	if maximum.Load() > 2 {
		t.Fatalf("maximum concurrent tasks = %d, want <= 2", maximum.Load())
	}
	close(release)
	time.Sleep(20 * time.Millisecond)
	if overlap.Load() {
		t.Fatal("tasks for one story overlapped")
	}
}

func TestStoryTaskExecutorCoalescesPendingKey(t *testing.T) {
	executor := newStoryTaskExecutor(context.Background(), 1, 8, time.Second)
	defer executor.Close()
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	done := make(chan int, 2)
	var once sync.Once
	if !executor.Submit("story", "summary", func(context.Context) error {
		once.Do(func() { close(firstStarted) })
		<-releaseFirst
		done <- 1
		return nil
	}) {
		t.Fatal("first submission failed")
	}
	<-firstStarted
	for value := 2; value <= 3; value++ {
		value := value
		if !executor.Submit("story", "summary", func(context.Context) error {
			done <- value
			return nil
		}) {
			t.Fatal("coalesced submission failed")
		}
	}
	close(releaseFirst)
	if got := <-done; got != 1 {
		t.Fatalf("first result = %d", got)
	}
	select {
	case got := <-done:
		if got != 3 {
			t.Fatalf("coalesced result = %d, want newest 3", got)
		}
	case <-time.After(time.Second):
		t.Fatal("coalesced task did not run")
	}
}
