package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Println("Exiting, Bye!")
}

func run() (runErr error) {
	f, err := os.Create("cpu_profile.prof")
	if err != nil {
		return fmt.Errorf("failed to create profiler: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	var wal *storage.LocalWAL
	var server *http.Server
	nodeConns := make(map[string]*grpc.ClientConn)
	defer func() {
		stop()
		if server != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := server.Shutdown(shutdownCtx); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("server shutdown failed: %w", err))
			}
			shutdownCancel()
		}
		for nodeID, conn := range nodeConns {
			if err := conn.Close(); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("failed to close gRPC client for node %s: %w", nodeID, err))
			}
		}
		if wal != nil {
			if err := wal.Close(); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("failed to close WAL: %w", err))
			}
		}
		if err := f.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("failed to close profiler: %w", err))
		}
	}()

	config, configLoadError := initEnvironment()

	if configLoadError != nil {
		return fmt.Errorf("error while loading config: %w", configLoadError)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to find user config directory: %w", err)
	}
	appPath := filepath.Join(configDir, "timely")

	walPath := filepath.Join(appPath, config.MetaDataConfig.WALName)
	connectedWAL := &storage.LocalWAL{}
	if err := connectedWAL.Connect(walPath); err != nil {
		return fmt.Errorf("failed to open WAL at %q: %w", walPath, err)
	}
	wal = connectedWAL

	grpcNodes, nodeHashInfo := nodes.LoadServers(ctx, config, wal)

	storageNodesIndex := make(map[string]*nodes.Node, len(grpcNodes))
	nodeClients := make(map[string]nodes.NodeServiceClient, len(grpcNodes))
	for _, n := range grpcNodes {
		if n == nil {
			continue
		}
		storageNodesIndex[n.ID] = n
		conn, connErr := grpc.NewClient(n.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			return fmt.Errorf("failed to create gRPC client for node %s: %w", n.ID, connErr)
		}
		nodeClients[n.ID] = nodes.NewNodeServiceClient(conn)
		nodeConns[n.ID] = conn
	}

	app := &core.App{
		Config:       config,
		Nodes:        grpcNodes,
		NodeByID:     storageNodesIndex,
		NodeClients:  nodeClients,
		NodeConns:    nodeConns,
		NodeHashInfo: nodeHashInfo,
		WAL:          wal,
	}

	logs.ReplayLogs(app)

	router := initRouter(app)
	handler := middleware(router, realIP, requestID, requestLogger)

	serverAddress := fmt.Sprintf(":%d", app.Config.Port)
	log.Printf("Starting %s server on %s", config.StoreName, serverAddress)
	server = &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting to listen on %s", serverAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Received shutdown signal. Stopping servers...")
	case err := <-serverErrors:
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}
