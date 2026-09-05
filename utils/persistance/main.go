package persistance

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"os"
)

type BloomFilter struct {
	Bitset []bool
	K      int
	M      int
}

func BitArraySize(numberOfKeys float64) int {
	const falsePositiveRate float64 = 0.01
	m := math.Max(1024.0, -numberOfKeys*math.Log(falsePositiveRate)/(math.Log(2)*math.Log(2)))
	return int(m)
}

func NewBloomFilter(m, k int) *BloomFilter {
	return &BloomFilter{
		Bitset: make([]bool, m),
		K:      k,
		M:      m,
	}
}

func hashStringWithSeed(s string, seed int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d-%s", seed, s)))
	return int(h.Sum64() % uint64(1<<31-1))
}

func (bf *BloomFilter) Add(key string) {
	if bf == nil || bf.M <= 0 || bf.K <= 0 || len(bf.Bitset) != bf.M {
		return
	}

	for i := 0; i < bf.K; i++ {
		index := hashStringWithSeed(key, i) % bf.M
		bf.Bitset[index] = true
	}
}

func (bf *BloomFilter) MightContain(key string) bool {
	if bf == nil || bf.M <= 0 || bf.K <= 0 || len(bf.Bitset) != bf.M {
		return false
	}

	for i := 0; i < bf.K; i++ {
		index := hashStringWithSeed(key, i) % bf.M
		if !bf.Bitset[index] {
			return false
		}
	}
	return true
}

// Save writes the bloom filter as packed binary:
// [uint32: M][uint32: K][uint32: byteLen][packed bits...]
// Each bool occupies one bit; 8 bools per byte.
func (bf *BloomFilter) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	byteLen := uint32((bf.M + 7) / 8)
	packed := make([]byte, byteLen)
	for i, bit := range bf.Bitset {
		if bit {
			packed[i/8] |= 1 << (uint(i) % 8)
		}
	}

	header := [3]uint32{uint32(bf.M), uint32(bf.K), byteLen}
	if err := binary.Write(file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write bloom filter header: %w", err)
	}
	if _, err := file.Write(packed); err != nil {
		return fmt.Errorf("failed to write bloom filter bitset: %w", err)
	}

	return nil
}

func (bf *BloomFilter) Validate() error {
	if bf == nil {
		return errors.New("nil bloom filter")
	}
	if bf.M <= 0 {
		return errors.New("invalid bloom filter: M must be > 0")
	}
	if bf.K <= 0 {
		return errors.New("invalid bloom filter: K must be > 0")
	}
	if len(bf.Bitset) != bf.M {
		return fmt.Errorf("invalid bloom filter: bitset length (%d) does not match M (%d)", len(bf.Bitset), bf.M)
	}
	return nil
}

// LoadBloomFilter reads a bloom filter written by Save.
// On format errors (e.g. old gob-encoded files), logs a warning and returns nil
// so the caller falls back to a full SSTable scan.
func LoadBloomFilter(path string) (*BloomFilter, error) {
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

	var header [3]uint32
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		log.Printf("Warning: bloom filter at %s has unrecognized format, skipping: %v", path, err)
		return nil, nil
	}

	m, k, byteLen := int(header[0]), int(header[1]), int(header[2])
	if m <= 0 || k <= 0 || byteLen != (m+7)/8 {
		log.Printf("Warning: bloom filter at %s has invalid header (M=%d K=%d byteLen=%d), skipping", path, m, k, byteLen)
		return nil, nil
	}

	packed := make([]byte, byteLen)
	if _, err := file.Read(packed); err != nil {
		log.Printf("Warning: failed to read bloom filter bitset at %s, skipping: %v", path, err)
		return nil, nil
	}

	bitset := make([]bool, m)
	for i := range bitset {
		bitset[i] = packed[i/8]&(1<<(uint(i)%8)) != 0
	}

	bf := &BloomFilter{Bitset: bitset, K: k, M: m}
	if err := bf.Validate(); err != nil {
		log.Printf("Warning: bloom filter validation failed at %s: %v", path, err)
		return nil, nil
	}

	return bf, nil
}
