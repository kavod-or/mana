package menu

import (
	"strings"
	"testing"
)

const validMenu = `
conference:
  name: {de: Konferenz, en: Conference}
  location: {de: Foyer, en: Foyer}
tags:
  vegan: {de: Vegan, en: Vegan}
permanent:
  drinks:
    - id: water
      name: {de: Wasser, en: Water}
  snacks: []
days:
  - date: "2026-10-12"
    services:
      - id: lunch
        title: {de: Mittagessen, en: Lunch}
        subtitle: {de: Frisch, en: Fresh}
        from: "12:30"
        until: "14:00"
        items: []
`

func TestDecodeValidMenu(t *testing.T) {
	config, err := Decode(strings.NewReader(validMenu))
	if err != nil {
		t.Fatalf("Decode returned an error: %v", err)
	}
	if got := config.Days[0].Services[0].Title.EN; got != "Lunch" {
		t.Fatalf("English title = %q, want Lunch", got)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := Decode(strings.NewReader(validMenu + "unexpected: true\n"))
	if err == nil {
		t.Fatal("Decode accepted an unknown field")
	}
}

func TestValidateRejectsIncompleteTranslation(t *testing.T) {
	config, err := Decode(strings.NewReader(strings.Replace(validMenu, "{de: Konferenz, en: Conference}", "{de: Konferenz}", 1)))
	if err == nil {
		t.Fatalf("Decode accepted incomplete translation: %#v", config)
	}
}

func TestValidateRejectsIncompleteTagTranslation(t *testing.T) {
	config, err := Decode(strings.NewReader(strings.Replace(validMenu, "{de: Vegan, en: Vegan}", "{de: Vegan}", 1)))
	if err == nil {
		t.Fatalf("Decode accepted incomplete tag translation: %#v", config)
	}
}

func TestValidateRejectsIncompleteOptionalTranslation(t *testing.T) {
	menuWithDescription := strings.Replace(validMenu, "        items: []", "        items:\n          - id: soup\n            name: {de: Suppe, en: Soup}\n            description: {de: Warm}", 1)
	config, err := Decode(strings.NewReader(menuWithDescription))
	if err == nil {
		t.Fatalf("Decode accepted incomplete description translation: %#v", config)
	}
}

func TestStoreKeepsLastValidMenu(t *testing.T) {
	loads := 0
	store, err := newStore(func() (Config, error) {
		loads++
		if loads == 1 {
			return Decode(strings.NewReader(validMenu))
		}
		return Decode(strings.NewReader("invalid: ["))
	}, 0)
	if err != nil {
		t.Fatalf("NewStore returned an error: %v", err)
	}

	config, reloadErr := store.Current()
	if reloadErr == nil {
		t.Fatal("Current did not report the invalid reload")
	}
	if config.Conference.Name.EN != "Conference" {
		t.Fatalf("Current did not retain the last valid menu: %#v", config)
	}
}
