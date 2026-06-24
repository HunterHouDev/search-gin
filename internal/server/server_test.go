package server

import (
	"net/http"
	"testing"
	"time"
)

func TestResolvePort_EmptyHost_ReturnsDefault(t *testing.T) {
	got := ResolvePort(":10081", "")
	if got != ":10081" {
		t.Errorf("ResolvePort('':10081', '') = %q, want %q", got, ":10081")
	}
}

func TestResolvePort_PortOnly_ReturnsSame(t *testing.T) {
	got := ResolvePort(":10081", ":10081")
	if got != ":10081" {
		t.Errorf("ResolvePort('':10081', '':10081') = %q, want %q", got, ":10081")
	}
}

func TestResolvePort_FullHost_ExtractsPort(t *testing.T) {
	got := ResolvePort(":10081", "127.0.0.1:10082")
	if got != ":10082" {
		t.Errorf("ResolvePort('':10081', '127.0.0.1:10082') = %q, want %q", got, ":10082")
	}
}

func TestResolvePort_NoColon_ReturnsDefault(t *testing.T) {
	got := ResolvePort(":10081", "localhost")
	if got != ":10081" {
		t.Errorf("ResolvePort('':10081', 'localhost') = %q, want %q", got, ":10081")
	}
}

func TestResolvePort_LocalhostWithPort_ExtractsPort(t *testing.T) {
	got := ResolvePort(":10081", "localhost:10082")
	if got != ":10082" {
		t.Errorf("ResolvePort('':10081', 'localhost:10082') = %q, want %q", got, ":10082")
	}
}

func TestCreateServer_SetsReadHeaderTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	srv := CreateServer(":10081", handler)

	if srv.Addr != ":10081" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":10081")
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 10*time.Second)
	}
	if srv.Handler == nil {
		t.Error("Handler should not be nil")
	}
}
