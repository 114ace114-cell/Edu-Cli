package permissions

import "testing"

func TestAdmitDeniesDestructiveShell(t *testing.T) {
	err := Admit(t.TempDir(), "bash", `{"command":"rm -rf ."}`, nil)
	if err == nil {
		t.Fatal("expected deny")
	}
}

func TestAdmitAllowsEcho(t *testing.T) {
	if err := Admit(t.TempDir(), "bash", `{"command":"echo hi"}`, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAdmitRejectsEscape(t *testing.T) {
	err := Admit(t.TempDir(), "file_read", `{"path":"../secret.txt"}`, nil)
	if err == nil {
		t.Fatal("expected deny")
	}
}
