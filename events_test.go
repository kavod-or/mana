package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestEmbeddedEvents(t *testing.T) {
	events, err := loadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("event count %d", len(events))
	}
	first, err := events["/example-conference"]()
	if err != nil {
		t.Fatal(err)
	}
	second, err := events["/community-day"]()
	if err != nil {
		t.Fatal(err)
	}
	if first.Conference.Name.EN == second.Conference.Name.EN {
		t.Fatal("events share menu")
	}
}

func TestRejectInvalidEventPaths(t *testing.T) {
	for _, path := range []string{"/", "/healthz", "/static", "/../secret", "missing-slash"} {
		_, err := loadEventFS(fstest.MapFS{"events.yaml": {Data: []byte("events:\n  - path: " + path + "\n    menu: menu.yaml\n")}})
		if err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
}

func TestRejectInvalidMenuPaths(t *testing.T) {
	for _, menuPath := range []string{"../secret.yaml", "/etc/passwd", `..\secret.yaml`, "./menu.yaml"} {
		manifest := "events:\n  - path: /test\n    menu: '" + menuPath + "'\n"
		_, err := loadEventFS(fstest.MapFS{"events.yaml": {Data: []byte(manifest)}})
		if err == nil {
			t.Fatalf("accepted menu path %q", menuPath)
		}
	}
}

func TestContentRootRejectsSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	contentDir := filepath.Join(parent, "content")
	if err := os.Mkdir(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "events.yaml"), []byte("events:\n  - path: /test\n    menu: escape.yaml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret.yaml"), []byte("secret: server-data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../secret.yaml", filepath.Join(contentDir, "escape.yaml")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONTENT_DIR", contentDir)
	content, err := loadContentFS()
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := content.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	if _, err := fs.ReadFile(content, "escape.yaml"); err == nil {
		t.Fatal("content filesystem followed a symlink outside its root")
	}
	if _, err := loadEventFS(content); err == nil {
		t.Fatal("event loader accepted a menu symlink outside the content root")
	}
}
