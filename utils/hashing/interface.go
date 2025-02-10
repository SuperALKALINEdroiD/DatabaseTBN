package hashing

type NodeHash interface {
	AddNode(nodeID string)
	RemoveNode(nodeID string)
	GetNode(key string) (string, error)
	GetAllNodes() []string
}
