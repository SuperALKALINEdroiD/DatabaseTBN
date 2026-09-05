package core

import (
	"net/http"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/hashing"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"google.golang.org/grpc"
)

type App struct {
	Config       *config.DatabaseConfig
	Router       *http.ServeMux
	Nodes        []*nodes.Node
	NodeByID     map[string]*nodes.Node
	NodeClients  map[string]nodes.NodeServiceClient
	NodeConns    map[string]*grpc.ClientConn
	NodeHashInfo hashing.NodeHash
	WAL          storage.WAL
}
