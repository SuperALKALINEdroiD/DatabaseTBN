package hashing

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

type ConsistentHashing struct {
	nodes            []int
	nodeMap          map[int]string
	virtualNodeCount int // how many time each node appears on hasing; from GS video for load balancing on distribited system
	mutex            sync.RWMutex
}

func NewConsistentHashing(virtualNodeCount int) *ConsistentHashing {
	return &ConsistentHashing{
		nodes:            []int{},
		nodeMap:          make(map[int]string),
		virtualNodeCount: virtualNodeCount,
	}
}

func (c *ConsistentHashing) hashKey(key string) int {
	return int(crc32.ChecksumIEEE([]byte(key)))
}

func (c *ConsistentHashing) AddNode(nodeID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i := 0; i < c.virtualNodeCount; i++ {
		hash := c.hashKey(fmt.Sprintf("%s-%d", nodeID, i))
		c.nodes = append(c.nodes, hash)
		c.nodeMap[hash] = nodeID
	}

	sort.Ints(c.nodes)
}

func (c *ConsistentHashing) RemoveNode(nodeID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i := 0; i < c.virtualNodeCount; i++ {
		hash := c.hashKey(fmt.Sprintf("%s-%d", nodeID, i))
		delete(c.nodeMap, hash)
	}

	c.nodes = c.nodes[:0]
	for key := range c.nodeMap {
		c.nodes = append(c.nodes, key)
	}

	sort.Ints(c.nodes)
}

func (c *ConsistentHashing) GetNode(key string) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.nodes) == 0 {
		return "", errors.New("no nodes in the hash ring")
	}

	hash := c.hashKey(key)

	idx := sort.Search(len(c.nodes), func(i int) bool { return c.nodes[i] >= hash })
	if idx == len(c.nodes) { // circular ring
		idx = 0
	}

	return c.nodeMap[c.nodes[idx]], nil
}

func (c *ConsistentHashing) GetAllNodes() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	nodeSet := make(map[string]struct{})
	for _, node := range c.nodeMap {
		nodeSet[node] = struct{}{}
	}

	nodes := make([]string, 0, len(nodeSet))
	for node := range nodeSet {
		nodes = append(nodes, node)
	}

	return nodes
}
