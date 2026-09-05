package manifest

import "time"

type Manifest struct {
	Version          string            `json:"version"`
	CreatedAt        time.Time         `json:"createdAt"`
	SSTables         []SSTableMetadata `json:"sstables"`
	NextSegmentID    uint64            `json:"nextSegmentID"`
	NodeCheckpoints  map[string]uint64 `json:"nodeCheckpoints"` // nodeID -> WAL line count at last flush
}

type SSTableMetadata struct {
	Path      string `json:"path"`
	FileName  string `json:"fileName"`
	BloomFile string `json:"bloomFile"`
	SegmentID uint64 `json:"segmentID"`
}
