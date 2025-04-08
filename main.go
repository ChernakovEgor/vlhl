package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading .env file: %v", err)
	}

	sessions := make(map[string]time.Time)
	cfg := apiConfig{
		password: os.Getenv("PASSWORD"),
		sessions: &sessions,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/login", cfg.handleLogin)
	mux.HandleFunc("POST /api/v1/upload", handleUpload)
	mux.HandleFunc("GET /static/", cfg.sessionMiddlewareHandler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))),
	)
	mux.HandleFunc("GET /favicon.ico", handleFavicon)
	mux.HandleFunc("GET /home", cfg.sessionMiddleware(handleHome))
	mux.HandleFunc("GET /", handleRoot)

	port := os.Getenv("PORT")
	log.Printf("Starting web server on port %s", port)
	log.Fatalln(http.ListenAndServe(":"+port, mux))
}
