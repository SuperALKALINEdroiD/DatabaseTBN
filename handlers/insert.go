package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
)

func InsertHandler(config *core.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		key := r.URL.Query().Get("key")

		if key == "" {
			http.Error(w, "Missing key", http.StatusUnprocessableEntity)
			return
		}

		value := r.URL.Query().Get("value")

		// find the node where this key will be saved
		grpcNode, hashError := config.NodeHashInfo.GetNode(key)

		if hashError != nil {
			panic("Unable to get a node to store data")
		}

		logs.AddWalEntry(config.WAL, key, value, grpcNode)

		destinationNodeIndex := sort.Search(len(config.Nodes), func(i int) bool { return config.Nodes[i].ID == grpcNode })

		grpcClient, connection := nodes.StartGRPCClient(config.Nodes[destinationNodeIndex].Address)
		defer connection.Close()

		insertionPayload := &nodes.NodeManipulationRequest{
			Node:      config.Nodes[destinationNodeIndex].Address,
			Key:       key,
			Value:     value,
			Operation: nodes.Operation_CREATE,
		}

		response, error := grpcClient.ManipulateNode(context.Background(), insertionPayload)

		if error != nil {
			panic(error)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response.Status)
	}
}
