package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
)

func InsertHandler(appConfig *core.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if appConfig.Config.MetaDataConfig.State != config.NodeStateReady {
			http.Error(w, "Database is not in ready state", http.StatusTooEarly)
			return
		}

		key := r.URL.Query().Get("key")

		if key == "" {
			http.Error(w, "Missing key", http.StatusUnprocessableEntity)
			return
		}

		value := r.URL.Query().Get("value")

		response, err := InsertPair(appConfig, key, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func InsertPair(appConfig *core.App, key string, value string) (*nodes.NodeResponse, error) {
	grpcNodeID, hashError := appConfig.NodeHashInfo.GetNode(key)
	if hashError != nil {
		return nil, fmt.Errorf("unable to locate node for key %q: %w", key, hashError)
	}

	logs.AddWalEntry(appConfig.WAL, key, value, grpcNodeID)

	destNode, ok := appConfig.NodeByID[grpcNodeID]
	if !ok {
		return nil, fmt.Errorf("node %q not found", grpcNodeID)
	}

	grpcClient, ok := appConfig.NodeClients[grpcNodeID]
	if !ok {
		return nil, fmt.Errorf("no gRPC client for node %q", grpcNodeID)
	}

	insertionPayload := &nodes.NodeManipulationRequest{
		Node:      destNode.Address,
		Key:       key,
		Value:     value,
		Operation: nodes.Operation_CREATE,
	}

	response, err := grpcClient.ManipulateNode(context.Background(), insertionPayload)
	if err != nil {
		return nil, fmt.Errorf("gRPC insert failed for node %q: %w", grpcNodeID, err)
	}

	return response, nil
}
