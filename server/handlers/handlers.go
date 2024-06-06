package handlers

import "github.com/go-chi/chi/v5"

func InitializeRouter(r *chi.Mux) {
	r.Post("/upload", createUpload)
	r.Put("/upload/{id}/chunk", uploadChunk)
	r.Get("/upload/{id}", getFile)
	r.Get("/upload/{id}/{chunk_position}", getFileChunk)
}
