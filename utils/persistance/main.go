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
	for i := 0; i < bf.K; i++ {
		index := hashStringWithSeed(key, i) % bf.M
		bf.Bitset[index] = true
	}
}

func (bf *BloomFilter) MightContain(key string) bool {
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
