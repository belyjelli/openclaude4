package tools

import (
	"net/url"
	"strings"
	"testing"
)

func TestValidateFetchURL(t *testing.T) {
	tests := []struct {
		raw string
		ok  bool
	}{
		{"http://127.0.0.1/foo", false},
		{"https://10.0.0.1/", false},
		{"https://192.168.0.1/x", false},
		{"http://[::1]/", false},
		{"http://localhost/foo", false},
		{"https://user:pass@example.com/", false},
		{"ftp://example.com/", false},
		{"https://8.8.8.8/", true},
	}
	for _, tc := range tests {
		u, err := url.Parse(tc.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.raw, err)
		}
		err = ValidateFetchURL(u)
		if tc.ok && err != nil {
			t.Errorf("%q: want ok, got %v", tc.raw, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("%q: want error, got nil", tc.raw)
		}
	}
}

func TestHTMLToPlainText(t *testing.T) {
	const in = `<!DOCTYPE html><html><head><title>x</title><script>evil()</script></head><body>
<p>Hello <b>world</b></p><style>.a{}</style><p>Second</p></body></html>`
	got, err := htmlToPlainText(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "world") {
		t.Fatalf("missing text: %q", got)
	}
	if strings.Contains(got, "evil") {
		t.Fatalf("script leaked: %q", got)
	}
}

func TestExtractFetchText_JSON(t *testing.T) {
	raw := []byte(`{"a":1}`)
	s, err := extractFetchText(raw, "application/json; charset=utf-8")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(s) != `{"a":1}` {
		t.Fatalf("got %q", s)
	}
}
