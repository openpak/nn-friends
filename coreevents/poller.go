// Package coreevents turns the account core's friend events into console
// notifications. The core is the one friend graph; when a request, an
// acceptance or a removal happens on another surface (Switch, phone, website,
// another console family), an online Wii U or 3DS hears about it here instead
// of at its next full refresh.
//
// Delivery rides the core's SubscribeAccountEvents (universal-social-prd US-2)
// with the poll kept as the fallback: an adapter that cannot reach the stream
// still catches up every pollEvery, and the cursor is the same version either
// way, so the two paths are interchangeable at any moment.
package coreevents

import (
	"context"
	"errors"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/database"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	notifications_3ds "github.com/PretendoNetwork/friends/notifications/3ds"
	notifications_wiiu "github.com/PretendoNetwork/friends/notifications/wiiu"
	friends_types "github.com/PretendoNetwork/friends/types"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	pollEvery   = 2 * time.Second  // the fallback tick, also the drain pace
	retryStream = 5 * time.Second  // wait before reopening a failed stream
	legacyWait  = 30 * time.Second // a core without the RPC: stay on the poll a while
)

// Start runs the event loop forever. Call after coregraph.Init and the
// database are up.
func Start() {
	if !coregraph.Configured() {
		return
	}
	if _, err := database.Manager.Exec(`CREATE TABLE IF NOT EXISTS core_events_cursor (id integer PRIMARY KEY, version bigint NOT NULL)`); err != nil {
		globals.Logger.Criticalf("coreevents: cursor table: %v", err)
		return
	}
	for {
		version := loadCursor()
		version = drain(version)
		if legacy, ok := stream(version); !ok {
			time.Sleep(retryStream)
			continue
		} else if legacy {
			// This core does not carry the subscription yet: keep the old
			// poll rhythm until it does, and ask again after a while.
			deadline := time.Now().Add(legacyWait)
			for time.Now().Before(deadline) {
				time.Sleep(pollEvery)
				version = drain(version)
			}
			continue
		}
		// stream returned because the connection broke; a short pause, then
		// catch up and resubscribe from the cursor.
		time.Sleep(retryStream)
	}
}

// stream subscribes from version and handles events until the stream breaks.
// Reports (legacy, ok): legacy means the core has no subscription RPC at all
// (an older core: the poll is the transport); ok false means a connection
// failure worth retrying after retryStream.
func stream(version uint64) (legacy, ok bool) {
	ctx := context.Background()
	sub, err := coregraph.C().SubscribeEvents(ctx, version)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			globals.Logger.Warningf("coreevents: core has no subscription, falling back to polling: %v", err)
			return true, true
		}
		globals.Logger.Warningf("coreevents: subscribe: %v", err)
		return false, false
	}
	globals.Logger.Infof("coreevents: subscribed from version %d", version)
	for {
		ev, err := sub.Recv()
		if err != nil {
			if !errors.Is(err, context.Canceled) && ctx.Err() == nil {
				globals.Logger.Warningf("coreevents: stream: %v", err)
			}
			return false, false
		}
		if ev.GetVersion() <= version {
			continue // at-least-once delivery: a replay across reconnects
		}
		handle(ev.GetType(), ev.GetAccountId(), ev.GetSubjectId())
		version = ev.GetVersion()
		saveCursor(version)
	}
}

// drain pages everything the cursor has not seen, whatever the transport
// later does. Used before subscribing and as the poll fallback's tick.
func drain(version uint64) uint64 {
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
		}
		if len(page.GetEvents()) < 500 {
			return version
		}
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
	case "message":
		// The friends protocol has no message push, and the chat surfaces
		// deliver their own messages; record the drop honestly (universal-
		// social-prd §4b.3) rather than pretending the console was told.
		globals.Logger.Infof("coreevents: message for %s dropped here; the chat surface delivers it", accountID)
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
