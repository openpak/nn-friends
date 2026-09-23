// Package crossnotify keeps connected Wii U and 3DS consoles up to date about
// friends who are live on another platform. A console hears about presence
// only through notifications, so a friend who starts playing on a Switch after
// the Wii U signed in would otherwise stay offline until its next sign-in.
package crossnotify

import (
	"sync"

	"github.com/PretendoNetwork/friends/crosspresence"
	"github.com/PretendoNetwork/friends/globals"
	notifications_3ds "github.com/PretendoNetwork/friends/notifications/3ds"
	notifications_wiiu "github.com/PretendoNetwork/friends/notifications/wiiu"
	friends_types "github.com/PretendoNetwork/friends/types"
)

var (
	mu   sync.Mutex
	sent = map[uint32]map[uint32]string{} // viewer PID -> friend PID -> line last sent
)

// Change is one notification to send to one viewer.
type Change struct {
	FriendPID uint32
	Line      string // "" means the friend went offline
}

// Diff is what a viewer must be told to move from prev to now. A friend who
// left the foreign set but is connected here now is not sent offline: their
// own connection already announced them.
func Diff(prev, now map[uint32]string, connected func(uint32) bool) []Change {
	var out []Change
	for pid, line := range now {
		if prev[pid] != line {
			out = append(out, Change{FriendPID: pid, Line: line})
		}
	}
	for pid := range prev {
		if _, still := now[pid]; !still && !connected(pid) {
			out = append(out, Change{FriendPID: pid})
		}
	}
	return out
}

// Tick runs once per publisher pass.
func Tick() {
	type viewer struct {
		pid  uint32
		user *friends_types.ConnectedUser
	}
	var viewers []viewer
	globals.ConnectedUsers.Each(func(pid uint32, u *friends_types.ConnectedUser) bool {
		if u != nil && pid != 0 {
			viewers = append(viewers, viewer{pid, u})
		}
		return false
	})
	connected := func(pid uint32) bool { return globals.ConnectedUsers.Has(pid) }

	mu.Lock()
	defer mu.Unlock()
	live := map[uint32]bool{}
	for _, v := range viewers {
		live[v.pid] = true
		ns := "wiiu"
		if v.user.Platform == friends_types.CTR {
			ns = "3ds"
		}
		now := crosspresence.ForeignFriends(ns, v.pid)
		for _, c := range Diff(sent[v.pid], now, connected) {
			switch {
			case v.user.Platform == friends_types.CTR && c.Line != "":
				notifications_3ds.SendPresenceTo(v.user, c.FriendPID, crosspresence.ThreeDSPresence(c.Line))
			case v.user.Platform == friends_types.CTR:
				notifications_3ds.SendWentOfflineTo(v.user, c.FriendPID)
			case c.Line != "":
				notifications_wiiu.SendPresenceTo(v.user, crosspresence.WiiUPresence(c.FriendPID, c.Line))
			default:
				notifications_wiiu.SendWentOfflineTo(v.user, c.FriendPID)
			}
		}
		sent[v.pid] = now
	}
	for pid := range sent {
		if !live[pid] {
			delete(sent, pid)
		}
	}
}
