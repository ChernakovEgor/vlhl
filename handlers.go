package main

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ChernakovEgor/vlhl/internal/database"
	"github.com/google/uuid"
)

type serverConfig struct {
	baseURL         string
	password        string
	mediaPath       string
	sessions        *map[string]time.Time
	sessionDuration time.Duration
	db              *database.Queries
	mux             *http.ServeMux
}

func NewServerConfig(baseURL, password, mediaPath string, sessions *map[string]time.Time, db *database.Queries) *serverConfig {
	cfg := serverConfig{
		baseURL:         baseURL,
		password:        password,
		mediaPath:       mediaPath,
		sessions:        sessions,
		sessionDuration: time.Hour,
		db:              db,
	}

	mux := http.NewServeMux()
	// middleware endpoints
	mux.HandleFunc("POST /api/v1/upload", cfg.sessionMiddleware(cfg.handleFileUpload))
	mux.HandleFunc("GET /static/", cfg.sessionMiddlewareHandler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))),
	)
	mux.HandleFunc("GET /home", cfg.sessionMiddleware(cfg.handleHome))
	mux.HandleFunc("GET /upload", cfg.sessionMiddleware(cfg.handleUpload))

	// public endpoints
	mux.HandleFunc("POST /api/v1/login", cfg.handleLogin)
	mux.HandleFunc("GET /favicon.ico", handleFavicon)
	mux.HandleFunc("GET /", cfg.handleRoot)

	cfg.mux = mux

	// session cleaner
	go func() {
		ticker := time.NewTicker(time.Minute * 10)
		for t := range ticker.C {
			slog.Info("Updating stored sessions")
			for k, v := range *cfg.sessions {
				if t.After(v) {
					delete(*cfg.sessions, k)
					slog.Info("Session deleted", "session_id", k)
				}
			}
		}
	}()

	return &cfg
}

func (s *serverConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("Handling request", "method", r.Method, "url", r.URL.String(), "addr", r.RemoteAddr)
	s.mux.ServeHTTP(w, r)
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Serving favicon")
	http.ServeFile(w, r, "./public/favicon.ico")
}

func (s *serverConfig) handleHome(w http.ResponseWriter, r *http.Request) {
	s.serveTemplatedHTML(w, "./static/home.html")
}

func (s *serverConfig) handleUpload(w http.ResponseWriter, r *http.Request) {
	s.serveTemplatedHTML(w, "./static/upload.html")
}

func (s *serverConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	passStruct := struct {
		Password string `json:"password"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&passStruct)
	if err != nil {
		slog.Error("Decoding request body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	password := passStruct.Password
	if password != s.password {
		slog.Info("Invalid login", "addr", r.RemoteAddr, "pass", password)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	slog.Info("Successful login", "addr", r.RemoteAddr, "pass", password)
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(s.sessionDuration)

	slog.Info("Starting session", "session_id", sessionID)
	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		// SameSite: http.SameSiteLaxMode,
		Path: "/",
		// Domain:   ".app.localhost",
	}
	http.SetCookie(w, sessionCookie)
	(*s.sessions)[sessionID] = expiresAt

	res, err := s.db.Ping(context.Background())
	if err != nil {
		slog.Error("Quering db", "error", err)
	}
	slog.Info("Ping from db", "result", res)

	w.WriteHeader(http.StatusOK)
}

func (s *serverConfig) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.serveTemplatedHTML(w, "./public/login.html")
}

func (s *serverConfig) sessionMiddlewareHandler(h http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			slog.Info("Cookie not found", "addr", r.RemoteAddr)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		slog.Info("Found cookie", "session_cookie", sessionCookie.Value)
		if _, ok := (*s.sessions)[sessionCookie.Value]; !ok {
			slog.Info("Session_id not found in active sessions", "session_id", sessionCookie.Value)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h.ServeHTTP(w, r)
	}
}

func (s *serverConfig) sessionMiddleware(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			slog.Info("Cookie not found", "addr", r.RemoteAddr)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		slog.Info("Found cookie", "session_cookie", sessionCookie.Value)
		if _, ok := (*s.sessions)[sessionCookie.Value]; !ok {
			slog.Info("Session_id not found in active sessions", "session_id", sessionCookie.Value)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}

func (s *serverConfig) handleFileUpload(w http.ResponseWriter, req *http.Request) {
	req.ParseMultipartForm(1000 << 20)
	file, handler, err := req.FormFile("videoFile")
	if err != nil {
		slog.Error("Retrieving file", "error", err)
		w.WriteHeader(404)
		return
	}
	defer file.Close()

	slog.Debug("Uploaded File", "filename", handler.Filename)
	slog.Debug("File Size", "size", handler.Size)
	slog.Debug("MIME Header", "mime_type", handler.Header)

	fileName := time.Now().Format("2006_02_01_15_04_05") + ".mp4"
	filePath := filepath.Join(s.mediaPath, fileName)
	localFile, err := os.Create(filePath)
	if err != nil {
		slog.Error("Creating local file", "error", err)
		return
	}
	defer localFile.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		slog.Error("Reading uploaded file", "error", err)
		return
	}

	_, err = localFile.Write(fileBytes)
	if err != nil {
		slog.Error("Writing to local file", "error", err)
		return
	}

	slog.Info("File uploaded", "path", filePath)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("File uploaded!"))
}

func (s *serverConfig) serveTemplatedHTML(w http.ResponseWriter, htmlPath string) {
	t := template.New(htmlPath)
	fileBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		slog.Error("serveTemplate: opening", "path", htmlPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	parsed, err := t.Parse(string(fileBytes))
	if err != nil {
		slog.Error("serveTemplate: parsing template", "path", htmlPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = parsed.Execute(w, s.baseURL)
	if err != nil {
		slog.Error("serveTemplate: executing template", "path", htmlPath, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
