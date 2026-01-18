package nodes

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/persistance"
)

func (server *internalNode) flushMemTableToMemory() {
	log.Println("Starting memory flush to persistent storage")
	defer log.Println("Completed Memory write, Continuing normal operations")

	keys := server.memTable.Keys()
	kvData := make(map[any]any)

	for _, key := range keys {
		value, ok := server.memTable.Get(key)
		if ok {
			kvData[key] = value
		}
	}

	err := server.atomicFlushToDisk(kvData)
	if err != nil {
		log.Printf("Error during flush: %v", err)
		return
	}

	server.memTable.Clear()
}

func (server *internalNode) atomicFlushToDisk(kvData map[any]any) error {
	// TODO: Implement manifest handling: not sure yet how to handle it
	manifest := server.dbConfig.Manifest
	basePath := manifest.SSTables[0].Path // temporary

	tmpSST := filepath.Join(basePath, "kv.tmp.sst")
	tmpBloom := filepath.Join(basePath, "filter.tmp.bf")

	finalSST := filepath.Join(basePath, "kv.sst")
	finalBloom := filepath.Join(basePath, "filter.bf")

	if err := server.writeToSSt(kvData, tmpSST); err != nil {
		return fmt.Errorf("SST write failed: %w", err)
	}

	if err := server.writeToBloom(kvData, tmpBloom); err != nil {
		os.Remove(tmpSST)
		return fmt.Errorf("BLOOM WRITE FAILED: %w", err)
	}

	if err := os.Rename(tmpSST, finalSST); err != nil {
		os.Remove(tmpSST)
		os.Remove(tmpBloom)
		return fmt.Errorf("SST WRITE FAILED: %w", err)
	}
	if err := os.Rename(tmpBloom, finalBloom); err != nil {
		os.Remove(finalSST)
		return fmt.Errorf("TEMP RENAME FAILED: %w", err)
	}

	return nil
}

func (server *internalNode) writeToBloom(kvData map[any]any, path string) error {
	if len(kvData) == 0 {
		return nil
	}

	filerSize := persistance.BitArraySize(float64(len(kvData)))
	fmt.Println("Filer size", filerSize)
	bf := persistance.NewBloomFilter(filerSize, 7)

	for key := range kvData {
		fmt.Println("Key", key)
		if keyStr, ok := key.(string); ok {
			bf.Add(keyStr)
		}
		fmt.Println("Added key", key)
	}
	fmt.Println("Saving bloom filter", path)
	return bf.Save(path)
}

func (server *internalNode) writeToSSt(kvData map[any]any, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create SST file: %w", err)
	}
	defer file.Close()

	numEntries := uint64(len(kvData))
	if err := binary.Write(file, binary.LittleEndian, numEntries); err != nil {
		return fmt.Errorf("failed to write entry count: %w", err)
	}

	for key, value := range kvData {
		keyStr, keyOk := key.(string)
		valueStr, valueOk := value.(string)

		if !keyOk || !valueOk {
			log.Printf("Warning: skipping non-string key-value pair")
			continue
		}

		keyBytes := []byte(keyStr)
		valueBytes := []byte(valueStr)

		keyLen := uint32(len(keyBytes))
		if err := binary.Write(file, binary.LittleEndian, keyLen); err != nil {
			return fmt.Errorf("failed to write key length: %w", err)
		}
		if _, err := file.Write(keyBytes); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}

		valueLen := uint32(len(valueBytes))
		if err := binary.Write(file, binary.LittleEndian, valueLen); err != nil {
			return fmt.Errorf("failed to write value length: %w", err)
		}
		if _, err := file.Write(valueBytes); err != nil {
			return fmt.Errorf("failed to write value: %w", err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync SST file: %w", err)
	}

	return nil
}
