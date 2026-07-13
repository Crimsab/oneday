package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

const (
	StoryMutationLockTTL       = 3 * time.Minute
	StoryMutationHeartbeatTick = 1 * time.Minute
)

// StoryMutationLease is a shared cross-process mutation lease for story state.
// Browser gateway calls, terminal saves/loads, autosaves, and normal turns all
// use the same SQLite-backed lock so they cannot commit conflicting snapshots.
type StoryMutationLease struct {
	lock      *storage.StoryTurnLock
	heartbeat *storage.StoryTurnLockHeartbeat
}

func AcquireStoryMutationLease(ctx context.Context, db *storage.DB, storyID, scope, ownerPrefix string) (*StoryMutationLease, error) {
	if db == nil {
		return nil, errors.New("database is not configured")
	}
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return nil, errors.New("story_id is required")
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "mutation"
	}
	ownerPrefix = strings.TrimSpace(ownerPrefix)
	if ownerPrefix == "" {
		ownerPrefix = "oneday"
	}
	owner := fmt.Sprintf("%s:%s:%d:%d", ownerPrefix, scope, os.Getpid(), time.Now().UnixNano())
	lock, err := db.AcquireStoryTurnLock(ctx, storyID, owner, StoryMutationLockTTL)
	if err != nil {
		return nil, err
	}
	return &StoryMutationLease{
		lock:      lock,
		heartbeat: lock.StartHeartbeat(ctx, StoryMutationHeartbeatTick, StoryMutationLockTTL),
	}, nil
}

func WithStoryMutationLease(ctx context.Context, db *storage.DB, storyID, scope, ownerPrefix string, fn func(*StoryMutationLease) error) error {
	lease, err := AcquireStoryMutationLease(ctx, db, storyID, scope, ownerPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = lease.Release() }()
	return fn(lease)
}

func (l *StoryMutationLease) Lock() *storage.StoryTurnLock {
	if l == nil {
		return nil
	}
	return l.lock
}

func (l *StoryMutationLease) Renew() error {
	if l == nil || l.lock == nil {
		return nil
	}
	return l.lock.Renew(StoryMutationLockTTL)
}

func (l *StoryMutationLease) Release() error {
	if l == nil {
		return nil
	}
	var firstErr error
	if l.heartbeat != nil {
		if err := l.heartbeat.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if l.lock != nil {
		if err := l.lock.Release(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
