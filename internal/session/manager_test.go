package session

import (
	"testing"

	"github.com/kocoro/edu-cli/internal/client"
)

func TestManagerNewResumeSave(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(st)
	sess, err := mgr.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if sess.Title != "New session" || sess.ID == "" {
		t.Fatalf("%+v", sess)
	}
	sess.Messages = []client.Message{{Role: "user", Content: "hello"}}
	sess.Title = "hello"
	if err := mgr.Save(); err != nil {
		t.Fatal(err)
	}
	id := sess.ID
	mgr2 := NewManager(st)
	if _, err := mgr2.NewSession(); err != nil {
		t.Fatal(err)
	}
	got, err := mgr2.Resume(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 || got.Title != "hello" {
		t.Fatalf("%+v", got)
	}
}

func TestManagerResumeByListIndex(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(st)
	for _, title := range []string{"first", "second"} {
		s, _ := mgr.NewSession()
		s.Title = title
		s.Messages = []client.Message{{Role: "user", Content: title}}
		_ = mgr.Save()
		_ = s
	}
	got, err := mgr.ResumeByListIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "second" {
		t.Fatalf("expected newest first in list, got %q", got.Title)
	}
}
