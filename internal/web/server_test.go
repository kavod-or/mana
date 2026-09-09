package web

import (
	"compress/gzip"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestPricesInRealTemplate(t *testing.T) {
	for _, configured := range []bool{false, true} {
		config := menu.Config{Days: []menu.Day{{Services: []menu.Service{{Items: []menu.Item{{}}}}}}, Permanent: menu.Permanent{Drinks: []menu.Item{{}}, Snacks: []menu.Item{{}}}}
		if configured {
			meal, item, zero := menu.Price(1250), menu.Price(450), menu.Price(0)
			config.Days[0].Services[0].Price = &meal
			config.Days[0].Services[0].Items[0].Price = &item
			config.Permanent.Drinks[0].Price = &zero
			config.Permanent.Snacks[0].Price = &item
		}
		handler, err := New(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		body := response.Body.String()
		if !configured {
			if strings.Contains(body, `class="price"`) {
				t.Fatal("absent prices rendered")
			}
			continue
		}
		for text, count := range map[string]int{"12,50\u00a0€": 2, "€12.50": 2, "4,50\u00a0€": 3, "€4.50": 3, "0,00\u00a0€": 1, "€0.00": 1} {
			if got := strings.Count(body, text); got != count {
				t.Errorf("%q appeared %d times, want %d", text, got, count)
			}
		}
	}
}

func TestSizePricesRender(t *testing.T) {
	normal, large := menu.Price(0), menu.Price(420)
	config := menu.Config{Days: []menu.Day{{Services: []menu.Service{{PriceNormal: &normal, PriceLarge: &large}}}}, Permanent: menu.Permanent{Coffee: []menu.Item{{PriceNormal: &normal}, {PriceLarge: &large}}}}
	handler, err := New(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, value := range []string{">Normal<", ">Regular<", ">Groß<", ">Large<", "0,00\u00a0€", "€0.00", "4,20\u00a0€", "€4.20"} {
		if count := strings.Count(body, value); count != 3 {
			t.Errorf("%q count = %d, want 3", value, count)
		}
	}
}

func TestFoodTrucksInExampleMenu(t *testing.T) {
	file, err := os.Open("../../content/menu.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := menu.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	if strings.Count(body, `class="food-trucks"`) != 2 || strings.Count(body, `class="food-truck-card"`) != 3 {
		t.Fatal("unexpected truck sections or cards")
	}
	if strings.Contains(body, `id="food-trucks-title-2"`) {
		t.Fatal("empty day has truck section")
	}
	for _, truck := range config.Days[0].FoodTrucks {
		for _, text := range []string{truck.Name.DE, truck.Name.EN, truck.Description.DE, truck.Description.EN, truck.Location.DE, truck.Location.EN, truck.From, truck.Until} {
			if !strings.Contains(body, text) {
				t.Errorf("missing %q", text)
			}
		}
	}
}

func TestSoldOutRendering(t *testing.T) {
	price := menu.Price(250)
	config := menu.Config{Days: []menu.Day{{Services: []menu.Service{{SoldOut: true, Price: &price, Items: []menu.Item{{SoldOut: true, Price: &price}}}}}}, Permanent: menu.Permanent{Coffee: []menu.Item{{SoldOut: true, PriceNormal: &price}}, Drinks: []menu.Item{{SoldOut: true}}, Snacks: []menu.Item{{SoldOut: true}}}}
	handler, err := New(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, label := range []string{"Ausverkauft", "Sold out"} {
		if count := strings.Count(body, label); count != 7 {
			t.Errorf("%s count = %d, want 7", label, count)
		}
	}
	if strings.Contains(body, "€2.50") {
		t.Fatal("sold out price remains visible")
	}
}

func TestEmptyRefreshmentsHidden(t *testing.T) {
	config := menu.Config{Days: []menu.Day{{Services: []menu.Service{{}}}}}
	handler, err := New(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if strings.Contains(response.Body.String(), `id="refreshments-`) || strings.Contains(response.Body.String(), `class="always-group"`) {
		t.Fatal("empty refreshments rendered")
	}
}
