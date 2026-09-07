package database_wiiu

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
)

// IsFriendRequestBlocked reports whether recipient has blocked sender.
// M3: canonical block state lives in the account core.
func IsFriendRequestBlocked(recipientPID uint32, senderPID uint32) (bool, error) {
	blocked, err := coregraph.C().IsBlocked(context.Background(), "wiiu", recipientPID, senderPID)
	if err != nil {
		if err == coregraph.ErrResolutionNotFound {
			return false, nil
		}
		return false, err
	}
	return blocked, nil
}
