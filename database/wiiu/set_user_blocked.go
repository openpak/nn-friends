package database_wiiu

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// SetUserBlocked marks a blocked PID as blocked on a blocker PID block list
func SetUserBlocked(blockerPID uint32, blockedPID uint32, titleID uint64, titleVersion uint16) error {
	// M3: block state is canonical in the account core; the local row only
	// carries protocol metadata (title scoping shown in the blacklist UI).
	if err := coregraph.C().Block(context.Background(), "wiiu", blockerPID, blockedPID); err != nil {
		return err
	}
	date := types.NewDateTime(0).Now()

	_, err := database.Manager.Exec(`
	INSERT INTO wiiu.blocks (blocker_pid, blocked_pid, title_id, title_version, date)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (blocker_pid, blocked_pid)
	DO UPDATE SET
	date = $5`, blockerPID, blockedPID, titleID, titleVersion, uint64(date))
	if err != nil {
		return err
	}

	return nil
}
