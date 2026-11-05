package database_wiiu

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/profilecache"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"google.golang.org/grpc/metadata"
)

// ErrProfileDeleted means the adapter reports the PID as a deleted account;
// nothing is cached for it.
var ErrProfileDeleted = errors.New("wiiu: profile deleted")

// EnsureProfiles makes sure every PID has an up-to-date local cache row for
// the name and Mii a Wii U friend list joins on. The account core owns that
// data (read through nn-account's GetUserData); the rows are refreshed from it
// when older than profilecache.TTL or after a profile_updated event. A refresh
// that fails keeps serving the old row; only a PID with no row at all errors.
func EnsureProfiles(pids []uint32) error {
	for _, pid := range pids {
		if profilecache.Fresh(pid) {
			continue
		}
		if err := RefreshProfile(pid); err != nil {
			if hasProfileRow(pid) {
				globals.Logger.Warningf("[Wii U] profile refresh of pid %d failed, serving cache: %v", pid, err)
				profilecache.MarkFailed(pid)
				continue
			}
			return err
		}
	}
	return nil
}

func hasProfileRow(pid uint32) bool {
	row, err := database.Manager.QueryRow(`SELECT 1 FROM wiiu.principal_basic_info WHERE pid = $1`, int64(pid))
	if err != nil {
		return false
	}
	var one int
	return row.Scan(&one) == nil
}

// RefreshProfile re-reads pid from the account core and upserts the cache rows.
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
	now := types.NewDateTime(0)
	now.FromTimestamp(time.Now())
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO wiiu.principal_basic_info (pid, username, unknown) VALUES ($1, $2, 2)
			ON CONFLICT (pid) DO UPDATE SET username = EXCLUDED.username`, []any{pid, user.GetUsername()}},
		{`INSERT INTO wiiu.mii (pid, name, unknown1, unknown2, data, unknown_datetime) VALUES ($1, $2, 0, 0, $3, $4)
			ON CONFLICT (pid) DO UPDATE SET name = EXCLUDED.name, data = EXCLUDED.data
			WHERE wiiu.mii.name IS DISTINCT FROM EXCLUDED.name OR wiiu.mii.data IS DISTINCT FROM EXCLUDED.data`,
			[]any{pid, user.GetMii().GetName(), miiData, uint64(now)}},
		// The birthday is not cached: no friends call reads it.
		{`INSERT INTO wiiu.network_account_info (pid, unknown1, unknown2, birthday) VALUES ($1, 0, 0, 0) ON CONFLICT (pid) DO NOTHING`, []any{pid}},
		{`INSERT INTO wiiu.user_data (pid) VALUES ($1) ON CONFLICT (pid) DO NOTHING`, []any{pid}},
	}
	for _, st := range stmts {
		if _, err := database.Manager.Exec(st.q, st.args...); err != nil {
			return err
		}
	}
	profilecache.Mark(pid)
	return nil
}
