package database_3ds

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_constants "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/constants"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
)

// SaveFriendship saves a friend relationship for a user.
// M3: the canonical pending/accepted state lives in the account core.
// 3DS semantics map as: unresolvable recipient = invalid; one-sided or
// pending = incomplete; mutual/accepted = complete.
func SaveFriendship(senderPID uint32, recipientPID uint32) (friends_3ds_types.FriendRelationship, error) {
	friendRelationship := friends_3ds_types.NewFriendRelationship()
	friendRelationship.PID = types.NewPID(uint64(recipientPID))

	state, err := coregraph.C().Request(context.Background(), "3ds", senderPID, recipientPID)
	if err != nil {
		if err == coregraph.ErrResolutionNotFound {
			friendRelationship.RelationshipType = friends_3ds_constants.RelationshipTypeInvalid
			return friendRelationship, nil
		}
		return friendRelationship, err
	}

	switch state {
	case coregraph.StateComplete:
		friendRelationship.RelationshipType = friends_3ds_constants.RelationshipTypeComplete
	case coregraph.StateIncomplete:
		friendRelationship.RelationshipType = friends_3ds_constants.RelationshipTypeIncomplete
	default:
		friendRelationship.RelationshipType = friends_3ds_constants.RelationshipTypeInvalid
	}
	return friendRelationship, nil
}
