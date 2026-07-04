package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	f, err := os.Create("cpu_profile.prof")
	if err != nil {
		panic(err)
	}
	defer f.Close()

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

	wal := &storage.LocalWAL{}
	appPath := common.GetAppPath()
	wal.Connect(filepath.Join(appPath, config.MetaDataConfig.WALName))

	grpcNodes, nodeHashInfo := nodes.LoadServers(ctx, config, wal)

	storageNodesIndex := make(map[string]*nodes.Node, len(grpcNodes))
	nodeClients := make(map[string]nodes.NodeServiceClient, len(grpcNodes))
	nodeConns := make(map[string]*grpc.ClientConn, len(grpcNodes))

	for _, n := range grpcNodes {
		if n == nil {
			continue
		}
		storageNodesIndex[n.ID] = n
		conn, connErr := grpc.NewClient(n.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			log.Fatalf("failed to create gRPC client for node %s: %v", n.ID, connErr)
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
	server := &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("Starting to listen on %s", serverAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down main server...")

	if err := app.WAL.Flush(); err != nil {
		log.Printf("WAL flush on shutdown failed: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	for nodeID, conn := range app.NodeConns {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close gRPC client for node %s: %v", nodeID, err)
		}
	}

	log.Println("Exiting, Bye!")

}
