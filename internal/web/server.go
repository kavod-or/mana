package web

import (
	"bytes"
	"compress/gzip"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mana/internal/menu"
	"mana/internal/version"
)

type server struct {
	events map[string]menu.Loader
	page   *template.Template
	static http.Handler
	logger *slog.Logger
}

func New(events map[string]menu.Loader, templates fs.FS, static fs.FS, logger *slog.Logger) (http.Handler, error) {
	functions := template.FuncMap{
		"version": func() string { return version.Current },
		"tag": func(tags map[string]menu.Localized, id, language string) string {
			value, ok := tags[id]
			if !ok {
				return id
			}
			if language == "en" {
				return value.EN
			}
			return value.DE
		},
	}

	page, err := template.New("index.html").Funcs(functions).ParseFS(templates, "web/templates/index.html")
	if err != nil {
		return nil, err
	}

	if _, err := page.New("landing").Parse(landingPage); err != nil {
		return nil, err
	}
	if _, err := page.New("not-found").Parse(notFoundPage); err != nil {
		return nil, err
	}
	s := &server{
		events: events,
		page:   page,
		static: http.StripPrefix("/static/", http.FileServer(http.FS(static))),
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /static/", s.asset)
	return securityHeaders(gzipResponses(accessLog(mux, logger))), nil
}

// Only shared presentation assets are public. Never expose a directory listing,
// source maps, or configuration files accidentally placed in the static folder.
func (s *server) asset(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/static/hearts.js", "/static/girly.css", "/static/manna.js", "/static/app.js", "/static/styles.css", "/static/logo.png", "/static/favicon.png":
		cacheStatic(s.static).ServeHTTP(writer, request)
	default:
		writer.Header().Set("Cache-Control", "no-store")
		http.NotFound(writer, request)
	}
}

func (s *server) index(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.URL.Path == "/" {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.page.ExecuteTemplate(writer, "landing", nil)
		return
	}
	loader, ok := s.events[request.URL.Path]
	if !ok {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusNotFound)
		_ = s.page.ExecuteTemplate(writer, "not-found", nil)
		return
	}
	config, err := loader()
	if err != nil {
		s.logger.Warn("menu reload failed; serving last valid version", "event", request.URL.Path, "error", err)
	}
	data := struct {
		menu.Config
		EventPath string
	}{config, request.URL.Path}
	var body bytes.Buffer
	if err := s.page.ExecuteTemplate(&body, "index.html", data); err != nil {
		s.logger.Error("render page", "error", err)
		http.Error(writer, "Could not render menu", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(body.Bytes())
}

func (s *server) health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; font-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(writer, request)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (writer gzipResponseWriter) WriteHeader(status int) {
	writer.Header().Del("Content-Length")
	writer.ResponseWriter.WriteHeader(status)
}

func (writer gzipResponseWriter) Write(content []byte) (int, error) {
	writer.Header().Del("Content-Length")
	return writer.writer.Write(content)
}

func gzipResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		acceptsGzip := strings.Contains(request.Header.Get("Accept-Encoding"), "gzip")
		compressible := !strings.HasPrefix(request.URL.Path, "/static/") || strings.HasSuffix(request.URL.Path, ".css") || strings.HasSuffix(request.URL.Path, ".js")
		if request.Method == http.MethodHead || !acceptsGzip || !compressible {
			next.ServeHTTP(writer, request)
			return
		}

		compressed, err := gzip.NewWriterLevel(writer, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(writer, request)
			return
		}
		defer compressed.Close()

		writer.Header().Set("Content-Encoding", "gzip")
		writer.Header().Add("Vary", "Accept-Encoding")
		next.ServeHTTP(gzipResponseWriter{ResponseWriter: writer, writer: compressed}, request)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *responseRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func accessLog(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		logger.Info("request", "method", request.Method, "path", request.URL.Path, "status", recorder.status, "duration", time.Since(started))
	})
}
