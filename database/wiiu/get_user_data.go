package database_wiiu

import (
	"database/sql"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

// GetUserData returns a data for a specific user
func GetUserData(pid types.PID) (friends_wiiu_types.FriendInfo, error) {
	friendInfo := friends_wiiu_types.NewFriendInfo()

	row, err := database.Manager.QueryRow(`
	SELECT
		u.comment, u.comment_changed,
		u.last_online,
		bi.username, bi.unknown,
		ai.unknown1, ai.unknown2,
		mii.name, mii.unknown1, mii.unknown2, mii.data, mii.unknown_datetime
	FROM wiiu.user_data AS u
	INNER JOIN wiiu.principal_basic_info AS bi ON bi.pid = $1
	INNER JOIN wiiu.network_account_info AS ai ON ai.pid = $1
	INNER JOIN wiiu.mii AS mii ON mii.pid = $1
	WHERE u.pid=$1
	LIMIT 1
	`, pid)

	if err != nil {
		if err == sql.ErrNoRows {
			return friendInfo, database.ErrPIDNotFound
		} else {
			return friendInfo, err
		}
	}
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

	err = row.Scan(&commentContents, &commentChanged, &lastOnlineTime, &nnid, &unknown, &unknown1, &unknown2, &miiName, &miiUnknown1, &miiUnknown2, &miiData, &miiDatetime)
	if err != nil {
		return friendInfo, err
	}

	friendInfo.NNAInfo.Unknown1 = types.NewUInt8(unknown1)
	friendInfo.NNAInfo.Unknown2 = types.NewUInt8(unknown2)
	friendInfo.NNAInfo.PrincipalBasicInfo.PID = types.NewPID(uint64(pid))
	friendInfo.NNAInfo.PrincipalBasicInfo.NNID = types.NewString(nnid)
	friendInfo.NNAInfo.PrincipalBasicInfo.Unknown = types.NewUInt8(unknown)
	friendInfo.NNAInfo.PrincipalBasicInfo.Mii.Name = types.NewString(miiName)
	friendInfo.NNAInfo.PrincipalBasicInfo.Mii.Unknown1 = types.NewUInt8(miiUnknown1)
	friendInfo.NNAInfo.PrincipalBasicInfo.Mii.Unknown2 = types.NewUInt8(miiUnknown2)
	friendInfo.NNAInfo.PrincipalBasicInfo.Mii.MiiData = types.NewBuffer(miiData)
	friendInfo.NNAInfo.PrincipalBasicInfo.Mii.Datetime = types.NewDateTime(miiDatetime)

	lastOnline := types.NewDateTime(0).Now()
	connectedUser, ok := globals.ConnectedUsers.Get(uint32(pid))
	if ok && connectedUser != nil {
		// * Online
		friendInfo.Presence = connectedUser.PresenceV2.Copy().(friends_wiiu_types.NintendoPresenceV2)
	} else {
		// * Offline
		lastOnline = types.NewDateTime(lastOnlineTime) // TODO - Change this
	}

	friendInfo.Status.Unknown = types.NewUInt8(0)
	friendInfo.Status.Contents = types.NewString(commentContents)
	friendInfo.Status.LastChanged = types.NewDateTime(commentChanged)
	friendInfo.BecameFriend = types.NewDateTime(date)
	friendInfo.LastOnline = lastOnline
	friendInfo.Unknown = types.NewUInt64(0)

	return friendInfo, nil
}
