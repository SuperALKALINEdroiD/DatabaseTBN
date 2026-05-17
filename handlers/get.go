package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
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

		value := r.URL.Query().Get("value")

		grpcNode, hashError := appConfig.NodeHashInfo.GetNode(key)

		if hashError != nil {
			panic("Unable to get a node to store data")
		}

		logs.AddWalEntry(appConfig.WAL, key, value, grpcNode)

		destinationNodeIndex := sort.Search(len(appConfig.Nodes), func(i int) bool { return appConfig.Nodes[i].ID == grpcNode }) % len(appConfig.Nodes)

		grpcClient, connection := nodes.StartGRPCClient(appConfig.Nodes[destinationNodeIndex].Address)
		defer connection.Close()

		searchPayload := &nodes.NodeSearchRequest{
			Node: appConfig.Nodes[destinationNodeIndex].Address,
			Key:  key,
		}

		response, error := grpcClient.SearchNode(context.Background(), searchPayload)

		if error != nil {
			panic(error)
		}

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
