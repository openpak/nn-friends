package globals

import (
	"context"
	"sync"

	"github.com/PretendoNetwork/friends/coregraph"
)

// connectedAccounts maps a core account id to the PID it is online with, so a
// core event about an account can be turned into a console notification without
// resolving every event's account through the adapter.
var (
	connectedAccountsMu sync.Mutex
	connectedAccounts   = map[string]uint32{}
	connectedPIDs       = map[uint32]string{}
)

// MarkAccountOnline records the account behind an online PID (best effort: an
// unresolvable PID simply gets no cross-surface notifications).
func MarkAccountOnline(namespace string, pid uint32) {
	if !coregraph.Configured() {
		return
	}
	go func() {
		accountID, err := coregraph.C().AccountOfPID(context.Background(), namespace, pid)
		if err != nil || accountID == "" {
			return
		}
		connectedAccountsMu.Lock()
		connectedAccounts[accountID] = pid
		connectedPIDs[pid] = accountID
		connectedAccountsMu.Unlock()
	}()
}

func MarkAccountOffline(pid uint32) {
	connectedAccountsMu.Lock()
	if accountID, ok := connectedPIDs[pid]; ok {
		delete(connectedPIDs, pid)
		if connectedAccounts[accountID] == pid {
			delete(connectedAccounts, accountID)
		}
	}
	connectedAccountsMu.Unlock()
}

// OnlinePIDOfAccount is the PID an account is currently connected with, if any.
func OnlinePIDOfAccount(accountID string) (uint32, bool) {
	connectedAccountsMu.Lock()
	defer connectedAccountsMu.Unlock()
	pid, ok := connectedAccounts[accountID]
	return pid, ok
}
