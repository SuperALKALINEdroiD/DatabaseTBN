package storage

type Storage interface {
	Connect(path string) error // connect to storage
	GetSize() (int, error)     // get in memory size
}

type WAL interface {
	Storage
	WriteLog(data []byte) error // write WAL logs
	ReadLog(startLine, endLine int) ([]string, error)
	GetPath() string
	// TruncateLog(offset int64) error // clear log file
	// Reconstruct(offset int64) error // reconstruct the data from WAL
}

type KVStore interface {
	Storage
	Put(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

type LogStore interface {
	Storage
	WriteLog(app string, data []byte) error
	ReadLog(app string, offset int64) ([]byte, error)
	DeleteLogs(app string) error
}
