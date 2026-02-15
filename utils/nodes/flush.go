package nodes

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/persistance"
)

type kvPair struct {
	key   string
	value string
}

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
	if len(manifest.SSTables) == 0 {
		return fmt.Errorf("no SSTable paths configured in manifest")
	}

	basePath := manifest.SSTables[0].Path // temporary

	// Ensure the directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create SSTable directory: %w", err)
	}

	sstableFile := manifest.SSTables[0].FileName
	if sstableFile == "" {
		sstableFile = "kv.sst"
	}
	bloomFile := manifest.SSTables[0].BloomFile
	if bloomFile == "" {
		bloomFile = "filter.bf"
	}

	tmpSST := filepath.Join(basePath, "kv.tmp.sst")
	tmpBloom := filepath.Join(basePath, "filter.tmp.bf")
	finalSST := filepath.Join(basePath, sstableFile)
	finalBloom := filepath.Join(basePath, bloomFile)

	mergedKV, err := mergeExistingAndIncomingKV(finalSST, kvData)
	if err != nil {
		return fmt.Errorf("failed to merge incoming data with existing SSTable: %w", err)
	}

	pairs := normalizeAndSortPairs(mergedKV)
	if len(pairs) == 0 {
		return fmt.Errorf("no serializable key-value pairs to flush")
	}

	if err := server.writeToSSt(pairs, tmpSST); err != nil {
		return fmt.Errorf("SST write failed: %w", err)
	}

	if err := server.writeToBloom(pairs, tmpBloom); err != nil {
		os.Remove(tmpSST)
		return fmt.Errorf("BLOOM WRITE FAILED: %w", err)
	}

	if err := os.Rename(tmpSST, finalSST); err != nil {
		os.Remove(tmpSST)
		os.Remove(tmpBloom)
		return fmt.Errorf("SST rename failed: %w", err)
	}
	if err := os.Rename(tmpBloom, finalBloom); err != nil {
		os.Remove(finalSST)
		return fmt.Errorf("bloom filter rename failed: %w", err)
	}

	if err := validatePersistedArtifacts(finalSST, finalBloom, pairs); err != nil {
		return fmt.Errorf("persisted file validation failed: %w", err)
	}

	log.Printf("Successfully flushed %d entries to SSTable at %s", len(kvData), finalSST)
	return nil
}

func mergeExistingAndIncomingKV(sstablePath string, incoming map[any]any) (map[any]any, error) {
	merged := make(map[any]any)

	existingPairs, err := readSSTable(sstablePath)
	if err != nil {
		return nil, err
	}
	for _, pair := range existingPairs {
		merged[pair.key] = pair.value
	}

	for key, value := range incoming {
		merged[key] = value
	}

	return merged, nil
}

func normalizeAndSortPairs(kvData map[any]any) []kvPair {
	pairs := make([]kvPair, 0, len(kvData))
	for key, value := range kvData {
		keyStr, keyOk := key.(string)
		valueStr, valueOk := value.(string)
		if !keyOk || !valueOk {
			log.Printf("Warning: skipping non-string key-value pair")
			continue
		}
		pairs = append(pairs, kvPair{key: keyStr, value: valueStr})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	return pairs
}

func (server *internalNode) writeToBloom(pairs []kvPair, path string) error {
	if len(pairs) == 0 {
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for bloom filter: %w", err)
	}

	filerSize := persistance.BitArraySize(float64(len(pairs)))
	bf := persistance.NewBloomFilter(filerSize, 7)

	// Add all keys to bloom filter
	for _, pair := range pairs {
		bf.Add(pair.key)
	}

	if err := bf.Save(path); err != nil {
		return fmt.Errorf("failed to save bloom filter: %w", err)
	}

	log.Printf("Wrote bloom filter with %d keys to: %s", len(pairs), path)
	return nil
}

func (server *internalNode) writeToSSt(pairs []kvPair, path string) error {
	if len(pairs) == 0 {
		log.Println("No data to write to SSTable, skipping")
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for SST file: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create SST file: %w", err)
	}
	defer file.Close()

	// Write number of entries
	numEntries := uint64(len(pairs))
	if err := binary.Write(file, binary.LittleEndian, numEntries); err != nil {
		return fmt.Errorf("failed to write entry count: %w", err)
	}

	// Write each key-value pair in sorted order
	for _, pair := range pairs {
		keyBytes := []byte(pair.key)
		valueBytes := []byte(pair.value)

		// Write key length and key
		keyLen := uint32(len(keyBytes))
		if err := binary.Write(file, binary.LittleEndian, keyLen); err != nil {
			return fmt.Errorf("failed to write key length: %w", err)
		}
		if _, err := file.Write(keyBytes); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}

		// Write value length and value
		valueLen := uint32(len(valueBytes))
		if err := binary.Write(file, binary.LittleEndian, valueLen); err != nil {
			return fmt.Errorf("failed to write value length: %w", err)
		}
		if _, err := file.Write(valueBytes); err != nil {
			return fmt.Errorf("failed to write value: %w", err)
		}
	}

	// Sync to ensure data is written to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync SST file: %w", err)
	}

	log.Printf("Wrote %d entries to SSTable file: %s", len(pairs), path)
	return nil
}

func readSSTable(path string) ([]kvPair, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if fileInfo.Size() == 0 {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var numEntries uint64
	if err := binary.Read(file, binary.LittleEndian, &numEntries); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read entry count: %w", err)
	}

	pairs := make([]kvPair, 0, numEntries)
	for i := uint64(0); i < numEntries; i++ {
		var keyLen uint32
		if err := binary.Read(file, binary.LittleEndian, &keyLen); err != nil {
			return nil, fmt.Errorf("failed to read key length at entry %d: %w", i, err)
		}

		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(file, keyBytes); err != nil {
			return nil, fmt.Errorf("truncated key data at entry %d: %w", i, err)
		}

		var valueLen uint32
		if err := binary.Read(file, binary.LittleEndian, &valueLen); err != nil {
			return nil, fmt.Errorf("failed to read value length at entry %d: %w", i, err)
		}

		valueBytes := make([]byte, valueLen)
		if _, err := io.ReadFull(file, valueBytes); err != nil {
			return nil, fmt.Errorf("truncated value data at entry %d: %w", i, err)
		}

		pairs = append(pairs, kvPair{key: string(keyBytes), value: string(valueBytes)})
	}

	return pairs, nil
}

func validatePersistedArtifacts(sstablePath string, bloomPath string, expected []kvPair) error {
	storedPairs, err := readSSTable(sstablePath)
	if err != nil {
		return err
	}
	if len(storedPairs) != len(expected) {
		return fmt.Errorf("SSTable validation mismatch: expected %d entries, got %d", len(expected), len(storedPairs))
	}

	for i := range expected {
		if expected[i].key != storedPairs[i].key || expected[i].value != storedPairs[i].value {
			return fmt.Errorf("SSTable validation mismatch at index %d", i)
		}
	}

	bf, err := persistance.LoadBloomFilter(bloomPath)
	if err != nil {
		return err
	}
	if bf == nil {
		return fmt.Errorf("bloom filter missing or empty at %s", bloomPath)
	}

	for _, pair := range expected {
		if !bf.MightContain(pair.key) {
			return fmt.Errorf("bloom filter missing persisted key: %s", pair.key)
		}
	}

	return nil
}

func (server *internalNode) lookupFromDisk(key string) (string, bool, error) {
	manifest := server.dbConfig.Manifest
	if len(manifest.SSTables) == 0 {
		return "", false, nil
	}

	basePath := manifest.SSTables[0].Path
	sstableFile := manifest.SSTables[0].FileName
	if sstableFile == "" {
		sstableFile = "kv.sst"
	}
	bloomFile := manifest.SSTables[0].BloomFile
	if bloomFile == "" {
		bloomFile = "filter.bf"
	}

	sstablePath := filepath.Join(basePath, sstableFile)
	bloomPath := filepath.Join(basePath, bloomFile)

	bf, err := persistance.LoadBloomFilter(bloomPath)
	if err != nil {
		return "", false, err
	}
	if bf != nil && !bf.MightContain(key) {
		return "", false, nil
	}

	pairs, err := readSSTable(sstablePath)
	if err != nil {
		return "", false, err
	}
	if len(pairs) == 0 {
		return "", false, nil
	}

	idx := sort.Search(len(pairs), func(i int) bool {
		return pairs[i].key >= key
	})
	if idx < len(pairs) && pairs[idx].key == key {
		return pairs[idx].value, true, nil
	}

	return "", false, nil
}
