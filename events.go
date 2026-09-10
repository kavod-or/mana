package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"mana/internal/menu"
)

type eventEntry struct {
	Path string `yaml:"path"`
	Menu string `yaml:"menu"`
}

var eventPathPattern = regexp.MustCompile(`^/[a-z0-9]+(?:-[a-z0-9]+)*$`)

type rootedFS struct {
	root *os.Root
}

func (root *rootedFS) Open(name string) (fs.File, error) {
	return root.root.Open(name)
}

func (root *rootedFS) Stat(name string) (fs.FileInfo, error) {
	return root.root.Stat(name)
}

func (root *rootedFS) Close() error {
	return root.root.Close()
}

func loadEvents() (map[string]menu.Loader, error) {
	content, err := loadContentFS()
	if err != nil {
		return nil, err
	}
	return loadEventFS(content)
}

func loadContentFS() (fs.FS, error) {
	if dir := os.Getenv("CONTENT_DIR"); dir != "" {
		// os.Root prevents relative paths and symlinks from escaping CONTENT_DIR.
		root, err := os.OpenRoot(dir)
		if err != nil {
			return nil, err
		}
		return &rootedFS{root: root}, nil
	}
	return fs.Sub(assets, "content")
}

func loadEventFS(content fs.FS) (map[string]menu.Loader, error) {
	file, err := content.Open("events.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var manifest struct {
		Events []eventEntry `yaml:"events"`
	}
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("events.yaml must contain exactly one YAML document")
	}
	events := make(map[string]menu.Loader)
	for _, entry := range manifest.Events {
		if !eventPathPattern.MatchString(entry.Path) || entry.Path == "/healthz" || entry.Path == "/static" {
			return nil, fmt.Errorf("invalid or reserved event path %q", entry.Path)
		}
		if _, exists := events[entry.Path]; exists {
			return nil, fmt.Errorf("duplicate event path %q", entry.Path)
		}
		if !fs.ValidPath(entry.Menu) || strings.Contains(entry.Menu, `\`) {
			return nil, fmt.Errorf("invalid menu path %q", entry.Menu)
		}
		store, err := menu.NewStore(func() (menu.Config, error) {
			file, err := content.Open(entry.Menu)
			if err != nil {
				return menu.Config{}, err
			}
			defer file.Close()
			return menu.Decode(file)
		})
		if err != nil {
			return nil, fmt.Errorf("event %s: %w", entry.Path, err)
		}
		events[entry.Path] = store.Current
	}
	return events, nil
}
