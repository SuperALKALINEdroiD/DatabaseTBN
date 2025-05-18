package storage

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

type LocalWAL struct {
	path  string
	mutex sync.Mutex
}

func openLocalStorageFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
}

func (localWAL *LocalWAL) Connect(path string) error {
	localWAL.path = path
	return nil
}

func (localWAL *LocalWAL) GetPath() string {
	return localWAL.path
}

func (localWAL *LocalWAL) WriteLog(data []byte) error {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	file, err := openLocalStorageFile(localWAL.path)
	if err != nil {
		log.Println("Error opening WAL file:", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		log.Println("Error writing log:", err)
		return err
	}

	if err := file.Sync(); err != nil {
		log.Println("Error syncing file:", err)
		return err
	}

	fmt.Printf("Log Added at %s", localWAL.GetPath())

	return nil
}

func (localWAL *LocalWAL) GetSize() (int, error) {
	file, err := os.Open(localWAL.path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return int(fileInfo.Size()), nil
}

func (localWAL *LocalWAL) ReadLog(startLine, endLine int) ([]string, error) {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	if startLine < 0 || endLine <= startLine {
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
