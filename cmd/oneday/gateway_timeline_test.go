package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

func TestGatewayTimelineListsAndForksWithRevisionGuard(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "timeline.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Conn().Exec(`INSERT INTO stories(id,name) VALUES('story-1','Story')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-1")
	if err != nil {
		t.Fatal(err)
	}
	story, err := db.GetStory("story-1")
	if err != nil {
		t.Fatal(err)
	}
	request := contracts.BrowserTimelineRequest{StoryID: "story-1", Action: contracts.TimelineFork, ClientRevision: story.Revision, FromCommitID: head.Commit.ID, Name: "alternate"}
	input, _ := json.Marshal(request)
	var output bytes.Buffer
	if err := runGatewayTimeline(db, bytes.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	var response contracts.BrowserTimelineResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Branches) != 2 || response.ActiveBranchID != head.Branch.ID || response.Revision <= story.Revision {
		t.Fatalf("unexpected timeline response: %+v", response)
	}
	for _, branch := range response.Branches {
		if branch.HeadTurn != head.Commit.CanonicalTurn {
			t.Fatalf("branch %s head turn = %d, want %d", branch.Name, branch.HeadTurn, head.Commit.CanonicalTurn)
		}
	}
	if len(response.Commits) != 1 || response.Commits[0].ID != head.Commit.ID {
		t.Fatalf("unexpected active ancestry: %+v", response.Commits)
	}
}
