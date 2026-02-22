package database_wiiu

import (
	"github.com/PretendoNetwork/friends/database"
)

// UnsetFriendRequestAccepted unmarks a friend request as accepted
func UnsetFriendRequestAccepted(friendRequestID uint64) error {
	result, err := database.Manager.Exec(`UPDATE wiiu.friend_requests SET accepted=false WHERE id=$1`, friendRequestID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return database.ErrFriendRequestNotFound
	}

	return nil
}