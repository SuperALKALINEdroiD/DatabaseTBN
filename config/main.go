package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SuperALKALINEdroiD/timelyDB/manifest"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
	"github.com/go-playground/validator/v10"
)

func LoadConfig(filePath string) (*DatabaseConfig, error) {
	if filePath == "" {
		return GenerateConfig()
	}

	log.Printf("Loading config at %s", filePath)

	file, err := os.Open(filePath)

	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Config file not found. Generating default config.")
			return GenerateConfig()
		}

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
		manifest, err := manifest.GetManifest()
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
		config.Manifest = *manifest
		return &config, nil
	}

	return nil, fmt.Errorf("invalid config")

}

func (config *DatabaseConfig) validateConfig() bool {
	if len(config.Nodes) == 0 && config.NodeCount > 0 {
		config.Nodes = generateNodeConfig(config.NodeCount, "")
	}
	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			log.Printf("Validation error: Field '%s', Tag: '%s', Param: '%s'\n", err.Field(), err.Tag(), err.Param())
		}

		return false
	}

	if len(config.Nodes) != config.NodeCount {
		log.Printf("Validation error: Node count mismatch. Expected %d nodes, but got %d nodes.\n", config.NodeCount, len(config.Nodes))
		return false
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

func GenerateConfig() (*DatabaseConfig, error) {
	defaultConfigFile := "default-config.json"

	defaultDbPath := common.GetAppPath()
	configData := GenerateExampleConfig(2, "localhost")

	defaultConfigPath := filepath.Join(defaultDbPath, defaultConfigFile)
	os.Setenv("DATABASE_SETTINGS", defaultDbPath)

	data, err := json.Marshal(configData)
	if err != nil {
		fmt.Println("Error marshalling example config:", err)
		return nil, err
	}

	if _, err := os.Stat(defaultDbPath); os.IsNotExist(err) {
		err := os.MkdirAll(defaultDbPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	err = os.WriteFile(defaultConfigPath, data, 0644)
	if err != nil {
		fmt.Println("Error writing config to file:", err)
		return nil, err
	}

	manifest, err := manifest.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	configData.Manifest = *manifest

	return &configData, nil
}
