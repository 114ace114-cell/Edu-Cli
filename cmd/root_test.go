package cmd

import (
	"net"
	"testing"
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
