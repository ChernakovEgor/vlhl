package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

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
