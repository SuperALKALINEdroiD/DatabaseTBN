package handlers

import (
	"fmt"
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
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

		logs.AddWalEntry(config.WAL, key, value, grpcNode) // WAL dependency

		// rpc the node and insert
		// return some response

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Insert Endpoint WIP")
	}
}
