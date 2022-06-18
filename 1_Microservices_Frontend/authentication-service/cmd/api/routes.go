package main

import (
	"github.com/go-chi/chi/v4"

	"github.com/go-chi/chi/v4/middleware"
	"github.com/go-chi/cors"
	"net/http"
)

func (app *Config) Routes() http.Handler {
	mux := chi.NewRouter()

	// who is allowed to connect

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(middleware.Heartbeat("/ping"))

	mux.Post("/authenticate", app.Authenticate)

	return mux

}
