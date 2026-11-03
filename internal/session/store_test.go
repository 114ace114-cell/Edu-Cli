package session

import (
	"testing"

	"github.com/kocoro/skeleton/internal/client"
)

func TestSaveAndLoad(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess, err := st.LoadOrCreate("", "new")
	if err != nil {
		t.Fatal(err)
	}
	sess.Messages = []client.Message{{Role: "user", Content: "hi"}}
	if err := st.Save(sess); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "hi" {
		t.Fatalf("%+v", got)
	}
	ids, err := st.List()
	if err != nil || len(ids) != 1 || ids[0] != sess.ID {
		t.Fatalf("%v %v", ids, err)
	}
}

func TestRejectsBadID(t *testing.T) {
	if ValidID("../etc") || ValidID("") == false {
		t.Fatal("id rules")
	}
	if ValidID("") != true {
		t.Fatal("empty id means mint")
	}
}
