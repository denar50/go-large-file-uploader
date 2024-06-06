package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/fileManager"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UploadChunkRequestBody struct {
	Checksum     string `json:"checksum"`
	FromPosition int64  `json:"from_position"`
	ChunkNumber  int    `json:"chunk_number"`
}

func uploadChunk(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		fmt.Printf("Invalid id %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Printf("Failed parsing multipart from %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawJson := r.FormValue("json")

	var body UploadChunkRequestBody
	if err = json.Unmarshal([]byte(rawJson), &body); err != nil {
		fmt.Printf("Failed unmarshalling request body %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chunkFile, _, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("Failed reading chunk file %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer chunkFile.Close()

	chunk, err := io.ReadAll(chunkFile)
	if err != nil {
		fmt.Printf("Failed reading chunk file %s", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sha := sha256.Sum256(chunk)
	checksum := hex.EncodeToString(sha[:])

	if body.Checksum != checksum {
		fmt.Printf("Checksums dont match received %s. Calculated %s", body.Checksum, checksum)
		http.Error(w, "Checksum doesnt match", http.StatusBadRequest)
		return
	}

	err = fileManager.Manager.ProcessChunk(id, body.ChunkNumber, body.FromPosition, checksum, chunk)

	if err != nil {
		fmt.Printf("Error while processing the chunk %s", err.Error())
		http.Error(w, "Internal server error", http.StatusBadRequest)
		return
	}
}
