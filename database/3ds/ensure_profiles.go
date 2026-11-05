package database_3ds

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/profilecache"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"google.golang.org/grpc/metadata"
)

// ErrProfileDeleted means the adapter reports the PID as a deleted account;
// nothing is cached for it.
var ErrProfileDeleted = errors.New("3ds: profile deleted")

// EnsureProfiles keeps the Mii cache of every PID current. The account core
// owns the Mii (read through nn-account's GetUserData); the user_data mii
// columns are refreshed when older than profilecache.TTL or after a
// profile_updated event. A failed refresh keeps serving the old row.
func EnsureProfiles(pids []uint32) error {
	for _, pid := range pids {
		if profilecache.Fresh(pid) {
			continue
		}
		if err := RefreshProfile(pid); err != nil {
			if hasProfileRow(pid) {
				globals.Logger.Warningf("[3DS] profile refresh of pid %d failed, serving cache: %v", pid, err)
				profilecache.MarkFailed(pid)
				continue
			}
			return err
		}
	}
	return nil
}

func hasProfileRow(pid uint32) bool {
	row, err := database.Manager.QueryRow(`SELECT 1 FROM "3ds".user_data WHERE pid = $1`, int64(pid))
	if err != nil {
		return false
	}
	var one int
	return row.Scan(&one) == nil
}

// RefreshProfile re-reads pid from the account core and upserts its Mii cache.
func RefreshProfile(pid uint32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, common_globals.GRPCAccountCommonMetadata)
	user, err := common_globals.GRPCAccountClient.GetUserData(ctx, &pb.GetUserDataRequest{Pid: pid})
	if err != nil {
		return err
	}
	if user.GetDeleted() {
		return ErrProfileDeleted
	}
	miiData, _ := base64.StdEncoding.DecodeString(user.GetMii().GetData())
	if _, err := database.Manager.Exec(`INSERT INTO "3ds".user_data (pid, mii_name, mii_data) VALUES ($1, $2, $3)
		ON CONFLICT (pid) DO UPDATE SET mii_name = EXCLUDED.mii_name, mii_data = EXCLUDED.mii_data
		WHERE "3ds".user_data.mii_name IS DISTINCT FROM EXCLUDED.mii_name OR "3ds".user_data.mii_data IS DISTINCT FROM EXCLUDED.mii_data`,
		pid, user.GetMii().GetName(), miiData); err != nil {
		return err
	}
	profilecache.Mark(pid)
	return nil
}
