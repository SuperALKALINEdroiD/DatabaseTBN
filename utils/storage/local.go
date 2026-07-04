package storage

import (
	"bufio"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

type LocalWAL struct {
	path   string
	file   *os.File
	writer *bufio.Writer
	done   chan struct{}
	mutex  sync.Mutex
}

func (localWAL *LocalWAL) Connect(path string) error {
	localWAL.path = path

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	localWAL.file = file
	localWAL.writer = bufio.NewWriterSize(file, 64*1024) // 64KB buffer
	localWAL.done = make(chan struct{})

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				localWAL.mutex.Lock()
				_ = localWAL.writer.Flush()
				_ = localWAL.file.Sync()
				localWAL.mutex.Unlock()
			case <-localWAL.done:
				return
			}
		}
	}()

	return nil
}

func (localWAL *LocalWAL) GetPath() string {
	return localWAL.path
}

func (localWAL *LocalWAL) WriteLog(data []byte) error {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	_, err := localWAL.writer.Write(append(data, '\n'))
	if err != nil {
		log.Println("Error writing log:", err)
		return err
	}

	return nil
}

func (localWAL *LocalWAL) Flush() error {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	if err := localWAL.writer.Flush(); err != nil {
		return err
	}
	return localWAL.file.Sync()
}

func (localWAL *LocalWAL) Close() error {
	close(localWAL.done)
	if err := localWAL.Flush(); err != nil {
		return err
	}
	return localWAL.file.Close()
}

func (localWAL *LocalWAL) GetSize() (int, error) {
	fileInfo, err := localWAL.file.Stat()
	if err != nil {
		return 0, err
	}
	return int(fileInfo.Size()), nil
}

func (localWAL *LocalWAL) GetTotalLines() (int, error) {
	file, err := os.Open(localWAL.path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	return lineCount, nil
}

func (localWAL *LocalWAL) ReadLog(lineNumber ...int) ([]string, error) {
	localWAL.mutex.Lock()
	// Flush before reading so buffered writes are visible
	_ = localWAL.writer.Flush()
	localWAL.mutex.Unlock()

	var startLine, endLine int
	var err error

	if len(lineNumber) == 0 {
		startLine = 0
		endLine, err = localWAL.GetTotalLines()
		if err != nil {
			return nil, err
		}
	}
	if len(lineNumber) == 1 {
		startLine = lineNumber[0]
		endLine, err = localWAL.GetTotalLines()
		if err != nil {
			return nil, err
		}
	}
	if len(lineNumber) == 2 {
		startLine = lineNumber[0]
		endLine = lineNumber[1]
	}
	if len(lineNumber) > 2 {
		return nil, errors.New("invalid line range")
	}

	file, err := os.Open(localWAL.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 0

	for scanner.Scan() {
		if lineNum >= startLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum >= endLine-1 {
			break
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, errors.New("startLine is beyond total lines in file")
	}

	return lines, nil
}

type LocalKVStore struct {
	mutex sync.RWMutex
}

func (localKVStore *LocalKVStore) Connect(path string) error {
	return nil
}

func (localKVStore *LocalKVStore) Close() error {
	return nil
}

func (localKVStore *LocalKVStore) GetSize() (int, error) {
	return 0, nil
}

func (localKVStore *LocalKVStore) Put(key string, value []byte) error {
	return nil
}

func (localKVStore *LocalKVStore) Get(key string) (value []byte, error error) {
	return nil, nil
}

func (localKVStore *LocalKVStore) Delete(key string) error {
	return nil
}

func (LocalKVStore *LocalKVStore) Compaction() error {
	return nil
}

type LocalLogStore struct {
}

func (localLogStore LocalLogStore) Connect() {

}
