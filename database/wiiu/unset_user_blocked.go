package database_wiiu

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
)

// UnsetUserBlocked removes a block from a user.
// M3: canonical block state lives in the account core.
func UnsetUserBlocked(user1_pid uint32, user2_pid uint32) error {
	// Core first: blocks are enforced from there. The local row is
	// protocol metadata and is deleted after.
	if err := coregraph.C().Unblock(context.Background(), "wiiu", user1_pid, user2_pid); err != nil {
		return err
	}
	result, err := database.Manager.Exec(`
		DELETE FROM wiiu.blocks WHERE blocker_pid=$1 AND blocked_pid=$2`, user1_pid, user2_pid)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return database.ErrPIDNotFound
	}

	return nil
}
