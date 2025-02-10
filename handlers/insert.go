package handlers

import (
	"fmt"
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/logs"
)

func InsertHandler(w http.ResponseWriter, r *http.Request) {

	logs.AddWalEntry() // TODO: send data to be logged into WAL for later reconctruction

	// find which node to access
	// rpc the node and insert
	// return some response

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Insert Endpoint WIP")
}
