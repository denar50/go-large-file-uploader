package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"server/fileManager"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GetFileChunkResponse struct {
	Checksum     string `json:"checksum"`
	FromPosition int64  `json:"from_position"`
}

func getFileChunk(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		fmt.Printf("Invalid id %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chunkPosition, err := strconv.Atoi(chi.URLParam(r, "chunk_position"))
	if err != nil {
		fmt.Printf("Error while parsing the chunk position %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Receive request to fetch chunk ", id, chunkPosition)

	result, err := fileManager.Manager.GetFileChunk(id, chunkPosition)

	if err != nil {
		fmt.Printf("Error while retrieving the chunk file %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mw := multipart.NewWriter(w)

	defer mw.Close()

	w.Header().Set("Content-Type", mw.FormDataContentType())

	jsonPart, err := mw.CreateFormField("json")

	if err != nil {
		fmt.Printf("Error creating the form field %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonEncoder := json.NewEncoder(jsonPart)
	if err := jsonEncoder.Encode(GetFileChunkResponse{Checksum: result.Checksum, FromPosition: result.FromPosition}); err != nil {
		fmt.Printf("Error encoding response %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filePart, err := mw.CreateFormFile("file", fmt.Sprintf("%s-%d", id, chunkPosition))
	if err != nil {
		fmt.Println("Error creating the file form field", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chunkFileHandle, err := os.Open(result.ChunkFileLocation)

	if err != nil {
		fmt.Println("error opening the chunk file", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer chunkFileHandle.Close()

	_, err = io.Copy(filePart, chunkFileHandle)
	if err != nil {
		fmt.Println("Error writing the file chunk to the response form", err.Error(), fmt.Sprintf("%s/%d", id, chunkPosition))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
