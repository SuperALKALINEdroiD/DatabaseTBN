package nodes

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		server.flushMemTableToMemory()
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
	if !server.shouldFlushToMemory() {
		log.Println("In memory storage within threshold, skipped memory write")
		return
	}
	log.Println("Starting memory flush to persistent storage")
	defer log.Println("Completed Memory write, Continuing normal operations")

	keys := server.memTable.Keys()
	kvData := make(map[any]any)

	for _, key := range keys {
		value, ok := server.memTable.Get(key)
		if ok {
			kvData[key] = value
		}
	}

	err := server.atomicFlushToDisk(kvData)
	if err != nil {
		log.Printf("Error during flush: %v", err)
		return
	}

	server.memTable.Clear()
}

func (server *internalNode) atomicFlushToDisk(kvData map[any]any) error {
	manifest := server.dbConfig.Manifest
	basePath := manifest.SSTables[0].Path // temporary

	tmpSST := filepath.Join(basePath, "kv.tmp.sst")
	tmpBloom := filepath.Join(basePath, "filter.tmp.bf")

	finalSST := filepath.Join(basePath, "kv.sst")
	finalBloom := filepath.Join(basePath, "filter.bf")

	if err := server.writeToSSt(); err != nil {
		return fmt.Errorf("SST write failed: %w", err)
	}

	if err := server.writeToBloom(); err != nil {
		os.Remove(tmpSST)
		return fmt.Errorf("BLOOM WRITE FAILED: %w", err)
	}

	if err := os.Rename(tmpSST, finalSST); err != nil {
		os.Remove(tmpSST)
		os.Remove(tmpBloom)
		return fmt.Errorf("SST WRITE FAILED: %w", err)
	}
	if err := os.Rename(tmpBloom, finalBloom); err != nil {
		os.Remove(finalSST)
		return fmt.Errorf("TEMP RENAME FAILED: %w", err)
	}

	return nil
}

func (Server *internalNode) writeToBloom() error {
	return nil
}

func (Server *internalNode) writeToSSt() error {
	return nil
}
