package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPingHTTP_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	got := pingHTTP(srv.URL)
	if !strings.Contains(got, "OK") {
		t.Fatalf("pingHTTP: %q", got)
	}
}

func TestPingHTTP_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	got := pingHTTP(srv.URL)
	if !strings.Contains(got, "500") {
		t.Fatalf("expected 500 in message, got %q", got)
	}
}
