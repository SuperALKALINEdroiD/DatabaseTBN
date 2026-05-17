package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
)

const (
	manifestFileName = "manifest.json"
	sstableDirPrefix = "sstable_"
)

func GetManifest() (*Manifest, error) {
	appPath := common.GetAppPath()
	manifestPath := filepath.Join(appPath, manifestFileName)

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		sstablePath := filepath.Join(appPath, "data", fmt.Sprintf("%s1", sstableDirPrefix))

		err := os.MkdirAll(sstablePath, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("failed to create sstable directory: %v", err)
		}

		initialManifest := &Manifest{
			SSTables: []SSTableMetadata{
				{
					Path:      sstablePath,
					FileName:  "kv.sst",
					BloomFile: "filter.bf",
				},
			},
		}

		kvFilePath := filepath.Join(sstablePath, "kv.sst")
		bloomFilePath := filepath.Join(sstablePath, "filter.bf")

		kvFile, err := os.Create(kvFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSTable file: %v", err)
		}
		defer kvFile.Close()

		bloomFile, err := os.Create(bloomFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create Bloom filter file: %v", err)
		}
		defer bloomFile.Close()

		manifestData, err := json.Marshal(initialManifest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial manifest: %v", err)
		}

		err = os.WriteFile(manifestPath, manifestData, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write manifest file: %v", err)
		}

		return initialManifest, nil
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %v", err)
	}

	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest file: %v", err)
	}

	validationResult := manifest.validateManifest()

	if !validationResult {
		return nil, fmt.Errorf("invalid manifest")
	}

	return &manifest, nil
}

func (manifest Manifest) validateManifest() bool {
	return true
}
