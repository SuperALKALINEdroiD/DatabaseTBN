package storage

import (
	"bufio"
	"errors"
	"log"
	"os"
	"sync"

	internalerrors "github.com/SuperALKALINEdroiD/timelyDB/utils/internalErrors"
)

type LocalWAL struct {
	file    *os.File
	mutex   sync.Mutex
	maxSize int
}

func openLocalStorageFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
}

func (localWAL *LocalWAL) Connect(config map[string]interface{}) error {
	path, ok := config["path"].(string) // type assertion to string
	if !ok {
		return internalerrors.InvalidPath
	}

	maxSize, ok := config["maxSize"].(int) // TODO: see if i can do something with this
	if !ok {
		return errors.New("invalid max size for WAL")
	}

	file, err := openLocalStorageFile(path)

	if err != nil {
		log.Println("Error opening file:", err)
		return err
	}

	localWAL.file = file
	localWAL.maxSize = maxSize

	return nil
}

func (localWAL *LocalWAL) Close() error {
	return localWAL.file.Close()
}

func (localWAL *LocalWAL) GetSize() (int, error) {
	fileInfo, err := localWAL.file.Stat()
	if err != nil {
		return 0, err
	}
	return int(fileInfo.Size()), nil
}

func (localWAL *LocalWAL) WriteLog(data []byte) error {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	_, err := localWAL.file.Write(data)
	if err != nil {
		log.Println("Error writing log:", err)
		return err
	}

	localWAL.file.Sync() // TODO: find a better time to sync changes to stable storage

	return nil
}

func (localWAL *LocalWAL) ReadLog(startLine, endLine int) ([]string, error) {
	localWAL.mutex.Lock()
	defer localWAL.mutex.Unlock()

	if startLine < 0 || endLine <= startLine {
		return nil, errors.New("invalid line range")
	}

	file, err := os.Open(localWAL.file.Name())
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
}

func (localKVStore LocalKVStore) Connect(config map[string]interface{}) error {

}

func (localKVStore LocalKVStore) Close() error {

}

func (localKVStore *LocalKVStore) GetSize() (int, error) {
	return 0, nil
}

func (localKVStore LocalKVStore) Put(key string, value []byte) error {
	return nil
}

func (localKVStore LocalKVStore) Get(key string) (value []byte, error error) {
	return nil, nil
}

func (localKVStore LocalKVStore) Delete(key string) error {
	return nil
}

type LocalLogStore struct {
}

func (localLogStore LocalLogStore) Connect() {

}
