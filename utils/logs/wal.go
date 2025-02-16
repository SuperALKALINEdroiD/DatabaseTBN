package logs

import (
	"encoding/json"
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
	NodeID    int       // node where data will be saved
	Timestamp time.Time // entry time
	Data      []byte    // The actual data being written
	Status    EntryStatus
}

func AddWalEntry(wal storage.WAL) {
	// TODO accept data to be logged as parameter

	writeAheadEntry := WriteAheadEntry{
		EntryID:   uuid.New().String(),
		NodeID:    0, // TODO: Retrieve actual node ID
		Timestamp: time.Now(),
		Data:      []byte("log data"), // Replace with actual data
		Status:    Committed,
	}

	logData, err := json.Marshal(writeAheadEntry)
	if err != nil {
		panic("Failed to serialize the log entry")
	}

	logData = append(logData, '\n') // Ensure entries are newline-separated

	if err := wal.WriteLog(logData); err != nil {
		panic("Failed to write to write-ahead logs")
	}
}
