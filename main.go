package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

const TOKEN = "token"
const SECRET = "secret"

func main() {
	http.HandleFunc("POST /api/v1/login", handleLogin)
	http.HandleFunc("POST /api/v1/upload", handleUpload)
	http.HandleFunc("GET /static/", sessionMiddleware(http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))))
	// http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("GET /", handleLanding)

	log.Println("starting web server")
	log.Fatalln(http.ListenAndServe(":8080", http.DefaultServeMux))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
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

	if tokenJSON.Token != TOKEN {
		log.Printf("invalid login with token %s", tokenJSON.Token)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	log.Printf("Successful login from %s with %s", r.RemoteAddr, tokenJSON.Token)

	sessionID := uuid.New().String()
	log.Printf("Starting session %s", sessionID)

	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		// Domain:   ".app.localhost",
	}
	http.SetCookie(w, sessionCookie)
	w.WriteHeader(http.StatusOK)
}

func handleLanding(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	log.Println(r.Header)
	log.Println("Cookies:", r.Cookies())
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, "./static/login.html")
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

func sessionMiddleware(s http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("No cookie found.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("Found cookie: %s", sessionCookie.Value)

		s.ServeHTTP(w, r)
	}
}
