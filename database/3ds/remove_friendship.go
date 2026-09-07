package database_3ds

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
)

// RemoveFriendship removes a user's friend relationship.
// M3: canonical state lives in the account core.
func RemoveFriendship(user1_pid uint32, user2_pid uint32) error {
	return coregraph.C().Remove(context.Background(), "3ds", user1_pid, user2_pid)
}
