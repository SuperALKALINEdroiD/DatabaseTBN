package persistance

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"math"
	"os"
)

type BloomFilter struct {
	bitset []bool
	k      int
	m      int
}

func BitArraySize(numberOfKeys float64) int {
	const falsePositiveRate float64 = 0.01
	m := int(-numberOfKeys * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2)))
	if m < 1024 {
		m = 1024
	}
	return m
}

func NewBloomFilter(m, k int) *BloomFilter {
	return &BloomFilter{
		bitset: make([]bool, m),
		k:      k,
		m:      m,
	}
}

func hashStringWithSeed(s string, seed int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d-%s", seed, s)))
	return int(h.Sum64() % uint64(1<<31-1))
}

func (bf *BloomFilter) Add(key string) {
	for i := 0; i < bf.k; i++ {
		index := hashStringWithSeed(key, i) % bf.m
		bf.bitset[index] = true
	}
}

func (bf *BloomFilter) MightContain(key string) bool {
	for i := 0; i < bf.k; i++ {
		index := hashStringWithSeed(key, i) % bf.m
		if !bf.bitset[index] {
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

func LoadBloomFilter(path string) (*BloomFilter, error) {
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
		return nil, err
	}

	return &bf, nil
}
