package storage

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteLogFlushesBeforeReturning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")
	wal := &LocalWAL{}
	if err := wal.Connect(path); err != nil {
		t.Fatal(err)
	}
	defer wal.Close()

	if err := wal.WriteLog([]byte("entry")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "entry\n" {
		t.Fatalf("WAL contains %q, want %q", data, "entry\n")
	}
}

func TestWriteLogReturnsSyncError(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	wal := &LocalWAL{file: writer, writer: bufio.NewWriter(writer)}
	if err := wal.WriteLog([]byte("entry")); err == nil {
		t.Fatal("WriteLog succeeded even though the file cannot be synced")
	}
}
