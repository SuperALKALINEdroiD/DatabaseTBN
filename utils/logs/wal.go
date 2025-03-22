package logs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/storage"
	"github.com/google/uuid"
)

type EntryStatus string

const (
	Pending   EntryStatus = "PENDING"
	Committed EntryStatus = "COMMITTED"
	Failed    EntryStatus = "FAILED"
)

type WriteAheadEntry struct {
	EntryID   string
	NodeID    string    // node where data will be saved
	Timestamp time.Time // entry time
	Data      []byte    // The actual data being written
	Status    EntryStatus
}

func AddWalEntry(wal storage.WAL, key string, value string, nodeId string) {
	writeAheadEntry := WriteAheadEntry{
		EntryID:   uuid.New().String(),
		NodeID:    nodeId,
		Timestamp: time.Now(),
		Data:      fmt.Appendf(nil, "%s:::%s", key, value),
		Status:    Committed,
	}

	logData, err := json.Marshal(writeAheadEntry)
	if err != nil {
		panic("Failed to serialize the log entry")
	}

	logData = append(logData, '\n')

	if err := wal.WriteLog(logData); err != nil {
		panic("Failed to write to write-ahead logs")
	}

	log.Printf("ADDED: %s :: %s to Write Ahead Logs", key, value)
}
