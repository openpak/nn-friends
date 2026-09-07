package database_wiiu

import (
	"context"
	"errors"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
)

// GetUserFriendPIDs returns a user's friend PIDs list.
// M3: canonical state lives in the account core; this is a projection.
func GetUserFriendPIDs(pid uint32) ([]uint32, error) {
	pids := make([]uint32, 0)

	pids2, err := coregraph.C().FriendPIDs(context.Background(), "wiiu", pid)
	if err != nil {
		if errors.Is(err, coregraph.ErrResolutionNotFound) {
			return pids, database.ErrEmptyList
		}
		return pids, err
	}
	if len(pids2) == 0 {
		return pids, database.ErrEmptyList
	}
	return pids2, nil
}
