package nodes

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"github.com/emirpasic/gods/trees/redblacktree"
)

type internalNode struct {
	UnimplementedNodeServiceServer
	storage     storage.KVStore
	memTable    redblacktree.Tree
	dbConfig    config.DatabaseConfig
	memTableMux sync.RWMutex
}

func (server *internalNode) ManipulateNode(ctx context.Context, request *NodeManipulationRequest) (*NodeResponse, error) {
	log.Printf("Incoming %s request on manipulation procedure at %s", request.Operation, request.Node)

	server.memTableMux.Lock()
	defer server.memTableMux.Unlock()

	if request.Operation == Operation_CREATE {
		if server.shouldFlushToMemory() {
			server.flushMemTableToMemory()
		}
		server.memTable.Put(request.GetKey(), request.GetValue())
		log.Printf("Inserted using manipulation procedure at %s", request.Node)
	}

	return &NodeResponse{
		Status:       Status_OK,
		Timestamp:    time.Now().String(),
		Node:         request.Node,
		Key:          request.Key,
		Value:        request.Value,
		ErrorMessage: "",
	}, nil
}

func (server *internalNode) SearchNode(ctx context.Context, request *NodeSearchRequest) (*NodeResponse, error) {

	searhResult, _ := server.memTable.Get(request.Key)

	return &NodeResponse{
		Status:       Status_OK,
		Timestamp:    time.Now().String(),
		Node:         request.Node,
		Key:          request.Key,
		Value:        searhResult.(string),
		ErrorMessage: "Not Found",
	}, nil
}

func (server *internalNode) SearchNodeStream(request *NodeSearchRequest, stream NodeService_SearchNodeStreamServer) error {
	for i := 0; i < 5; i++ { // Sending 5 dummy responses
		err := stream.Send(&NodeResponse{
			Status:       Status_OK,
			Timestamp:    time.Now().String(),
			Node:         request.Node,
			Key:          request.Key,
			Value:        "StreamedDummyValue",
			ErrorMessage: "",
		})
		if err != nil {
			return err
		}
		time.Sleep(time.Second) // Simulating delay
	}
	return nil
}

func (server *internalNode) BatchSearch(ctx context.Context, request *NodeBatchRequest) (*NodeBatchResponse, error) {
	responses := []*NodeResponse{}
	for _, node := range request.Searches {
		responses = append(responses, &NodeResponse{
			Status:       Status_OK,
			Timestamp:    time.Now().String(),
			Node:         node.Node,
			Key:          "DummyKey",
			Value:        "BatchDummyValue",
			ErrorMessage: "",
		})
	}
	return &NodeBatchResponse{Responses: responses}, nil
}

func (server *internalNode) BatchManipulate(ctx context.Context, request *NodeBatchRequest) (*NodeBatchResponse, error) {
	responses := []*NodeResponse{}
	for _, node := range request.Searches {

		responses = append(responses, &NodeResponse{
			Status:       Status_OK,
			Timestamp:    time.Now().String(),
			Node:         node.Node,
			Key:          "DummyKey",
			Value:        "BatchManipulatedDummyValue",
			ErrorMessage: "",
		})
	}
	return &NodeBatchResponse{Responses: responses}, nil
}

func (server *internalNode) StreamNodeUpdates(stream NodeService_StreamNodeUpdatesServer) error {
	for {
		request, err := stream.Recv()
		if err != nil {
			return err
		}

		err = stream.Send(&NodeResponse{
			Status:       Status_OK,
			Timestamp:    time.Now().String(),
			Node:         request.Node,
			Key:          "StreamedKey",
			Value:        "StreamedUpdateValue",
			ErrorMessage: "",
		})
		if err != nil {
			return err
		}
	}
}

func (server *internalNode) shouldFlushToMemory() bool {
	nodeCount := server.memTable.Size()
	var sampleNode redblacktree.Node
	nodeSize := int64(unsafe.Sizeof(sampleNode))
	totalSize := int64(nodeCount) * nodeSize

	return totalSize > server.dbConfig.InMemoryStorageThreshold
}

func (server *internalNode) flushMemTableToMemory() {
	log.Println("Starting memory flush to persistant storage")
	defer log.Println("Completed Memory write, Continuing normal operations")

	keysInTable := server.memTable.Keys()

	for index, value := range keysInTable {
		fmt.Println(index, value.(string))
	}

	defer server.memTable.Clear()

}
