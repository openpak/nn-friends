package coreevents

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	database_3ds "github.com/PretendoNetwork/friends/database/3ds"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/profilecache"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// refreshDelay lets nn-account take the same profile_updated event before this
// service re-reads the profile from it.
const refreshDelay = 2 * time.Second

// profileUpdated drops the cached profile of every PID the account has, then
// refreshes the PIDs that already have cache rows. A PID with no row is filled
// by its next read.
func profileUpdated(accountID string) {
	pids, err := coregraph.C().PIDsOfAccount(context.Background(), accountID)
	if err != nil {
		globals.Logger.Warningf("coreevents: profile_updated: pids of %s: %v", accountID, err)
		return
	}
	for _, pid := range pids {
		profilecache.Invalidate(pid)
	}
	go func() {
		time.Sleep(refreshDelay)
		for _, pid := range pids {
			profilecache.Invalidate(pid)
			var n int
			if row, err := database.Manager.QueryRow(`SELECT (SELECT count(*) FROM wiiu.principal_basic_info WHERE pid=$1) + (SELECT count(*) FROM "3ds".user_data WHERE pid=$1)`, int64(pid)); err != nil || row.Scan(&n) != nil || n == 0 {
				continue
			}
			if err := database_wiiu.RefreshProfile(pid); err != nil {
				globals.Logger.Warningf("coreevents: profile_updated: refresh pid %d: %v", pid, err)
				continue
			}
			if err := database_3ds.RefreshProfile(pid); err != nil {
				globals.Logger.Warningf("coreevents: profile_updated: refresh 3ds pid %d: %v", pid, err)
			}
		}
	}()
}

// PurgeOrphans deletes cache and settings rows whose PID has no PNID in
// nn-account at all (leftovers from before the adapter owned identity; their
// owners could never erase them). A PID nn-account does not answer for with
// "No PNID found" or "deleted" is kept. Logs counts only. Retries until the
// adapter answers, then runs once.
func PurgeOrphans() {
	for attempt := 0; attempt < 60; attempt++ {
		if attempt > 0 {
			time.Sleep(10 * time.Second)
		}
		pids, err := cachedPIDs()
		if err != nil {
			globals.Logger.Warningf("orphan purge: list pids: %v", err)
			continue
		}
		var orphans []int64
		failed := false
		for _, pid := range pids {
			orphan, err := isOrphan(pid)
			if err != nil {
				failed = true
				break
			}
			if orphan {
				orphans = append(orphans, int64(pid))
			}
		}
		if failed {
			globals.Logger.Warningf("orphan purge: adapter not answering, retrying")
			continue
		}
		deleted, err := deleteRows(orphans)
		if err != nil {
			globals.Logger.Warningf("orphan purge: delete: %v", err)
			continue
		}
		globals.Logger.Infof("orphan purge: checked=%d orphans=%d rows_deleted=%d", len(pids), len(orphans), deleted)
		return
	}
}

func cachedPIDs() ([]uint32, error) {
	rows, err := database.Manager.Query(`SELECT pid FROM wiiu.user_data UNION SELECT pid FROM wiiu.principal_basic_info UNION SELECT pid FROM wiiu.mii UNION SELECT pid FROM wiiu.network_account_info UNION SELECT pid FROM "3ds".user_data`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uint32
	for rows.Next() {
		var p int64
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, uint32(p))
	}
	return out, rows.Err()
}

func isOrphan(pid uint32) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, common_globals.GRPCAccountCommonMetadata)
	user, err := common_globals.GRPCAccountClient.GetUserData(ctx, &pb.GetUserDataRequest{Pid: pid})
	if err != nil {
		if status.Code(err) == codes.InvalidArgument && status.Convert(err).Message() == "No PNID found" {
			return true, nil
		}
		return false, err
	}
	return user.GetDeleted(), nil
}

func deleteRows(pids []int64) (int, error) {
	if len(pids) == 0 {
		return 0, nil
	}
	total := 0
	for _, q := range []string{
		`DELETE FROM wiiu.user_data WHERE pid = ANY($1::bigint[])`,
		`DELETE FROM wiiu.network_account_info WHERE pid = ANY($1::bigint[])`,
		`DELETE FROM wiiu.principal_basic_info WHERE pid = ANY($1::bigint[])`,
		`DELETE FROM wiiu.mii WHERE pid = ANY($1::bigint[])`,
		`DELETE FROM "3ds".user_data WHERE pid = ANY($1::bigint[])`,
	} {
		res, err := database.Manager.Exec(q, pq.Array(pids))
		if err != nil {
			return total, err
		}
		n, _ := res.RowsAffected()
		total += int(n)
	}
	return total, nil
}
