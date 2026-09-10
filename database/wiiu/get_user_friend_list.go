package database_wiiu

import (
	"context"

	"database/sql"
	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/lib/pq"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

// GetUserFriendList returns a user's friend list
// GetUserFriendList returns the user's friend list. The core decides who the
// friends are; the local tables only carry what a console told us about them,
// and EnsureProfiles fills in anyone it never met.
func GetUserFriendList(pid uint32) (types.List[friends_wiiu_types.FriendInfo], error) {
	friendPIDs, err := coregraph.C().FriendPIDs(context.Background(), "wiiu", pid)
	if err != nil {
		return types.NewList[friends_wiiu_types.FriendInfo](), err
	}
	return FriendInfosForPIDs(pid, friendPIDs)
}

// FriendInfosForPIDs builds FriendInfo entries as seen by viewerPID.
func FriendInfosForPIDs(viewerPID uint32, pids []uint32) (types.List[friends_wiiu_types.FriendInfo], error) {
	friendList := types.NewList[friends_wiiu_types.FriendInfo]()
	if len(pids) == 0 {
		return friendList, database.ErrEmptyList
	}
	if err := EnsureProfiles(pids); err != nil {
		return friendList, err
	}
	ids := make([]int64, 0, len(pids))
	for _, p := range pids {
		ids = append(ids, int64(p))
	}

	rows, err := database.Manager.Query(`
	SELECT
		bi.pid,
		COALESCE((SELECT f.date FROM wiiu.friendships AS f WHERE f.user1_pid=$1 AND f.user2_pid=bi.pid AND f.active LIMIT 1),
		         (SELECT fr.sent_on FROM wiiu.friend_requests AS fr WHERE fr.accepted AND ((fr.sender_pid=$1 AND fr.recipient_pid=bi.pid) OR (fr.sender_pid=bi.pid AND fr.recipient_pid=$1)) LIMIT 1),
		         0),
		COALESCE(u.comment, ''), COALESCE(u.comment_changed, 0),
		COALESCE(u.last_online, 0),
		bi.username, bi.unknown,
		COALESCE(ai.unknown1, 0), COALESCE(ai.unknown2, 0),
		COALESCE(mii.name, ''), COALESCE(mii.unknown1, 0), COALESCE(mii.unknown2, 0), COALESCE(mii.data, ''), COALESCE(mii.unknown_datetime, 0)
	FROM wiiu.principal_basic_info AS bi
	LEFT JOIN wiiu.user_data AS u ON u.pid = bi.pid
	LEFT JOIN wiiu.network_account_info AS ai ON ai.pid = bi.pid
	LEFT JOIN wiiu.mii AS mii ON mii.pid = bi.pid
	WHERE bi.pid = ANY($2::int[])
	LIMIT 100
	`, viewerPID, pq.Array(ids))

	if err != nil {
		if err == sql.ErrNoRows {
			return friendList, database.ErrEmptyList
		} else {
			return friendList, err
		}
	}
	defer rows.Close()
	for rows.Next() {
		var friendPID uint32
		var date uint64
		var lastOnlineTime uint64
		var commentContents string
		var commentChanged uint64 = 0
		var nnid string
		var unknown uint8
		var unknown1 uint8
		var unknown2 uint8
		var miiName string
		var miiUnknown1 uint8
		var miiUnknown2 uint8
		var miiData []byte
		var miiDatetime uint64

		err := rows.Scan(&friendPID, &date, &commentContents, &commentChanged, &lastOnlineTime, &nnid, &unknown, &unknown1, &unknown2, &miiName, &miiUnknown1, &miiUnknown2, &miiData, &miiDatetime)
		if err != nil {
			return nil, err
		}

		comment := friends_wiiu_types.NewComment()
		comment.Unknown = types.NewUInt8(0)
		comment.Contents = types.NewString(commentContents)
		comment.LastChanged = types.NewDateTime(commentChanged)

		mii := friends_wiiu_types.NewMiiV2()
		mii.Name = types.NewString(miiName)
		mii.Unknown1 = types.NewUInt8(miiUnknown1)
		mii.Unknown2 = types.NewUInt8(miiUnknown2)
		mii.MiiData = types.NewBuffer(miiData)
		mii.Datetime = types.NewDateTime(miiDatetime)

		principalBasicInfo := friends_wiiu_types.NewPrincipalBasicInfo()
		principalBasicInfo.PID = types.NewPID(uint64(friendPID))
		principalBasicInfo.NNID = types.NewString(nnid)
		principalBasicInfo.Unknown = types.NewUInt8(unknown)
		principalBasicInfo.Mii = mii

		nnaInfo := friends_wiiu_types.NewNNAInfo()
		nnaInfo.Unknown1 = types.NewUInt8(unknown1)
		nnaInfo.Unknown2 = types.NewUInt8(unknown2)
		nnaInfo.PrincipalBasicInfo = principalBasicInfo

		friendInfo := friends_wiiu_types.NewFriendInfo()
		friendInfo.NNAInfo = nnaInfo

		lastOnline := types.NewDateTime(0).Now()
		connectedUser, ok := globals.ConnectedUsers.Get(friendPID)
		if ok && connectedUser != nil {
			// * Online
			friendInfo.Presence = connectedUser.PresenceV2.Copy().(friends_wiiu_types.NintendoPresenceV2)
		} else {
			// * Offline
			lastOnline = types.NewDateTime(lastOnlineTime) // TODO - Change this
		}

		friendInfo.Status = comment
		if date == 0 {
			// ponytail: the core keeps no acceptance time; first sight is the date shown.
			date = uint64(types.NewDateTime(0).Now())
		}
		friendInfo.BecameFriend = types.NewDateTime(date)
		friendInfo.LastOnline = lastOnline
		friendInfo.Unknown = types.NewUInt64(0)

		friendList = append(friendList, friendInfo)
	}

	return friendList, nil
}
