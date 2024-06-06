package main

import (
	"net/http"

	"server/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	handlers.InitializeRouter(r)
	http.ListenAndServe(":3000", r)
}
