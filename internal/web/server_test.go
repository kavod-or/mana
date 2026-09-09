package web

import (
	"compress/gzip"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"mana/internal/menu"
)

func TestServerRendersMenuAndSecurityHeaders(t *testing.T) {
	templateFS := fstest.MapFS{
		"web/templates/index.html": {Data: []byte(`{{define "index.html"}}<h1>{{.Conference.Name.DE}}</h1>{{end}}`)},
	}
	staticFS := fstest.MapFS{"styles.css": {Data: []byte("body{}")}}
	config := menu.Config{
		Conference: menu.Conference{Name: menu.Localized{DE: "Mana Konferenz", EN: "Mana Conference"}, Location: menu.Localized{DE: "Foyer", EN: "Foyer"}},
		Days:       []menu.Day{{Date: "2026-10-12", Services: []menu.Service{{ID: "lunch", Title: menu.Localized{DE: "Mittagessen", EN: "Lunch"}, From: "12:00", Until: "13:00"}}}},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler, err := New(func() (menu.Config, error) { return config, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Mana Konferenz") {
		t.Fatalf("response does not contain conference name: %s", response.Body.String())
	}
	if got := response.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("unexpected CSP: %q", got)
	}
}

func TestHealthEndpoint(t *testing.T) {
	templateFS := fstest.MapFS{
		"web/templates/index.html": {Data: []byte(`{{define "index.html"}}ok{{end}}`)},
	}
	staticFS := fstest.MapFS{"styles.css": {Data: []byte("body{}")}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := New(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("unexpected health response: %d %s", response.Code, response.Body.String())
	}
}

func TestCompressesHTMLWhenAccepted(t *testing.T) {
	templateFS := fstest.MapFS{
		"web/templates/index.html": {Data: []byte(`{{define "index.html"}}<h1>Mana Conference</h1>{{end}}`)},
	}
	staticFS := fstest.MapFS{"styles.css": {Data: []byte("body{}")}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := New(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", response.Header().Get("Content-Encoding"))
	}
	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("open gzip response: %v", err)
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip response: %v", err)
	}
	if !strings.Contains(string(body), "Mana Conference") {
		t.Fatalf("unexpected uncompressed body: %s", body)
	}
}

func TestStaticAssetsUseLongLivedCache(t *testing.T) {
	templateFS := fstest.MapFS{
		"web/templates/index.html": {Data: []byte(`{{define "index.html"}}ok{{end}}`)},
	}
	staticFS := fstest.MapFS{"styles.css": {Data: []byte("body{}")}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := New(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/static/styles.css?v=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

var _ fs.FS = fstest.MapFS{}
