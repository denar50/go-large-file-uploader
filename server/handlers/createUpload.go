package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"server/fileManager"
)

type CreateUploadRequestBody struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
}

type CreateUploadResponseBody struct {
	Id         string `json:"id"`
	ChunkSize  int    `json:"chunk_size"`
	ChunkCount int    `json:"chunk_count"`
}

// POST /upload
func createUpload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got a request to upload a file!")
	var b CreateUploadRequestBody

	err := json.NewDecoder(r.Body).Decode(&b)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := fileManager.Manager.AllocateFile(b.Name, b.Size, b.Checksum)

	if err != nil {
		fmt.Printf("Error while allocating the file %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	responseBody, err := json.Marshal(CreateUploadResponseBody{Id: result.Id, ChunkSize: result.ChunkSize, ChunkCount: result.ChunkCount})
	if err != nil {
		fmt.Printf("Error creating the response body %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(responseBody)

	if err != nil {
		fmt.Printf("Error writing the JSON response %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
