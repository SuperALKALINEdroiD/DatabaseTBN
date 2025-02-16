package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/hashing"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Config       *config.DatabaseConfig
	Router       *chi.Mux
	Nodes        []*nodes.Node
	NodeHashInfo hashing.NodeHash
	WAL          storage.WAL
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT) // capture all signals on channel

	go func() {
		<-signalChannel
		log.Println("Received shutdown signal. Stopping servers...")
		cancel() // context cancelled: shutdown the servers
	}()

	config, configLoadError := initEnvironment()

	if configLoadError != nil {
		panic("error while loading config")
	}

	grpcNodes, nodeHashInfo := nodes.LoadNodes(ctx, config)
	wal := &storage.LocalWAL{}
	wal.Connect("test-path")

	app := &App{
		Config:       config,
		Router:       initRouter(wal),
		Nodes:        grpcNodes,
		NodeHashInfo: nodeHashInfo,
		WAL:          wal,
	}

	serverAddress := fmt.Sprintf(":%d", app.Config.Port)
	log.Printf("Starting server on %s", serverAddress)

	server := &http.Server{Addr: ":7001", Handler: app.Router}

	go func() {
		log.Printf("Starting server on %s", serverAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down main server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Bye")

}
