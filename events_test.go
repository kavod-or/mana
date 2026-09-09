package main

import (
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
