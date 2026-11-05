package database_3ds

import (
	"context"
	"errors"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/profilecache"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
)

// UpdateUserMii stores a Mii a 3DS uploaded. The Mii belongs to the account, so
// it goes to the account core first (shared with the Wii U, Wii and Switch); the
// local row is the read cache and is written only after the core took it. A PID
// with no core account (nothing to attach a Mii to) keeps the local row alone.
func UpdateUserMii(pid uint32, mii friends_3ds_types.Mii) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	accountID, err := coregraph.C().AccountOfPID(ctx, "3ds", pid)
	switch {
	case err == nil:
		if _, err := coregraph.C().PutMii(ctx, accountID, []byte(mii.MiiData)); err != nil {
			return err
		}
	case errors.Is(err, coregraph.ErrResolutionNotFound):
		// no account: local only
	default:
		return err
	}
	_, err = database.Manager.Exec(`
		INSERT INTO "3ds".user_data (pid, mii_name, mii_profanity, mii_character_set, mii_data, mii_changed)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pid)
		DO UPDATE SET 
		mii_name = $2,
		mii_profanity = $3,
		mii_character_set = $4,
		mii_data = $5,
		mii_changed = $6`, pid, string(mii.Name), bool(mii.ProfanityFlag), uint8(mii.CharacterSet), []byte(mii.MiiData), uint64(types.NewDateTime(0).Now()))
	if err != nil {
		return err
	}
	profilecache.Mark(pid)
	return nil
}
