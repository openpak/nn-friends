package database_wiiu

import (
	"context"
	"database/sql"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

// AcceptFriendRequestAndReturnFriendInfo accepts the given friend reuqest and returns the friend's information
func AcceptFriendRequestAndReturnFriendInfo(friendRequestID uint64) (friends_wiiu_types.FriendInfo, error) {
	var senderPID uint32
	var recipientPID uint32

	row, err := database.Manager.QueryRow(`SELECT sender_pid, recipient_pid FROM wiiu.friend_requests WHERE id=$1`, friendRequestID)
	if err != nil {
		return friends_wiiu_types.NewFriendInfo(), err
	}

	err = row.Scan(&senderPID, &recipientPID)
	if err != nil {
		if err == sql.ErrNoRows {
			return friends_wiiu_types.NewFriendInfo(), database.ErrFriendRequestNotFound
		} else {
			return friends_wiiu_types.NewFriendInfo(), err
		}
	}

	acceptedTime := types.NewDateTime(0).Now()

	// M3: the canonical friendship state lives in the account core; the
	// local friendships table is no longer written (no competing graph
	// store). Accepting is a core operation; the local request row keeps
	// only protocol metadata.
	if err := coregraph.C().AcceptRequest(context.Background(), "wiiu", recipientPID, senderPID); err != nil {
		return friends_wiiu_types.NewFriendInfo(), err
	}

	err = SetFriendRequestAccepted(friendRequestID)
	if err != nil {
		return friends_wiiu_types.NewFriendInfo(), err
	}

	friendInfo := friends_wiiu_types.NewFriendInfo()
	connectedUser, ok := globals.ConnectedUsers.Get(senderPID)
	friendInfo.NNAInfo, err = GetUserNetworkAccountInfo(senderPID)
	if err != nil {
		return friends_wiiu_types.NewFriendInfo(), err
	}
	lastOnline := types.NewDateTime(0).Now()

	if ok && connectedUser != nil {
		// * Online
		friendInfo.Presence = connectedUser.PresenceV2.Copy().(friends_wiiu_types.NintendoPresenceV2)
	} else {
		// * Offline
		var lastOnlineTime uint64
		row, err = database.Manager.QueryRow(`SELECT last_online FROM wiiu.user_data WHERE pid=$1`, senderPID)
		if err != nil {
			return friends_wiiu_types.NewFriendInfo(), err
		}

		err = row.Scan(&lastOnlineTime)
		if err != nil {
			if err == sql.ErrNoRows {
				return friends_wiiu_types.NewFriendInfo(), database.ErrPIDNotFound
			} else {
				return friends_wiiu_types.NewFriendInfo(), err
			}
		}

		lastOnline = types.NewDateTime(lastOnlineTime) // TODO - Change this
	}

	status, err := GetUserComment(senderPID)
	if err != nil {
		return friends_wiiu_types.NewFriendInfo(), err
	}

	friendInfo.Status = status
	friendInfo.BecameFriend = acceptedTime
	friendInfo.LastOnline = lastOnline // TODO - Change this
	friendInfo.Unknown = types.NewUInt64(0)

	return friendInfo, nil
}
