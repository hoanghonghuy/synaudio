package httpserver_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/platform/httpserver"
)

func TestApplyPublicAPIPolicySetsExplicitBounds(t *testing.T) {
	srv := &http.Server{Addr: ":8080"}
	httpserver.ApplyPublicAPIPolicy(srv)

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout: got %v", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 60*time.Second {
		t.Fatalf("ReadTimeout: got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 11*time.Minute {
		t.Fatalf("WriteTimeout: got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 120*time.Second {
		t.Fatalf("IdleTimeout: got %v", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != 64*1024 {
		t.Fatalf("MaxHeaderBytes: got %d", srv.MaxHeaderBytes)
	}
}

func TestApplyMetricsPolicySetsScrapeAppropriateBounds(t *testing.T) {
	srv := &http.Server{Addr: "127.0.0.1:9090"}
	httpserver.ApplyMetricsPolicy(srv)

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout: got %v", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout: got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 30*time.Second {
		t.Fatalf("WriteTimeout: got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout: got %v", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != 32*1024 {
		t.Fatalf("MaxHeaderBytes: got %d", srv.MaxHeaderBytes)
	}
}

func TestPublicMaxRequestBodyBytesIsDocumentedAndFinite(t *testing.T) {
	if httpserver.PublicMaxRequestBodyBytes <= 0 {
		t.Fatal("expected positive request body limit")
	}
	if httpserver.PublicMaxRequestBodyBytes > 8<<20 {
		t.Fatalf("request body limit %d looks unreasonably large", httpserver.PublicMaxRequestBodyBytes)
	}
}
