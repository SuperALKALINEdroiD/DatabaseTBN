package manifest

import "time"

type Manifest struct {
	Version   string            `json:"version"`
	CreatedAt time.Time         `json:"createdAt"`
	SSTables  []SSTableMetadata `json:"sstables"`
}

type SSTableMetadata struct {
	Path      string `json:"path"`
	FileName  string `json:"fileName"`
	BloomFile string `json:"bloomFile"`
}
