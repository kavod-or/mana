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
  vegetarian: {de: Vegetarisch, en: Vegetarian}
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

func TestConferenceLogo(t *testing.T) {
	for _, filename := range []string{"brand.png", "logos/brand.JPEG", "brand.webp", "brand.gif", "brand.avif"} {
		source := strings.Replace(validMenu, "conference:", "conference:\n  logo: "+filename, 1)
		config, err := Decode(strings.NewReader(source))
		if err != nil {
			t.Fatalf("accepted logo %q returned an error: %v", filename, err)
		}
		if config.Conference.Logo != filename {
			t.Fatalf("logo = %q, want %q", config.Conference.Logo, filename)
		}
	}
	for _, filename := range []string{"../brand.png", "/brand.png", "brand.svg", "brand.txt"} {
		source := strings.Replace(validMenu, "conference:", "conference:\n  logo: "+filename, 1)
		if _, err := Decode(strings.NewReader(source)); err == nil {
			t.Fatalf("accepted unsafe logo path %q", filename)
		}
	}
}

func TestValidateRejectsIncompleteTranslation(t *testing.T) {
	config, err := Decode(strings.NewReader(strings.Replace(validMenu, "{de: Konferenz, en: Conference}", "{de: Konferenz}", 1)))
	if err == nil {
		t.Fatalf("Decode accepted incomplete translation: %#v", config)
	}
}

func TestValidateRejectsIncompleteTagTranslation(t *testing.T) {
	config, err := Decode(strings.NewReader(strings.Replace(validMenu, "{de: Vegetarisch, en: Vegetarian}", "{de: Vegetarisch}", 1)))
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

func TestOptionalPrices(t *testing.T) {
	for _, value := range []string{"12.50", "0", "1234.5", "null"} {
		t.Run(value, func(t *testing.T) {
			source := strings.Replace(validMenu, "        items: []", "        items:\n          - id: soup\n            name: {de: Suppe, en: Soup}\n            price: "+value, 1)
			source = strings.Replace(source, "- id: water", "- id: water\n      price: "+value, 1)
			config, err := Decode(strings.NewReader(source))
			if err != nil {
				t.Fatal(err)
			}
			meal, item := config.Days[0].Services[0].Items[0].Price, config.Permanent.Drinks[0].Price
			if value == "null" {
				if meal != nil || item != nil {
					t.Fatal("null prices must be absent")
				}
				return
			}
			if meal == nil || item == nil || *meal != *item {
				t.Fatal("prices were not decoded")
			}
			want := map[string][2]string{"12.50": {"12,50\u00a0€", "€12.50"}, "0": {"0,00\u00a0€", "€0.00"}, "1234.5": {"1.234,50\u00a0€", "€1,234.50"}}[value]
			if meal.German() != want[0] || meal.English() != want[1] {
				t.Fatalf("unexpected formatting: %s / %s", meal.German(), meal.English())
			}
		})
	}
	for _, value := range []string{"-1", "1.234", "NaN", ".inf", "true", "12,50", "[12]", "{amount: 12}", "999999999999999999999"} {
		t.Run("invalid_"+value, func(t *testing.T) {
			source := strings.Replace(validMenu, "- id: water", "- id: water\n      price: "+value, 1)
			if _, err := Decode(strings.NewReader(source)); err == nil {
				t.Fatalf("accepted %s", value)
			}
		})
	}
}

func TestCoffeeItems(t *testing.T) {
	source := strings.Replace(validMenu, "  drinks:", "  coffee:\n    - id: espresso\n      price: 2.00\n      name: {de: Espresso, en: Espresso}\n  drinks:", 1)
	config, err := Decode(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Permanent.Coffee) != 1 || config.Permanent.Coffee[0].Price.English() != "€2.00" {
		t.Fatal("coffee not decoded")
	}
	if _, err := Decode(strings.NewReader(strings.Replace(source, "{de: Espresso, en: Espresso}", "{de: Espresso}", 1))); err == nil {
		t.Fatal("accepted missing coffee translation")
	}
}

func TestSizePrices(t *testing.T) {
	for _, fields := range []string{"price_normal: 0", "price_large: 4.20", "price_normal: 3.20\n      price_large: 4.20"} {
		source := strings.Replace(validMenu, "- id: water", "- id: water\n      "+fields, 1)
		config, err := Decode(strings.NewReader(source))
		if err != nil {
			t.Fatal(err)
		}
		item := config.Permanent.Drinks[0]
		if item.PriceNormal == nil && item.PriceLarge == nil {
			t.Fatal("size prices missing")
		}
	}
	for _, fields := range []string{"price_normal: -1", "price_large: 1.234", "price: 2\n      price_large: 3"} {
		source := strings.Replace(validMenu, "- id: water", "- id: water\n      "+fields, 1)
		if _, err := Decode(strings.NewReader(source)); err == nil {
			t.Fatalf("accepted %s", fields)
		}
	}
}

func TestFoodTrucks(t *testing.T) {
	truck := `
    food_trucks:
      - id: pita
        name: {de: Pita-Pause, en: Pita Stop}
        description: {de: Frische Pita, en: Fresh pita}
        location: {de: Innenhof, en: Courtyard}
        from: "11:30"
        until: "15:00"
`
	source := strings.Replace(validMenu, "    services:", truck+"    services:", 1)
	config, err := Decode(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Days[0].FoodTrucks) != 1 {
		t.Fatal("missing food truck")
	}
	for _, pair := range [][2]string{{"en: Pita Stop", ""}, {"en: Fresh pita", ""}, {"en: Courtyard", ""}, {"11:30", "25:00"}, {"15:00", "10:00"}, {"id: pita", "id: ''"}} {
		if _, err := Decode(strings.NewReader(strings.Replace(source, pair[0], pair[1], 1))); err == nil {
			t.Errorf("accepted invalid truck: %s", pair[0])
		}
	}
}

func TestSoldOut(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		source := strings.Replace(validMenu, "- id: water", "- id: water\n      sold_out: "+value, 1)
		source = strings.Replace(source, "- id: lunch", "- id: lunch\n        sold_out: "+value, 1)
		config, err := Decode(strings.NewReader(source))
		if err != nil {
			t.Fatal(err)
		}
		if config.Permanent.Drinks[0].SoldOut != (value == "true") || config.Days[0].Services[0].SoldOut != (value == "true") {
			t.Fatal("incorrect sold out value")
		}
	}
}

func TestDecodeRejectsAmbiguousMenu(t *testing.T) {
	for _, source := range []string{
		validMenu + "\n---\n" + validMenu,
		strings.Replace(validMenu, `until: "14:00"`, `until: "11:00"`, 1),
		strings.Replace(validMenu, "de: Konferenz", `de: " "`, 1),
	} {
		if _, err := Decode(strings.NewReader(source)); err == nil {
			t.Fatal("accepted invalid menu")
		}
	}
}

func TestTimeRequiresZeroPaddedHours(t *testing.T) {
	for _, value := range []string{"9:00", "09:0", "24:00", "12:30:00"} {
		if err := validateTime("from", value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	for _, value := range []string{"00:00", "09:00", "23:59"} {
		if err := validateTime("from", value); err != nil {
			t.Errorf("rejected %q: %v", value, err)
		}
	}
}

func TestStoreServesCachedMenuDuringReload(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	loads := 0
	store, err := newStore(func() (Config, error) {
		loads++
		if loads > 1 {
			close(started)
			<-release
		}
		return Config{Conference: Conference{Name: Localized{EN: "Conference"}}}, nil
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { store.Current(); close(done) }()
	<-started
	defer func() { close(release); <-done }()
	config, err := store.Current()
	if err != nil || config.Conference.Name.EN != "Conference" {
		t.Fatalf("cached reload: %v, %v", config, err)
	}
}

func TestPaymentNoticeValidation(t *testing.T) {
	for _, tc := range []struct {
		payment string
		valid   bool
	}{
		{"{de: Nur Barzahlung, en: Cash only}", true},
		{"{de: Kartenzahlung, en: Card payment}", true},
		{"{}", true}, {"null", true},
		{"{de: Nur Barzahlung}", false},
		{"{de: '   ', en: Cash only}", false},
	} {
		t.Run(tc.payment, func(t *testing.T) {
			source := strings.Replace(validMenu, "conference:", "conference:\n  payment: "+tc.payment, 1)
			_, err := Decode(strings.NewReader(source))
			if (err == nil) != tc.valid {
				t.Fatalf("valid = %v, error = %v", tc.valid, err)
			}
		})
	}
}

func TestFoodTruckPaymentValidation(t *testing.T) {
	for _, payment := range []Localized{{}, {DE: "Nur Barzahlung", EN: "Cash only"}, {EN: "Card payment"}, {DE: "   ", EN: "Cash only"}} {
		config, err := Decode(strings.NewReader(validMenu))
		if err != nil {
			t.Fatal(err)
		}
		config.Days[0].FoodTrucks = []FoodTruck{{ID: "truck", Name: Localized{DE: "Truck", EN: "Truck"}, Description: Localized{DE: "Essen", EN: "Food"}, Location: Localized{DE: "Hof", EN: "Yard"}, From: "12:00", Until: "14:00", Payment: payment}}
		err = config.Validate()
		valid := payment == (Localized{}) || payment.DE == "Nur Barzahlung"
		if (err == nil) != valid {
			t.Fatalf("payment %v: %v", payment, err)
		}
	}
}

func TestFoodTruckItemValidation(t *testing.T) {
	for _, tc := range []struct {
		item  Item
		valid bool
	}{
		{Item{ID: "pita", Name: Localized{DE: "Pita", EN: "Pita"}}, true},
		{Item{Name: Localized{DE: "Pita", EN: "Pita"}}, false},
		{Item{ID: "pita", Name: Localized{DE: "Pita"}}, false},
		{Item{ID: "pita", Name: Localized{DE: "Pita", EN: "Pita"}, Description: Localized{DE: "Salat"}}, false},
	} {
		config, err := Decode(strings.NewReader(validMenu))
		if err != nil {
			t.Fatal(err)
		}
		config.Days[0].FoodTrucks = []FoodTruck{{ID: "truck", Name: Localized{DE: "Truck", EN: "Truck"}, Description: Localized{DE: "Essen", EN: "Food"}, Location: Localized{DE: "Hof", EN: "Yard"}, From: "12:00", Until: "14:00", Items: []Item{tc.item}}}
		if err := config.Validate(); (err == nil) != tc.valid {
			t.Fatalf("item %v: %v", tc.item, err)
		}
	}
}
