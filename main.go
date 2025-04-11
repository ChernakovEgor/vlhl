package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ChernakovEgor/vlhl/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("FATAL loading .env file: %v", err)
	}

	conn, err := sql.Open("sqlite3", "file:./sql/vlhl.db")
	if err != nil {
		log.Fatalf("FATAL connecting to database: %v", err)
	}
	db := database.New(conn)

	sessions := make(map[string]time.Time)
	baseURL := os.Getenv("BASE_URL")
	password := os.Getenv("PASSWORD")
	mediaPath := os.Getenv("MEDIA_PATH")
	server := NewServerConfig(baseURL, password, mediaPath, &sessions, db)

	port := os.Getenv("PORT")
	log.Printf("INFO Starting web server on port %s", port)
	log.Fatalln(http.ListenAndServe(":"+port, server))
}
