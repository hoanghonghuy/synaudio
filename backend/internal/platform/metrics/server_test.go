package metrics

import (
	"net/http"
	"testing"
)

func TestNewPrivateServerAcceptsOnlyExplicitPrivateAddresses(t *testing.T) {
	handler := http.NewServeMux()

	for _, addr := range []string{"127.0.0.1:9090", "10.0.0.8:9090", "192.168.1.8:9090", "[::1]:9090"} {
		server, err := NewPrivateServer(addr, handler)
		if err != nil {
			t.Fatalf("expected private addr %q to be accepted: %v", addr, err)
		}
		if server == nil || server.Addr != addr {
			t.Fatalf("expected server for %q", addr)
		}
	}

	for _, addr := range []string{":9090", "0.0.0.0:9090", "8.8.8.8:9090", "metrics.example.com:9090"} {
		if server, err := NewPrivateServer(addr, handler); err == nil || server != nil {
			t.Fatalf("expected public/wildcard addr %q to be rejected", addr)
		}
	}
}

func TestNewPrivateServerDisabledWhenAddressEmpty(t *testing.T) {
	server, err := NewPrivateServer("", http.NewServeMux())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server != nil {
		t.Fatal("expected metrics listener to remain disabled")
	}
}
