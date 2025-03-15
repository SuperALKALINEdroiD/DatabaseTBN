package nodes

import (
	context "context"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/hashing"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	ID      string
	Address string
}

func nodeSetupTask(ctx context.Context, nodeID string, port string, config *config.DatabaseConfig) (*Node, error) {
	listener, httpError := net.Listen("tcp", port)
	if httpError != nil {
		return nil, fmt.Errorf("failed to start listener: %v", httpError)
	}

	grpcServer := grpc.NewServer()
	nodeStorage := storage.LocalKVStore{} // TODO: based on config
	dataStoreServer := &internalNode{Storage: &nodeStorage}
	RegisterNodeServiceServer(grpcServer, dataStoreServer)

	stop := make(chan struct{})

	go func() {
		log.Printf("Node %s: gRPC server started", nodeID)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
			close(stop)
		}
	}()

	go func() {
		<-ctx.Done() // context cancelled
		log.Printf("Shutting down gRPC server for Node %s...", nodeID)
		grpcServer.GracefulStop()
		listener.Close()
		close(stop)
	}()

	return &Node{ID: nodeID, Address: port}, nil
}

func LoadServers(ctx context.Context, config *config.DatabaseConfig) ([]*Node, hashing.NodeHash) {
	if len(config.Nodes) == 0 || config.NodeCount == 0 {
		log.Println("No node configuration found.")
		return []*Node{}, nil
	}

	log.Println("Loading nodes...")

	grpcNodes := make([]*Node, len(config.Nodes))
	clusterHashing := hashing.NewConsistentHashing(len(config.Nodes)) // todo add config for virtual node count

	for i, node := range config.Nodes {
		log.Printf("Node %d: Endpoint ==> %s\n", i+1, node.Endpoint)

		var setupError error
		grpcNodes[i], setupError = nodeSetupTask(ctx, strconv.Itoa(i), node.Endpoint, config)

		if setupError != nil {
			log.Printf("Error setting up Node %d: %v\n", i+1, setupError)
			continue
		}

		clusterHashing.AddNode(strconv.Itoa(i))
	}

	log.Println("Nodes are up and running.")
	return grpcNodes, clusterHashing
}

func StartGRPCClient(destNodeAddr string) (NodeServiceClient, *grpc.ClientConn) {
	formattedAddress := fmt.Sprintf("localhost%s", destNodeAddr)
	conn, err := grpc.NewClient(formattedAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("gRPC client started for node running at", formattedAddress)
	client := NewNodeServiceClient(conn)

	return client, conn
}
