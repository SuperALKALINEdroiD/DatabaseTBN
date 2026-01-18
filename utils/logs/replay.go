package logs

import (
	"log"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
)

func ReplayLogs(appConfig *core.App) {
	log.Printf("%s : Replaying logs...", common.LogPrefix())
	appConfig.Config.MetaDataConfig.State = config.NodeStateBuilding
	defer func() {
		appConfig.Config.MetaDataConfig.State = config.NodeStateReady
		log.Printf("%s : Logs replayed...", common.LogPrefix())
	}()

	logs, err := appConfig.WAL.ReadLog()
	if err != nil {
		panic(err)
	}

	// var wg sync.WaitGroup

	for _, log := range logs {
		_, _, _, err := ParseWalEntry(log)
		if err != nil {
			panic(err)
		}
	}
}
