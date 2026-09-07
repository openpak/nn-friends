package database_3ds

import (
	"context"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_constants "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/constants"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
)

// GetUserFriends returns all friend relationships of a user.
// M3: canonical state lives in the account core. The 3DS model is:
// complete = core friends; incomplete = one-sided outgoing pending
// requests (the console shows those as half-open friendships).
func GetUserFriends(pid uint32) (types.List[friends_3ds_types.FriendRelationship], error) {
	friendRelationships := types.NewList[friends_3ds_types.FriendRelationship]()

	// Complete friendships.
	friendPIDs, err := coregraph.C().FriendPIDs(context.Background(), "3ds", pid)
	if err != nil && err != coregraph.ErrResolutionNotFound {
		return friendRelationships, err
	}
	for _, friendPID := range friendPIDs {
		relationship := friends_3ds_types.NewFriendRelationship()
		relationship.LFC = types.NewUInt64(0)
		relationship.PID = types.NewPID(uint64(friendPID))
		relationship.RelationshipType = friends_3ds_constants.RelationshipTypeComplete
		friendRelationships = append(friendRelationships, relationship)
	}

	// One-sided outgoing requests.
	outgoingPIDs, err := coregraph.C().OutgoingAddresseePIDs(context.Background(), "3ds", pid)
	if err != nil && err != coregraph.ErrResolutionNotFound {
		return friendRelationships, err
	}
	for _, otherPID := range outgoingPIDs {
		relationship := friends_3ds_types.NewFriendRelationship()
		relationship.LFC = types.NewUInt64(0)
		relationship.PID = types.NewPID(uint64(otherPID))
		relationship.RelationshipType = friends_3ds_constants.RelationshipTypeIncomplete
		friendRelationships = append(friendRelationships, relationship)
	}

	if len(friendRelationships) == 0 {
		return friendRelationships, database.ErrEmptyList
	}
	return friendRelationships, nil
}
