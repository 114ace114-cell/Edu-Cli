package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kocoro/skeleton/internal/client"
	"github.com/kocoro/skeleton/internal/daemon"
)

func TestEndpointAddrDefaultsHTTPSPort(t *testing.T) {
	addr, err := endpointAddr("https://api.deepseek.com")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "api.deepseek.com:443" {
		t.Fatal(addr)
	}
	local, err := endpointAddr("http://127.0.0.1:11434")
	if err != nil {
		t.Fatal(err)
	}
	if local != "127.0.0.1:11434" {
		t.Fatal(local)
	}
}

func TestEndpointReachableLocal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if !endpointReachable("http://" + ln.Addr().String()) {
		t.Fatal("listener was reported unreachable")
	}
}

func TestDemoEndToEnd(t *testing.T) {
	workspace := t.TempDir()
	data := t.TempDir()
	runner, err := buildRunner(workspace, data, client.DemoScript())
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.RunAgent(daemon.Request{Text: "change greet and write proof.txt"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reply != "updated greet and wrote proof.txt" {
		t.Fatal(result.Reply)
	}
	src, err := os.ReadFile(filepath.Join(workspace, "hello.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `return "hello from skeleton"`) {
		t.Fatalf("source:\n%s", src)
	}
	proof, err := os.ReadFile(filepath.Join(workspace, "proof.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proof), "skeleton-ok") {
		t.Fatalf("proof %q", proof)
	}
}
