package nodes

import (
	"context"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
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
	prefix := common.LogPrefix()
	log.Printf("%s :: Incoming %s request on manipulation procedure at %s", prefix, request.Operation, request.Node)

	server.memTableMux.Lock()
	defer server.memTableMux.Unlock()

	if request.Operation == Operation_CREATE {
		if true || server.shouldFlushToMemory() { // TODO: remove true
			defer server.flushMemTableToMemory()
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

	searhResult, found := server.memTable.Get(request.Key)

	if !found {
		diskValue, diskFound, err := server.lookupFromDisk(request.Key)
		if err != nil {
			log.Printf("disk lookup failed for key %s: %v", request.Key, err)
		}
		if diskFound {
			return &NodeResponse{
				Status:       Status_OK,
				Timestamp:    time.Now().String(),
				Node:         request.Node,
				Key:          request.Key,
				Value:        diskValue,
				ErrorMessage: "",
			}, nil
		}

		return &NodeResponse{
			Status:       Status_NOT_FOUND,
			Timestamp:    time.Now().String(),
			Node:         request.Node,
			Key:          request.Key,
			Value:        "",
			ErrorMessage: "Not Found",
		}, nil
	}

	return &NodeResponse{
		Status:       Status_OK,
		Timestamp:    time.Now().String(),
		Node:         request.Node,
		Key:          request.Key,
		Value:        searhResult.(string),
		ErrorMessage: "",
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
	if nodeCount == 0 {
		return false
	}

	var sampleNode redblacktree.Node
	nodeOverhead := int64(unsafe.Sizeof(sampleNode))

	keys := server.memTable.Keys()
	totalDataSize := int64(0)

	for _, key := range keys {
		if keyStr, ok := key.(string); ok {
			totalDataSize += int64(len(keyStr))
		}

		// Size of value string
		if value, ok := server.memTable.Get(key); ok {
			if valueStr, ok := value.(string); ok {
				totalDataSize += int64(len(valueStr))
			}
		}
	}

	const stringHeaderSize = 16 //
	totalSize := (int64(nodeCount) * nodeOverhead) + totalDataSize + (int64(nodeCount) * 2 * stringHeaderSize)

	return totalSize > server.dbConfig.InMemoryStorageThreshold
}
