package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"server/fileManager"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GetFileResponseBody struct {
	Id         string `json:"id"`
	Checksum   string `json:"checksum"`
	Size       int64  `json:"size"`
	ChunkCount int    `json:"chunk_count"`
}

func getFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		fmt.Printf("Invalid id %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := fileManager.Manager.GetFileInfo(id)

	if err != nil {
		fmt.Printf("Error while looking for the chunk %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	marshalledResponse, err := json.Marshal(GetFileResponseBody{Id: result.Id, Checksum: result.Checksum, Size: result.Size, ChunkCount: result.ChunkCount})

	if err != nil {
		fmt.Printf("Error while looking for the chunk %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(marshalledResponse)

	if err != nil {
		fmt.Printf("Error writing the JSON response %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
