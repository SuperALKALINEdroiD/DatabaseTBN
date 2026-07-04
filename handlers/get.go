package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
)

func GetValue(appConfig *core.App) http.HandlerFunc {
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

		grpcNode, hashError := appConfig.NodeHashInfo.GetNode(key)

		if hashError != nil {
			http.Error(w, fmt.Sprintf("unable to locate node for key %q", key), http.StatusInternalServerError)
			return
		}

		destNode, ok := appConfig.NodeByID[grpcNode]
		if !ok {
			http.Error(w, fmt.Sprintf("node %q not found", grpcNode), http.StatusInternalServerError)
			return
		}

		grpcClient, ok := appConfig.NodeClients[grpcNode]
		if !ok {
			http.Error(w, fmt.Sprintf("no gRPC client for node %q", grpcNode), http.StatusInternalServerError)
			return
		}

		searchPayload := &nodes.NodeSearchRequest{
			Node: destNode.Address,
			Key:  key,
		}

		rpcCtx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
		defer cancel()

		response, err := grpcClient.SearchNode(rpcCtx, searchPayload)
		if err != nil {
			http.Error(w, fmt.Sprintf("gRPC search failed for node %q: %v", grpcNode, err), http.StatusGatewayTimeout)
			return
		}

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
