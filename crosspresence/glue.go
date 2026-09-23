package crosspresence

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/globals"
	accountv1 "github.com/PretendoNetwork/friends/internal/accountpb"
	friends_types "github.com/PretendoNetwork/friends/types"
)

var publisher *Publisher

func namespaceOf(p friends_types.Platform) string {
	if p == friends_types.CTR {
		return "3ds"
	}
	return "wiiu"
}

// Connected is everybody connected here, as the publisher sees them.
func Connected() []Online {
	var out []Online
	globals.ConnectedUsers.Each(func(pid uint32, u *friends_types.ConnectedUser) bool {
		if u == nil || pid == 0 {
			return false
		}
		ns := namespaceOf(u.Platform)
		var title uint64
		if u.Platform == friends_types.CTR {
			title = uint64(u.Presence.GameKey.TitleID)
		} else {
			title = uint64(u.PresenceV2.GameKey.TitleID)
		}
		out = append(out, Online{PID: pid, Namespace: ns, TitleID: TitleIDOf(ns, title)})
		return false
	})
	return out
}

// Start publishes presence into the core for as long as the process runs, and
// calls tick after each pass (the foreign-presence notifier).
func Start(tick func()) {
	if !coregraph.Configured() {
		return
	}
	publisher = &Publisher{Core: coregraph.C().Sessions(), AccountOf: coregraph.C().AccountOfPID}
	for range time.Tick(ReconcileEvery) {
		ctx, cancel := context.WithTimeout(context.Background(), ReconcileEvery)
		publisher.Reconcile(ctx, Connected())
		cancel()
		if tick != nil {
			tick()
		}
	}
}

// Disconnected ends a person's core session now rather than at lease end.
func Disconnected(pid uint32) {
	if publisher == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		publisher.Expire(ctx, pid)
	}()
}

// ForeignFriends is, for a viewer on namespace, each friend PID that is live
// on another platform with the line to show for them. Friends are the core's
// (coregraph); a friend connected here is left to its own presence.
func ForeignFriends(namespace string, viewerPID uint32) map[uint32]string {
	if !coregraph.Configured() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	byPID, err := coregraph.C().FriendAccountsByPID(ctx, namespace, viewerPID)
	if err != nil || len(byPID) == 0 {
		return nil
	}
	return ForeignFor(ctx, coregraph.C().Sessions(), namespace, byPID, func(pid uint32) bool {
		return globals.ConnectedUsers.Has(pid)
	})
}

// ForeignFor is ForeignFriends with its inputs given.
func ForeignFor(ctx context.Context, core accountv1.SessionsClient, namespace string, byPID map[uint32]string, connected func(uint32) bool) map[uint32]string {
	accounts := make([]string, 0, len(byPID))
	for pid, a := range byPID {
		if !connected(pid) {
			accounts = append(accounts, a)
		}
	}
	live := Foreign(ctx, core, namespace, accounts)
	if len(live) == 0 {
		return nil
	}
	out := map[uint32]string{}
	for pid, a := range byPID {
		if l, ok := live[a]; ok && !connected(pid) {
			out[pid] = Description(l)
		}
	}
	return out
}
