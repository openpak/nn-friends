package database_wiiu

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/PretendoNetwork/friends/database"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
	"google.golang.org/grpc/metadata"
)

// EnsureProfiles gives every PID the local rows a Wii U friend list joins on.
// A PID a Wii U console has never signed in with (a Switch or phone user behind
// a shadow PNID) gets its name and Mii from the adapter's GetUserData; rows a
// console wrote itself are left alone.
func EnsureProfiles(pids []uint32) error {
	if len(pids) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(pids))
	for _, p := range pids {
		ids = append(ids, int64(p))
	}
	rows, err := database.Manager.Query(`SELECT pid FROM wiiu.principal_basic_info WHERE pid = ANY($1::int[])`, pq.Array(ids))
	if err != nil {
		return err
	}
	have := map[uint32]bool{}
	for rows.Next() {
		var pid uint32
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		have[pid] = true
	}
	rows.Close()
	for _, pid := range pids {
		if have[pid] {
			continue
		}
		if err := materializeProfile(pid); err != nil {
			return err
		}
	}
	return nil
}

func materializeProfile(pid uint32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, common_globals.GRPCAccountCommonMetadata)
	user, err := common_globals.GRPCAccountClient.GetUserData(ctx, &pb.GetUserDataRequest{Pid: pid})
	if err != nil {
		return err
	}
	miiData, _ := base64.StdEncoding.DecodeString(user.GetMii().GetData())
	now := types.NewDateTime(0)
	now.FromTimestamp(time.Now())
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO wiiu.principal_basic_info (pid, username, unknown) VALUES ($1, $2, 2) ON CONFLICT (pid) DO NOTHING`, []any{pid, user.GetUsername()}},
		{`INSERT INTO wiiu.mii (pid, name, unknown1, unknown2, data, unknown_datetime) VALUES ($1, $2, 0, 0, $3, $4) ON CONFLICT (pid) DO NOTHING`, []any{pid, user.GetMii().GetName(), miiData, uint64(now)}},
		{`INSERT INTO wiiu.network_account_info (pid, unknown1, unknown2, birthday) VALUES ($1, 0, 0, 0) ON CONFLICT (pid) DO NOTHING`, []any{pid}},
		{`INSERT INTO wiiu.user_data (pid) VALUES ($1) ON CONFLICT (pid) DO NOTHING`, []any{pid}},
	}
	for _, st := range stmts {
		if _, err := database.Manager.Exec(st.q, st.args...); err != nil {
			return err
		}
	}
	return nil
}
