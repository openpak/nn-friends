package database_3ds

import (
	"database/sql"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
)

// GetUserData returns a data for a specific user
func GetUserData(pid types.PID) (friends_3ds_types.FriendPersistentInfo, error) {
	friendPersistentInfo := friends_3ds_types.NewFriendPersistentInfo()

	row, err := database.Manager.QueryRow(`
	SELECT pid, region, area,
		   language, country, favorite_title,
		   favorite_title_version, comment,
		   comment_changed, last_online, mii_changed
	FROM "3ds".user_data WHERE pid=$1
	`, pid)
	if err != nil {
		if err == sql.ErrNoRows {
			return friendPersistentInfo, database.ErrPIDNotFound
		} else {
			return friendPersistentInfo, err
		}
	}

	var region uint8
	var area uint8
	var language uint8
	var country uint8
	var titleID uint64
	var titleVersion uint16
	var message string
	var lastOnlineTime uint64
	var msgUpdateTime uint64
	var miiModifiedAtTime uint64

	err = row.Scan(
		&pid,
		&region,
		&area,
		&language,
		&country,
		&titleID,
		&titleVersion,
		&message,
		&msgUpdateTime,
		&lastOnlineTime,
		&miiModifiedAtTime,
	)
	if err != nil {
		return friendPersistentInfo, err
	}

	friendPersistentInfo.PID = types.NewPID(uint64(pid))
	friendPersistentInfo.Region = types.NewUInt8(region)
	friendPersistentInfo.Country = types.NewUInt8(country)
	friendPersistentInfo.Area = types.NewUInt8(area)
	friendPersistentInfo.Language = types.NewUInt8(language)
	friendPersistentInfo.Platform = types.NewUInt8(2) // * Always 3DS
	friendPersistentInfo.GameKey.TitleID = types.NewUInt64(titleID)
	friendPersistentInfo.GameKey.TitleVersion = types.NewUInt16(titleVersion)
	friendPersistentInfo.Message = types.NewString(message)
	friendPersistentInfo.MessageUpdatedAt = types.NewDateTime(msgUpdateTime)
	friendPersistentInfo.MiiModifiedAt = types.NewDateTime(miiModifiedAtTime)
	friendPersistentInfo.LastOnline = types.NewDateTime(lastOnlineTime)

	return friendPersistentInfo, nil
}
