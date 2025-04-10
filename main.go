package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ChernakovEgor/vl_hl/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading .env file: %v", err)
	}

	conn, err := sql.Open("sqlite3", "file:./sql/vlhl.db")
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	db := database.New(conn)

	sessions := make(map[string]time.Time)
	cfg := NewApiConfig(os.Getenv("BASE_URL"), os.Getenv("PASSWORD"), &sessions, db)
	mux := http.NewServeMux()

	// middleware endpoints
	mux.HandleFunc("POST /api/v1/upload", cfg.sessionMiddleware(handleUpload))
	mux.HandleFunc("GET /static/", cfg.sessionMiddlewareHandler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))),
	)
	mux.HandleFunc("GET /home", cfg.sessionMiddleware(handleHome))

	// public endpoints
	mux.HandleFunc("POST /api/v1/login", cfg.handleLogin)
	mux.HandleFunc("GET /favicon.ico", handleFavicon)
	mux.HandleFunc("GET /", cfg.handleRoot)

	port := os.Getenv("PORT")
	log.Printf("Starting web server on port %s", port)
	log.Fatalln(http.ListenAndServe(":"+port, mux))
}
