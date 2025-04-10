package main

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"os"
	"time"

	"github.com/ChernakovEgor/vlhl/internal/database"

	"net/http"

	"github.com/google/uuid"
)

type apiConfig struct {
	baseURL         string
	password        string
	sessions        *map[string]time.Time
	sessionDuration time.Duration
	db              *database.Queries
}

func NewApiConfig(baseURL, password string, sessions *map[string]time.Time, db *database.Queries) *apiConfig {
	cfg := apiConfig{
		baseURL:         baseURL,
		password:        password,
		sessions:        sessions,
		sessionDuration: time.Hour,
		db:              db,
	}

	go func() {
		ticker := time.NewTicker(time.Minute * 10)
		for t := range ticker.C {
			log.Println("Updating stored sessions")
			for k, v := range *cfg.sessions {
				if t.After(v) {
					delete(*cfg.sessions, k)
					log.Printf("Session %s deleted", k)
				}
			}
		}
	}()

	return &cfg
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/favicon.ico")
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/home.html")
}

func (a *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	passStruct := struct {
		Password string `json:"password"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&passStruct)
	if err != nil {
		log.Printf("error decoding request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	password := passStruct.Password
	if password != a.password {
		log.Printf("invalid login with password %s", password)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Printf("Successful login from %s with %s", r.RemoteAddr, password)
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(a.sessionDuration)

	log.Printf("Starting session %s", sessionID)
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
	(*a.sessions)[sessionID] = expiresAt

	res, err := a.db.Ping(context.Background())
	if err != nil {
		log.Printf("error quering db: %v", err)
	}
	log.Printf("Ping from db: %v", res)

	w.WriteHeader(http.StatusOK)
}

func (a *apiConfig) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	a.serveTemplatedHTML(w, "./static/login.html")
}

func (a *apiConfig) sessionMiddlewareHandler(h http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("No cookie found.")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		log.Printf("Found cookie: %s", sessionCookie.Value)
		if _, ok := (*a.sessions)[sessionCookie.Value]; !ok {
			log.Printf("No cookie in sessions.")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h.ServeHTTP(w, r)
	}
}

func (a *apiConfig) sessionMiddleware(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("No cookie found.")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		log.Printf("Found cookie: %s", sessionCookie.Value)
		if _, ok := (*a.sessions)[sessionCookie.Value]; !ok {
			log.Printf("No cookie in sessions.")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}

func handleUpload(w http.ResponseWriter, req *http.Request) {
	req.ParseMultipartForm(1000 << 20)
	file, handler, err := req.FormFile("videoFile")
	if err != nil {
		log.Println("error retrieving file:", err)
		w.WriteHeader(404)
		return
	}
	defer file.Close()

	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)

	localFile, err := os.Create("uploadedVideo.mp4")
	if err != nil {
		log.Println("error creating local file:", err)
		return
	}
	defer localFile.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("error reading uploaded file:", err)
		return
	}

	_, err = localFile.Write(fileBytes)
	if err != nil {
		log.Println("error writing to local file:", err)
		return
	}

	log.Println("File uploaded")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("File uploaded!"))
}

func (a *apiConfig) serveTemplatedHTML(w http.ResponseWriter, htmlPath string) {
	t := template.New(htmlPath)
	fileBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Printf("error opening %s: %v", htmlPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	parsed, err := t.Parse(string(fileBytes))
	if err != nil {
		log.Printf("error parsing %s template: %v", htmlPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = parsed.Execute(w, a.baseURL)
	if err != nil {
		log.Printf("error executing %s template: %v", htmlPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
