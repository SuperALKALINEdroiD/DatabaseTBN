package nodes

import (
	context "context"
	"log"
	sync "sync"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"github.com/emirpasic/gods/trees/redblacktree"
)

type internalNode struct {
	UnimplementedNodeServiceServer
	Storage     storage.KVStore
	MemTable    redblacktree.Tree
	memTableMux sync.RWMutex
}

func (server *internalNode) ManipulateNode(ctx context.Context, request *NodeManipulationRequest) (*NodeResponse, error) {
	log.Printf("Incoming %s request on manipulation procedure at %s", request.Operation, request.Node)

	server.memTableMux.Lock()
	defer server.memTableMux.Unlock()

	if request.Operation == Operation_CREATE {
		memTableSize := server.getTreeSize()
		if memTableSize > 0 {
			server.flushMemTableToMemory()
		}
		server.MemTable.Put(request.GetKey(), request.GetValue())
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
	return &NodeResponse{
		Status:       Status_OK,
		Timestamp:    time.Now().String(),
		Node:         request.Node,
		Key:          request.Key,
		Value:        "DummyValue", // Dummy response
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

func (server *internalNode) getTreeSize() int {
	return 2
}

func (server *internalNode) flushMemTableToMemory() {

	server.MemTable.Clear()
}
