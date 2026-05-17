package persistance

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
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

func (bf *BloomFilter) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	defer bufferedWriter.Flush()

	enc := gob.NewEncoder(bufferedWriter)
	err = enc.Encode(bf)
	if err != nil {
		return err
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

	bufferedReader := bufio.NewReader(file)
	dec := gob.NewDecoder(bufferedReader)

	var bf BloomFilter
	err = dec.Decode(&bf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}

	if err := bf.Validate(); err != nil {
		return nil, err
	}

	return &bf, nil
}
