package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/handlers"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func initEnvironment() (*config.DatabaseConfig, error) {
	var configPath = os.Getenv("LOG_BASE_SETTINGS")

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return nil, err
	}

	return cfg, nil
}

func GetAppPath() string {
	path := common.GetAppPath()

	return path
}

func initRouter(app *core.App) *chi.Mux {
	router := chi.NewRouter()
	addMiddlewares(router)
	initRoutes(router, app)
	return router
}

func addMiddlewares(router *chi.Mux) {
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
}

func initRoutes(router *chi.Mux, app *core.App) {
	// init routes based on config ??
	router.Route("/data-in", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Server is running")
		})

		r.Post("/upsert", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Upsert Endpoint WIP - Config: %+v", app)
		})

		r.Post("/insert", handlers.InsertHandler(app))

		r.Get("/", handlers.GetValue(app))

		r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Update Endpoint WIP - Config: %+v", app)
		})
	})
}
