package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TOKEN = "token"
const SECRET = "secret"

func main() {
	http.HandleFunc("POST /api/v1/login", handleLogin)
	http.HandleFunc("POST /api/v1/upload", handleUpload)
	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("GET /", handleLanding)

	log.Println("starting web server")
	log.Fatalln(http.ListenAndServe(":8080", http.DefaultServeMux))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Cookies())
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

	log.Println("Generating JWT")
	tokenString, err := generateJWT(time.Hour)
	if err != nil {
		log.Printf("generating token: %v", err)
	}

	responseBytes, err := json.Marshal(struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	})
	if err != nil {
		log.Printf("erro marshaling token: %v", err)
	}

	w.Write(responseBytes)
}

func handleLanding(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Println(r.Cookies())
	c := &http.Cookie{
		Name:    "server-set-cookie",
		Value:   "my suctom cookie",
		Expires: time.Now().Add(time.Hour),
	}
	http.SetCookie(w, c)
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

func generateJWT(expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "vlhl",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
	})

	tokenString, err := token.SignedString([]byte(SECRET))
	if err != nil {
		return "", fmt.Errorf("signing token: %v", err)
	}
	return tokenString, nil
}

func validateJWT(tokenString string) (bool, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) { return []byte(SECRET), nil })
	if err != nil {
		return false, err
	}
	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return false, err
	}

	log.Println("tokene validated, issuer is", issuer)
	return true, nil
}

func corsMiddleware(handler http.Handler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	}
}

func jwtMiddleware(handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		headerToken := r.Header.Get("Authorization")
		tokenString := strings.TrimPrefix(headerToken, "Bearer ")
		valid, err := validateJWT(tokenString)
		if err != nil {
			log.Printf("validating jwt: %v", err)
		}
		if !valid {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler.ServeHTTP(w, r)
	}
}
