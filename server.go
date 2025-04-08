package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"net/http"
)

type apiConfig struct {
	password string
	sessions *map[string]time.Time
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/favicon.ico")
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/home.html")
}

func (a *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	tokenJSON := struct {
		Token string `json:"token"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&tokenJSON)
	if err != nil {
		log.Printf("error decoding request body: %v", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if tokenJSON.Token != a.password {
		log.Printf("invalid login with token %s", tokenJSON.Token)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	log.Printf("Successful login from %s with %s", r.RemoteAddr, tokenJSON.Token)

	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(time.Hour)
	log.Printf("Starting session %s", sessionID)

	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		// Domain:   ".app.localhost",
	}
	http.SetCookie(w, sessionCookie)
	(*a.sessions)[sessionID] = expiresAt
	w.WriteHeader(http.StatusOK)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "./static/login.html")
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
