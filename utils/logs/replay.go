package logs

import (
	"context"
	"log"

	"github.com/SuperALKALINEdroiD/timelyDB/config"
	"github.com/SuperALKALINEdroiD/timelyDB/core"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/common"
	"github.com/SuperALKALINEdroiD/timelyDB/utils/nodes"
)

func ReplayLogs(appConfig *core.App) {
	log.Printf("%s : Replaying logs...", common.LogPrefix())
	appConfig.Config.MetaDataConfig.State = config.NodeStateBuilding
	defer func() {
		appConfig.Config.MetaDataConfig.State = config.NodeStateReady
		log.Printf("%s : Logs replayed...", common.LogPrefix())
	}()

	walEntries, err := appConfig.WAL.ReadLog()
	if err != nil {
		log.Printf("%s : WAL empty or unreadable, skipping replay: %v", common.LogPrefix(), err)
		return
	}

	checkpoints := appConfig.Config.Manifest.NodeCheckpoints // may be nil

	replayed, skipped := 0, 0

	for lineIdx, entry := range walEntries {
		key, value, nodeID, err := ParseWalEntry(entry)
		if err != nil {
			log.Printf("%s : Skipping malformed WAL entry at line %d: %v", common.LogPrefix(), lineIdx, err)
			skipped++
			continue
		}

		// Skip entries that were already flushed to SSTable before the last checkpoint
		if checkpoint, ok := checkpoints[nodeID]; ok && uint64(lineIdx) < checkpoint {
			skipped++
			continue
		}

		client, ok := appConfig.NodeClients[nodeID]
		if !ok {
			log.Printf("%s : No client for nodeID %s, skipping line %d", common.LogPrefix(), nodeID, lineIdx)
			skipped++
			continue
		}

		_, err = client.ManipulateNode(context.Background(), &nodes.NodeManipulationRequest{
			Operation: nodes.Operation_CREATE,
			Key:       key,
			Value:     value,
			Node:      nodeID,
		})
		if err != nil {
			log.Printf("%s : Failed to replay entry for key %s: %v", common.LogPrefix(), key, err)
			skipped++
			continue
		}

		replayed++
	}

	log.Printf("%s : Replay complete — %d replayed, %d skipped", common.LogPrefix(), replayed, skipped)
}
