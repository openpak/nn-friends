package database_3ds

import (
	"database/sql"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
)

// GetUserData returns a data for a specific user
func GetUserData(pid uint32) (friends_3ds_types.FriendPersistentInfo, error) {
	userData := friends_3ds_types.NewFriendPersistentInfo()

	row, err := database.Manager.QueryRow(`
	SELECT pid, region, area,
		   language, country, favorite_title,
		   favorite_title_version, comment,
		   comment_changed, last_online, mii_changed
	FROM "3ds".user_data WHERE pid=$1
	`, pid)
	if err != nil {
		if err == sql.ErrNoRows {
			return userData, database.ErrPIDNotFound
		} else {
			return userData, err
		}
	}

	var relationshipType uint8

	err = row.Scan(&pid, &relationshipType)
	if err != nil {
		return userData, err
	}

	gameKey := friends_3ds_types.NewGameKey()

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
		return userData, err
	}

	gameKey.TitleID = types.NewUInt64(titleID)
	gameKey.TitleVersion = types.NewUInt16(titleVersion)

	userData.PID = types.NewPID(uint64(pid))
	userData.Region = types.NewUInt8(region)
	userData.Country = types.NewUInt8(country)
	userData.Area = types.NewUInt8(area)
	userData.Language = types.NewUInt8(language)
	userData.Platform = types.NewUInt8(2) // * Always 3DS
	userData.GameKey = gameKey
	userData.Message = types.NewString(message)
	userData.MessageUpdatedAt = types.NewDateTime(msgUpdateTime)
	userData.MiiModifiedAt = types.NewDateTime(miiModifiedAtTime)
	userData.LastOnline = types.NewDateTime(lastOnlineTime)

	return userData, nil
}
