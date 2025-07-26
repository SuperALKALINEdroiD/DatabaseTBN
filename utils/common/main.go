package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func GetAppPath() (configDir string) {
	configDir, configDirError := os.UserConfigDir()

	if configDirError != nil {
		panic(configDirError)
	}

	return filepath.Join(configDir, "timely")
}

func findDestinationNode() {

}

func logPrefix() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf(
		"[🧠 Alloc: %.1fMB | 🔥 Sys: %.1fMB | 💀 GC: %d | 📦 HeapObjs: %d] ",
		float64(m.Alloc)/1024/1024,
		float64(m.Sys)/1024/1024,
		m.NumGC,
		m.HeapObjects,
	)
}
