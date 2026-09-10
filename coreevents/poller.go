// Package coreevents turns the account core's friend events into console
// notifications. The core is the one friend graph; when a request, an
// acceptance or a removal happens on another surface (Switch, phone, website,
// another console family), an online Wii U or 3DS hears about it here instead
// of at its next full refresh.
package coreevents

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	notifications_3ds "github.com/PretendoNetwork/friends/notifications/3ds"
	notifications_wiiu "github.com/PretendoNetwork/friends/notifications/wiiu"
	friends_types "github.com/PretendoNetwork/friends/types"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

const pollEvery = 2 * time.Second

// Start polls forever. Call after coregraph.Init and the database are up.
func Start() {
	if !coregraph.Configured() {
		return
	}
	if _, err := database.Manager.Exec(`CREATE TABLE IF NOT EXISTS core_events_cursor (id integer PRIMARY KEY, version bigint NOT NULL)`); err != nil {
		globals.Logger.Criticalf("coreevents: cursor table: %v", err)
		return
	}
	version := loadCursor()
	for {
		page, err := coregraph.C().PollEvents(context.Background(), version)
		if err != nil {
			globals.Logger.Warningf("coreevents: poll: %v", err)
			time.Sleep(5 * pollEvery)
			continue
		}
		for _, e := range page.GetEvents() {
			handle(e.GetType(), e.GetAccountId(), e.GetSubjectId())
			version = e.GetVersion()
		}
		if len(page.GetEvents()) > 0 {
			saveCursor(version)
			continue // drain
		}
		time.Sleep(pollEvery)
	}
}

func loadCursor() uint64 {
	var v int64
	if row, err := database.Manager.QueryRow(`SELECT version FROM core_events_cursor WHERE id=1`); err == nil {
		if err := row.Scan(&v); err == nil {
			return uint64(v)
		}
	}
	// A fresh cursor starts at the head: replaying history would re-notify
	// every console about friendships it already has.
	if page, err := coregraph.C().PollEvents(context.Background(), ^uint64(0)>>1); err == nil {
		saveCursor(page.GetMaxVersion())
		return page.GetMaxVersion()
	}
	return 0
}

func saveCursor(v uint64) {
	database.Manager.Exec(`INSERT INTO core_events_cursor (id, version) VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET version = EXCLUDED.version`, int64(v))
}

func handle(typ, accountID, subjectID string) {
	switch typ {
	case "friend_requested", "friend_accepted", "friend_removed":
	default:
		return
	}
	if typ == "friend_removed" {
		// Local request metadata must not resurrect a friendship the core ended.
		if a, err := coregraph.C().PIDOfAccount(context.Background(), "wiiu", accountID); err == nil {
			if b, err := coregraph.C().PIDOfAccount(context.Background(), "wiiu", subjectID); err == nil {
				database.Manager.Exec(`DELETE FROM wiiu.friend_requests WHERE (sender_pid=$1 AND recipient_pid=$2) OR (sender_pid=$2 AND recipient_pid=$1)`, a, b)
			}
		}
	}
	pid, online := globals.OnlinePIDOfAccount(accountID)
	if !online {
		return // the next list refresh materializes it
	}
	user, ok := globals.ConnectedUsers.Get(pid)
	if !ok || user == nil || user.Connection == nil {
		return
	}
	namespace := "wiiu"
	if user.Platform == friends_types.CTR {
		namespace = "3ds"
	}
	otherPID, err := coregraph.C().PIDOfAccount(context.Background(), namespace, subjectID)
	if err != nil {
		globals.Logger.Warningf("coreevents: %s: resolve %s: %v", typ, subjectID, err)
		return
	}
	switch user.Platform {
	case friends_types.WUP:
		switch typ {
		case "friend_requested":
			requests, err := database_wiiu.GetUserFriendRequestsIn(pid)
			if err != nil {
				return
			}
			for _, fr := range requests {
				if uint32(fr.PrincipalInfo.PID) == otherPID {
					notifications_wiiu.SendFriendRequest(user.Connection, fr)
					return
				}
			}
		case "friend_accepted":
			database.Manager.Exec(`UPDATE wiiu.friend_requests SET accepted=true WHERE sender_pid=$1 AND recipient_pid=$2`, pid, otherPID)
			infos, err := database_wiiu.FriendInfosForPIDs(pid, []uint32{otherPID})
			if err != nil || len(infos) == 0 {
				return
			}
			notifications_wiiu.SendFriendRequestAccepted(user.Connection, infos[0])
		case "friend_removed":
			notifications_wiiu.SendFriendshipRemoved(user.Connection, types.NewPID(uint64(otherPID)))
		}
	case friends_types.CTR:
		if typ == "friend_accepted" {
			notifications_3ds.SendFriendshipCompleted(user.Connection, types.NewPID(uint64(otherPID)))
		}
		// A 3DS learns of new and removed relationships on its next friend-list sync.
	}
}
