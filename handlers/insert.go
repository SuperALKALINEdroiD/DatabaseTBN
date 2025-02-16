package handlers

import (
	"fmt"
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
)

func InsertHandler(config storage.WAL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logs.AddWalEntry(config) // WAL dependency

		// TODO: send data to be logged into WAL for later reconstruction
		// find which node to access
		// rpc the node and insert
		// return some response

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Insert Endpoint WIP")
	}
}
