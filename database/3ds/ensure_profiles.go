package database_3ds

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/PretendoNetwork/friends/database"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
	"google.golang.org/grpc/metadata"
)

// EnsureProfiles gives PIDs the 3DS never met (Switch and phone users behind a
// shadow PNID) a user_data row with the name and Mii the adapter knows.
func EnsureProfiles(pids []uint32) error {
	if len(pids) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(pids))
	for _, p := range pids {
		ids = append(ids, int64(p))
	}
	rows, err := database.Manager.Query(`SELECT pid FROM "3ds".user_data WHERE pid = ANY($1::int[])`, pq.Array(ids))
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ctx = metadata.NewOutgoingContext(ctx, common_globals.GRPCAccountCommonMetadata)
		user, err := common_globals.GRPCAccountClient.GetUserData(ctx, &pb.GetUserDataRequest{Pid: pid})
		cancel()
		if err != nil {
			return err
		}
		miiData, _ := base64.StdEncoding.DecodeString(user.GetMii().GetData())
		if _, err := database.Manager.Exec(`INSERT INTO "3ds".user_data (pid, mii_name, mii_data) VALUES ($1, $2, $3) ON CONFLICT (pid) DO NOTHING`,
			pid, user.GetMii().GetName(), miiData); err != nil {
			return err
		}
	}
	return nil
}
