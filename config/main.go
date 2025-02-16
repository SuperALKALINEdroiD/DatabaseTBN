package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func LoadConfig(filePath string) (*DatabaseConfig, error) {
	if filePath == "" {
		config, error := GenerateConfig("default-config.json")

		if error != nil {
			return nil, fmt.Errorf("failed to create config file: %v", error)
		}

		return config, nil
	}

	file, err := os.Open(filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}

	defer file.Close()

	var config DatabaseConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if config.validateConfig() {
		return &config, nil
	}

	return nil, fmt.Errorf("invalid config")

}

func (config *DatabaseConfig) validateConfig() bool {
	if len(config.Nodes) == 0 {
		config.Nodes = generateNodeConfig(config.NodeCount, "")
	}

	return true
}

func (c *DatabaseConfig) SaveConfig(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(c)

	if err != nil {
		return fmt.Errorf("failed to save Config File, Error on Encoder: %v", err)
	}

	return nil
}

func GenerateConfig(filePath string) (*DatabaseConfig, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	exampleConfig := GenerateExampleConfig(2, "") // if no config, start with 1 node on local machine

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "	 ")
	if err := encoder.Encode(exampleConfig); err != nil {
		return nil, fmt.Errorf("failed to save config file, error on encoder: %v", err)
	}

	log.Println("Configuration file created successfully at:", filePath)
	return &exampleConfig, nil
}
