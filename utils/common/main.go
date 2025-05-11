package common

import (
	"os"
	"path/filepath"
)

func GetAppPath() (configDir string) {
	configDir, configDirError := os.UserConfigDir()

	if configDirError != nil {
		panic(configDirError)
	}

	return filepath.Join(configDir, "timely")
}
