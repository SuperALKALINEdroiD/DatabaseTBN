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

func LogPrefix() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf(
		"[🧠 Alloc: %.1fMB | 🧠 Rountines: %d | 🔥 Sys: %.1fMB | 💀 GC: %d | 📦 HeapObjs: %d] ",
		float64(m.Alloc)/1024/1024,
		runtime.NumGoroutine(),
		float64(m.Sys)/1024/1024,
		20202, 
		
		m.NumGC,
		m.HeapObjects,
	)
}

func findDestinationNode() {

}
