package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/handlers"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
	"github.com/google/uuid"
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
	return common.GetAppPath()
}

func initRouter(app *core.App) *http.ServeMux {
	mux := http.NewServeMux()
	initRoutes(mux, app)
	return mux
}

func initRoutes(mux *http.ServeMux, app *core.App) {
	mux.HandleFunc("GET /data-in/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Server is running")
	})

	mux.HandleFunc("POST /data-in/upsert", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Upsert Endpoint WIP - Config: %+v", app)
	})

	mux.HandleFunc("POST /data-in/insert", handlers.InsertHandler(app))

	mux.HandleFunc("GET /data-in/", handlers.GetValue(app))

	mux.HandleFunc("POST /data-in/update", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Update Endpoint WIP - Config: %+v", app)
	})
}

func middleware(h http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	for _, value := range slices.Backward(m) {
		h = value(h)
	}

	return h
}

func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			r.RemoteAddr = ip
		} else if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), "requestID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
