package crosspresence

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	accountv1 "github.com/PretendoNetwork/friends/internal/accountpb"
)

// Lease is the core session lease; the reconciler renews every ReconcileEvery,
// so one lost heartbeat is not somebody blinking offline.
const (
	Lease          = 45 * time.Second
	ReconcileEvery = 15 * time.Second
)

// Online is one person connected here right now.
type Online struct {
	PID       uint32
	Namespace string // "wiiu" or "3ds"
	TitleID   string // "" when no game is running
	// Client is "" today: a Cemu or Azahar identity is the same PNID and the
	// same NEX login as the console it came from (nn-account mints one
	// identity per person), so nothing here can tell them apart.
	Client string
}

// Publisher keeps one core session per connected person, in step with what
// they are playing, and ends it when they leave. The same approach as the
// Switch adapter's reconciler: register, heartbeat, re-register on a change
// the heartbeat cannot carry (title, client), expire on disconnect.
type Publisher struct {
	Core      accountv1.SessionsClient
	AccountOf func(ctx context.Context, namespace string, pid uint32) (string, error)

	mu   sync.Mutex
	live map[uint32]*published
}

type published struct {
	sessionID, accountID string
	namespace, title     string
	client               string
}

// Reconcile brings the core in line with who is connected.
func (p *Publisher) Reconcile(ctx context.Context, online []Online) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.live == nil {
		p.live = map[uint32]*published{}
	}
	seen := make(map[uint32]bool, len(online))
	for _, o := range online {
		seen[o.PID] = true
		cur := p.live[o.PID]
		if cur != nil && cur.namespace == o.Namespace && cur.title == o.TitleID && cur.client == o.Client {
			if _, err := p.Core.Heartbeat(ctx, &accountv1.HeartbeatRequest{SessionId: cur.sessionID}); err == nil {
				continue
			}
			// Gone from the core (restart, lapsed): register again below.
			delete(p.live, o.PID)
			cur = nil
		}
		if cur != nil {
			p.expireLocked(ctx, o.PID)
		}
		p.registerLocked(ctx, o)
	}
	for pid := range p.live {
		if !seen[pid] {
			p.expireLocked(ctx, pid)
		}
	}
}

// Expire ends a person's session at once (their connection ended).
func (p *Publisher) Expire(ctx context.Context, pid uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(ctx, pid)
}

func (p *Publisher) registerLocked(ctx context.Context, o Online) {
	accountID, err := p.AccountOf(ctx, o.Namespace, o.PID)
	if err != nil || accountID == "" {
		return // no core account behind this PID: nothing to publish
	}
	resp, err := p.Core.RegisterSession(ctx, &accountv1.RegisterSessionRequest{
		AccountId: accountID, Namespace: o.Namespace, TitleId: o.TitleID,
		EndpointRef: fmt.Sprintf("nex:%d", o.PID), LeaseSeconds: int32(Lease.Seconds()), Client: o.Client,
	})
	if err != nil {
		log.Printf("[crosspresence] register %s pid=%d: %v", o.Namespace, o.PID, err)
		return
	}
	p.live[o.PID] = &published{sessionID: resp.GetSessionId(), accountID: accountID,
		namespace: o.Namespace, title: o.TitleID, client: o.Client}
}

func (p *Publisher) expireLocked(ctx context.Context, pid uint32) {
	cur := p.live[pid]
	if cur == nil {
		return
	}
	delete(p.live, pid)
	if _, err := p.Core.ExpireSession(ctx, &accountv1.ExpireSessionRequest{SessionId: cur.sessionID}); err != nil {
		log.Printf("[crosspresence] expire pid=%d: %v", pid, err)
	}
}

// TitleIDOf renders a presence game key as the title id published to the
// core: 16 upper-case hex digits, or "" when it is not a game. Wii U games are
// 00050000xxxxxxxx and 3DS applications 00040000xxxxxxxx; everything else a
// console reports (its own menu, system applets) is "signed in, no game".
func TitleIDOf(namespace string, titleID uint64) string {
	hi := titleID >> 32
	switch {
	case namespace == "wiiu" && hi == 0x00050000:
	case namespace == "3ds" && hi == 0x00040000:
	default:
		return ""
	}
	return strings.ToUpper(fmt.Sprintf("%016x", titleID))
}
