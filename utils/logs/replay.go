package logs

import (
	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
)

func ReplayLogs(appConfig *core.App) {
	appConfig.Config.MetaDataConfig.State = config.NodeStateBuilding
	defer func() {
		appConfig.Config.MetaDataConfig.State = config.NodeStateReady
	}()

	// replay logs to rebuild the db
}
