package web

import (
	"compress/gzip"
	"html"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

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

	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
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
	handler, err := newTestServer(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
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
	handler, err := newTestServer(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
	if err != nil {
		t.Fatalf("New returned an error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
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
	handler, err := newTestServer(func() (menu.Config, error) { return menu.Config{}, nil }, templateFS, staticFS, logger)
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
		handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
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
	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
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
	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
	body := response.Body.String()
	if strings.Count(body, `class="food-trucks"`) != 2 || strings.Count(body, `class="food-truck-card"`) != 3 {
		t.Fatal("unexpected truck sections or cards")
	}
	if strings.Contains(body, `id="food-trucks-title-2"`) {
		t.Fatal("empty day has truck section")
	}
	for _, truck := range config.Days[0].FoodTrucks {
		for _, text := range []string{truck.Name.DE, truck.Name.EN, truck.Description.DE, truck.Description.EN, truck.Location.DE, truck.Location.EN, truck.From, truck.Until, html.EscapeString(truck.Payment.DE), html.EscapeString(truck.Payment.EN)} {
			if !strings.Contains(body, text) {
				t.Errorf("missing %q", text)
			}
		}
	}
}

func TestSoldOutRendering(t *testing.T) {
	price := menu.Price(250)
	config := menu.Config{Days: []menu.Day{{Services: []menu.Service{{SoldOut: true, Price: &price, Items: []menu.Item{{SoldOut: true, Price: &price}}}}}}, Permanent: menu.Permanent{Coffee: []menu.Item{{SoldOut: true, PriceNormal: &price}}, Drinks: []menu.Item{{SoldOut: true}}, Snacks: []menu.Item{{SoldOut: true}}}}
	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
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
	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
	if strings.Contains(response.Body.String(), `id="refreshments-`) || strings.Contains(response.Body.String(), `class="always-group"`) {
		t.Fatal("empty refreshments rendered")
	}
}

func newTestServer(loader menu.Loader, templates fs.FS, static fs.FS, logger *slog.Logger) (http.Handler, error) {
	return New(map[string]menu.Loader{"/test": loader}, templates, static, logger)
}

func TestEventRouting(t *testing.T) {
	events := map[string]menu.Loader{}
	for _, name := range []string{"alpha", "beta"} {
		events["/"+name] = func() (menu.Config, error) {
			return menu.Config{Conference: menu.Conference{Name: menu.Localized{DE: name}}}, nil
		}
	}
	templates := fstest.MapFS{"web/templates/index.html": {Data: []byte(`{{.Conference.Name.DE}} {{.EventPath}}`)}}
	handler, err := New(events, templates, fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/alpha", "/beta", "/unknown", "/alpha/nested"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		switch path {
		case "/":
			if response.Code != 200 || !strings.Contains(response.Body.String(), "Please scan the QR code") || strings.Contains(response.Body.String(), "alpha") {
				t.Fatal("incorrect landing page")
			}
		case "/alpha", "/beta":
			if response.Code != 200 || response.Body.String() != path[1:]+" "+path {
				t.Fatalf("incorrect event: %s", response.Body.String())
			}
		default:
			if response.Code != 404 {
				t.Fatalf("unknown path status %d", response.Code)
			}
		}
	}
}

func TestNotFoundPage(t *testing.T) {
	handler, err := New(nil, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/missing-event", nil))
	if response.Code != 404 {
		t.Fatalf("status %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatal("expected HTML")
	}
	for _, text := range []string{"Hier ist noch nicht gedeckt.", "This page couldn’t be found.", "QR-Code"} {
		if !strings.Contains(response.Body.String(), text) {
			t.Errorf("missing %q", text)
		}
	}
}

func TestClientDoesNotExposeOtherEvents(t *testing.T) {
	events := make(map[string]menu.Loader)
	for _, name := range []string{"private-alpha", "private-beta"} {
		events["/"+name] = func() (menu.Config, error) {
			return menu.Config{Conference: menu.Conference{Name: menu.Localized{DE: name, EN: name}}, Days: []menu.Day{{Services: []menu.Service{{Title: menu.Localized{DE: name + "-dish", EN: name + "-dish"}}}}}}, nil
		}
	}
	handler, err := New(events, os.DirFS("../.."), os.DirFS("../../web/static"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/missing", "/private-alpha", "/private-beta", "/static/app.js", "/static/manna.js", "/static/styles.css", "/static/logo.png", "/static/favicon.png"} {
		for _, encoding := range []string{"", "gzip"} {
			request := httptest.NewRequest("GET", path, nil)
			request.Header.Set("Accept-Encoding", encoding)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var reader io.Reader = response.Body
			if response.Header().Get("Content-Encoding") == "gzip" {
				compressed, err := gzip.NewReader(reader)
				if err != nil {
					t.Fatal(err)
				}
				defer compressed.Close()
				reader = compressed
			}
			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"private-alpha", "private-beta"} {
				if path != "/"+name && strings.Contains(string(body), name) {
					t.Errorf("%s exposes %s", path, name)
				}
			}
			for _, forbidden := range []string{"events.yaml", "community-day.yaml"} {
				if strings.Contains(string(body), forbidden) {
					t.Errorf("%s exposes %s", path, forbidden)
				}
			}
		}
	}
}

func TestConfigurationAndStaticListingsAreNotPublic(t *testing.T) {
	// Even accidental configuration files in the static directory stay private.
	static := fstest.MapFS{"events.yaml": {Data: []byte("secret-event")}, "app.js.map": {Data: []byte("secret-event")}}
	handler, err := New(nil, os.DirFS("../.."), static, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/events.yaml", "/content/events.yaml", "/content/menu.yaml", "/content/community-day.yaml", "/static/", "/static/events.yaml", "/static/app.js.map", "/static/../content/events.yaml"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code == 200 || strings.Contains(response.Body.String(), "secret-event") {
			t.Errorf("exposed %s", path)
		}
	}
}

func TestGzipNegotiation(t *testing.T) {
	for _, tc := range []struct {
		header     string
		compressed bool
	}{
		{"", false}, {"br", false}, {"gzip", true}, {"br, gzip;q=0.5", true},
		{"gzip;q=0", false}, {"*;q=1, gzip;q=0", false}, {"*", true},
		{"xgzip", false}, {"gzip;q=invalid", false},
	} {
		t.Run(tc.header, func(t *testing.T) {
			handler := gzipResponses(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hello")) }))
			request := httptest.NewRequest("GET", "/test", nil)
			request.Header.Set("Accept-Encoding", tc.header)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if got := response.Header().Get("Content-Encoding") == "gzip"; got != tc.compressed {
				t.Errorf("compressed = %v", got)
			}
			if response.Header().Get("Vary") != "Accept-Encoding" {
				t.Error("missing Vary")
			}
		})
	}
}

func TestGzipPreservesRangeResponses(t *testing.T) {
	handler := gzipResponses(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "app.js", time.Time{}, strings.NewReader("hello"))
	}))
	request := httptest.NewRequest("GET", "/static/app.js", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("Range", "bytes=0-1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "he" || response.Header().Get("Content-Encoding") != "" {
		t.Fatalf("invalid range response: %d %q", response.Code, response.Body.String())
	}
}

func TestPaymentNoticeRendering(t *testing.T) {
	for _, payment := range []menu.Localized{{}, {DE: "Nur Barzahlung", EN: "Cash only"}, {DE: "Bar & Karte", EN: "Cash & card <accepted>"}} {
		config := menu.Config{Conference: menu.Conference{Payment: payment}}
		handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", "/test", nil))
		body := response.Body.String()
		if response.Code != http.StatusOK {
			t.Fatalf("status %d", response.Code)
		}
		if strings.Contains(body, `class="payment-notice"`) != (payment.DE != "") {
			t.Fatal("unexpected notice visibility")
		}
		if payment.DE != "" {
			for _, text := range []string{`<span data-lang-content="de">` + html.EscapeString(payment.DE) + `</span>`, `<span data-lang-content="en" hidden>` + html.EscapeString(payment.EN) + `</span>`} {
				if !strings.Contains(body, text) {
					t.Errorf("missing translated notice %q", text)
				}
			}
		}
	}
}

func TestFoodTruckItemRendering(t *testing.T) {
	zero, normal, large := menu.Price(0), menu.Price(750), menu.Price(1000)
	config := menu.Config{Days: []menu.Day{{FoodTrucks: []menu.FoodTruck{{Items: []menu.Item{
		{Name: menu.Localized{DE: "Pita", EN: "Pita"}, Price: &normal, Description: menu.Localized{DE: "Mit Salat", EN: "With salad"}},
		{Name: menu.Localized{DE: "Wasser", EN: "Water"}, Price: &zero},
		{Name: menu.Localized{DE: "Pizza", EN: "Pizza"}, PriceNormal: &normal, PriceLarge: &large},
		{Name: menu.Localized{DE: "Tacos", EN: "Tacos"}, Price: &large, SoldOut: true},
		{Name: menu.Localized{DE: "Tagesgericht", EN: "Daily special"}},
	}}, {Name: menu.Localized{DE: "Leer", EN: "Empty"}}}}}}
	handler, err := newTestServer(func() (menu.Config, error) { return config, nil }, os.DirFS("../.."), fstest.MapFS{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/test", nil))
	body := response.Body.String()
	if response.Code != 200 {
		t.Fatalf("status %d", response.Code)
	}
	for text, count := range map[string]int{`class="food-truck-items"`: 1, "7,50\u00a0€": 2, "€7.50": 2, "€0.00": 1, "€10.00": 1, "Sold out": 1, "With salad": 1, "Daily special": 1} {
		if got := strings.Count(body, text); got != count {
			t.Errorf("%q count = %d, want %d", text, got, count)
		}
	}
}
