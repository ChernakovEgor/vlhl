package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/upload", handleUpload)
	log.Println("starting web server")
	log.Fatalln(http.ListenAndServe(":8090", http.DefaultServeMux))
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
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Write([]byte("File uploaded!"))
}
