package nodes

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/SuperALKALINEdroiD/timelyDB/manifest"
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
	m := &server.dbConfig.Manifest
	if len(m.SSTables) == 0 {
		return fmt.Errorf("no SSTable paths configured in manifest")
	}

	basePath := m.SSTables[0].Path
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create SSTable directory: %w", err)
	}

	pairs := normalizeAndSortPairs(kvData)
	if len(pairs) == 0 {
		return fmt.Errorf("no serializable key-value pairs to flush")
	}

	// Assign the next segment ID
	m.NextSegmentID++
	segID := m.NextSegmentID
	sstFile := fmt.Sprintf("seg_%08d.sst", segID)
	bloomFile := fmt.Sprintf("seg_%08d.bf", segID)
	tmpSST := filepath.Join(basePath, fmt.Sprintf("seg_%08d.tmp.sst", segID))
	tmpBloom := filepath.Join(basePath, fmt.Sprintf("seg_%08d.tmp.bf", segID))
	finalSST := filepath.Join(basePath, sstFile)
	finalBloom := filepath.Join(basePath, bloomFile)

	if err := server.writeToSSt(pairs, tmpSST); err != nil {
		m.NextSegmentID-- // roll back on failure
		return fmt.Errorf("SST write failed: %w", err)
	}

	if err := server.writeToBloom(pairs, tmpBloom); err != nil {
		m.NextSegmentID--
		os.Remove(tmpSST)
		return fmt.Errorf("bloom write failed: %w", err)
	}

	if err := os.Rename(tmpSST, finalSST); err != nil {
		m.NextSegmentID--
		os.Remove(tmpSST)
		os.Remove(tmpBloom)
		return fmt.Errorf("SST rename failed: %w", err)
	}
	if err := os.Rename(tmpBloom, finalBloom); err != nil {
		m.NextSegmentID--
		os.Remove(finalSST)
		return fmt.Errorf("bloom filter rename failed: %w", err)
	}

	// Register the new segment in the manifest
	m.SSTables = append(m.SSTables, manifest.SSTableMetadata{
		Path:      basePath,
		FileName:  sstFile,
		BloomFile: bloomFile,
		SegmentID: segID,
	})

	// Save WAL checkpoint: record line count so replay skips these entries on next startup
	if server.wal != nil {
		if err := server.wal.Flush(); err != nil {
			log.Printf("Warning: WAL flush before checkpoint failed: %v", err)
		} else if lineCount, err := server.wal.GetTotalLines(); err == nil {
			if m.NodeCheckpoints == nil {
				m.NodeCheckpoints = make(map[string]uint64)
			}
			m.NodeCheckpoints[server.nodeID] = uint64(lineCount)
		}
	}

	if err := manifest.SaveManifest(m); err != nil {
		log.Printf("Warning: failed to save manifest after flush: %v", err)
	}

	log.Printf("Successfully flushed %d entries to segment %d at %s", len(pairs), segID, finalSST)

	server.compactSSTables()

	return nil
}

// TODO: implement full compaction — merge segments into one sorted file, deduplicate keys, remove old files.
func (server *internalNode) compactSSTables() {
	const compactionThreshold = 10
	if len(server.dbConfig.Manifest.SSTables) > compactionThreshold {
		log.Printf("Compaction threshold (%d segments) reached — compaction not yet implemented", compactionThreshold)
	}
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

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for bloom filter: %w", err)
	}

	filerSize := persistance.BitArraySize(float64(len(pairs)))
	bf := persistance.NewBloomFilter(filerSize, 7)

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

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for SST file: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create SST file: %w", err)
	}
	defer file.Close()

	numEntries := uint64(len(pairs))
	if err := binary.Write(file, binary.LittleEndian, numEntries); err != nil {
		return fmt.Errorf("failed to write entry count: %w", err)
	}

	for _, pair := range pairs {
		keyBytes := []byte(pair.key)
		valueBytes := []byte(pair.value)

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

// lookupFromDisk searches all segments newest-first (most recent write wins).
func (server *internalNode) lookupFromDisk(key string) (string, bool, error) {
	m := server.dbConfig.Manifest
	if len(m.SSTables) == 0 {
		return "", false, nil
	}

	for i := len(m.SSTables) - 1; i >= 0; i-- {
		seg := m.SSTables[i]
		sstablePath := filepath.Join(seg.Path, seg.FileName)
		bloomPath := filepath.Join(seg.Path, seg.BloomFile)

		bf, err := persistance.LoadBloomFilter(bloomPath)
		if err != nil {
			log.Printf("Warning: failed to load bloom filter for segment %d: %v", seg.SegmentID, err)
		}
		if bf != nil && !bf.MightContain(key) {
			continue // definitely not in this segment
		}

		pairs, err := readSSTable(sstablePath)
		if err != nil {
			return "", false, fmt.Errorf("error reading segment %d: %w", seg.SegmentID, err)
		}
		if len(pairs) == 0 {
			continue
		}

		idx := sort.Search(len(pairs), func(j int) bool {
			return pairs[j].key >= key
		})
		if idx < len(pairs) && pairs[idx].key == key {
			return pairs[idx].value, true, nil
		}
	}

	return "", false, nil
}
