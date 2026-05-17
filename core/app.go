package core

import (
	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/hashing"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Config       *config.DatabaseConfig
	Router       *chi.Mux
	Nodes        []*nodes.Node
	NodeByID     map[string]*nodes.Node
	NodeClients  map[string]nodes.NodeServiceClient
	NodeHashInfo hashing.NodeHash
	WAL          storage.WAL
}
