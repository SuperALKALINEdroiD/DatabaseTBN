package config

import "fmt"

type StoreMode string

type DatabaseConfig struct {
	StoreName                string         `json:"dbName"`
	Port                     int            `json:"port"`
	IsLockEnabled            bool           `json:"isLockEnabled"`
	TimelyConfig             TimelyConfig   `json:"timelyConfig"`
	Nodes                    []NodeConfig   `json:"nodes"`
	NodeCount                int            `json:"nodeCount"`
	Mode                     StoreMode      `json:"mode"`
	InMemoryStorageThreshold int64          `json:"inMemoryStorageThreshold"`
	MetaDataConfig           MetaDataConfig `json:"metaDataConfig"`
}

type MetaDataConfig struct {
	State   NodeState `json:"state"`
	WALPath string    `json:"walPath"`
}

type NodeState int

const (
	NodeStateBuilding NodeState = iota
	NodeStateReady
	NodeStateRebalancing
	NodeStateDown
)

func (s NodeState) String() string {
	return [...]string{"Building", "Ready", "Rebalancing", "Down"}[s]
}

const (
	KV   StoreMode = "KV"
	Logs StoreMode = "Logs"
)

type TimelyConfig struct {
	IsEnabled bool `json:"isEnabled"`
	Type      rune `json:"type"`
}

type NodeConfig struct {
	Endpoint string `json:"endpoint"`
}

func GenerateExampleConfig(nodeCount int, host string) DatabaseConfig {
	nodes := generateNodeConfig(nodeCount, host)

	return DatabaseConfig{
		StoreName:     "example_store",
		Port:          7001,
		IsLockEnabled: true,
		TimelyConfig: TimelyConfig{
			IsEnabled: true,
			Type:      'H',
		},
		Mode:                     KV,
		Nodes:                    nodes,
		NodeCount:                nodeCount,
		InMemoryStorageThreshold: 2000,
		MetaDataConfig: MetaDataConfig{
			State:   NodeStateReady,
			WALPath: "/var/lib/db/wal",
		},
	}
}

func generateNodeConfig(nodeCount int, host string) []NodeConfig {
	nodes := make([]NodeConfig, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = NodeConfig{
			Endpoint: fmt.Sprintf("%s:%d", host, 50051+i),
		}
	}
	return nodes
}
