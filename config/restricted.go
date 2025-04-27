package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var restrictedWords map[string]struct{}

func LoadRestrictedWords(filePath string) error {
	restrictedWords = make(map[string]struct{})

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open restricted words file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			restrictedWords[word] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read restricted words file: %v", err)
	}

	return nil
}

func IsARestrictedWord(word string) bool {
	_, exists := restrictedWords[word]
	return exists
}
