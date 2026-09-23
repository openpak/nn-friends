package crosspresence

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	accountv1 "github.com/PretendoNetwork/friends/internal/accountpb"
	"google.golang.org/grpc"
)

// fakeCore is the core's session registry: one row per (account, namespace,
// title), like the real one.
type fakeCore struct {
	accountv1.SessionsClient
	mu       sync.Mutex
	next     int
	sessions map[string]*accountv1.RegisterSessionRequest
	expired  []string
	presence []*accountv1.Presence
}

func newFakeCore() *fakeCore { return &fakeCore{sessions: map[string]*accountv1.RegisterSessionRequest{}} }

func (f *fakeCore) RegisterSession(_ context.Context, r *accountv1.RegisterSessionRequest, _ ...grpc.CallOption) (*accountv1.RegisterSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	id := fmt.Sprintf("s%d", f.next)
	f.sessions[id] = r
	return &accountv1.RegisterSessionResponse{SessionId: id, LeaseSeconds: r.GetLeaseSeconds()}, nil
}

func (f *fakeCore) Heartbeat(_ context.Context, r *accountv1.HeartbeatRequest, _ ...grpc.CallOption) (*accountv1.HeartbeatResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[r.GetSessionId()]; !ok {
		return nil, errors.New("not found")
	}
	return &accountv1.HeartbeatResponse{LeaseSeconds: 45}, nil
}

func (f *fakeCore) ExpireSession(_ context.Context, r *accountv1.ExpireSessionRequest, _ ...grpc.CallOption) (*accountv1.ExpireSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, r.GetSessionId())
	f.expired = append(f.expired, r.GetSessionId())
	return &accountv1.ExpireSessionResponse{}, nil
}

func (f *fakeCore) GetPresence(_ context.Context, r *accountv1.GetPresenceRequest, _ ...grpc.CallOption) (*accountv1.GetPresenceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := map[string]bool{}
	for _, id := range r.GetAccountIds() {
		want[id] = true
	}
	out := &accountv1.GetPresenceResponse{}
	for _, p := range f.presence {
		if want[p.GetAccountId()] {
			out.Presence = append(out.Presence, p)
		}
	}
	return out, nil
}

func (f *fakeCore) only(t *testing.T) *accountv1.RegisterSessionRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sessions) != 1 {
		t.Fatalf("want one live session, have %d", len(f.sessions))
	}
	for _, r := range f.sessions {
		return r
	}
	return nil
}

func accountOf(_ context.Context, ns string, pid uint32) (string, error) {
	if pid == 999 {
		return "", errors.New("no link")
	}
	return fmt.Sprintf("acct-%s-%d", ns, pid), nil
}

func TestPublisherRegistersRenewsAndExpires(t *testing.T) {
	core := newFakeCore()
	p := &Publisher{Core: core, AccountOf: accountOf}
	ctx := context.Background()

	// Signed in on a Wii U, no game.
	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu"}, {PID: 999, Namespace: "wiiu"}})
	r := core.only(t) // the unlinked PID publishes nothing
	if r.GetAccountId() != "acct-wiiu-7" || r.GetNamespace() != "wiiu" || r.GetTitleId() != "" ||
		r.GetEndpointRef() != "nex:7" || r.GetLeaseSeconds() != int32(Lease.Seconds()) {
		t.Fatalf("registered %+v", r)
	}

	// Same state: heartbeat, no new registration.
	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu"}})
	if core.next != 1 {
		t.Fatalf("re-registered an unchanged session (%d registrations)", core.next)
	}

	// Starts Mario Kart 8: the session is replaced, carrying the title.
	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu", TitleID: "000500001010EC00"}})
	if r := core.only(t); r.GetTitleId() != "000500001010EC00" {
		t.Fatalf("title not carried: %+v", r)
	}

	// The core lost the session (restart): registered again, not left dead.
	core.mu.Lock()
	core.sessions = map[string]*accountv1.RegisterSessionRequest{}
	core.mu.Unlock()
	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu", TitleID: "000500001010EC00"}})
	core.only(t)

	// Disconnect ends it at once.
	p.Expire(ctx, 7)
	if len(core.sessions) != 0 {
		t.Fatalf("session outlived the connection: %v", core.sessions)
	}

	// And somebody who vanished without a disconnect is expired on the next pass.
	p.Reconcile(ctx, []Online{{PID: 8, Namespace: "3ds"}})
	p.Reconcile(ctx, nil)
	if len(core.sessions) != 0 {
		t.Fatalf("stale session kept: %v", core.sessions)
	}
}

func TestTitleIDOf(t *testing.T) {
	for _, c := range []struct {
		ns   string
		id   uint64
		want string
	}{
		{"wiiu", 0x000500001010EC00, "000500001010EC00"},
		{"wiiu", 0x0005001010040200, ""}, // Wii U Menu
		{"wiiu", 0, ""},
		{"3ds", 0x0004000000030800, "0004000000030800"},
		{"3ds", 0x0004003000008F02, ""}, // HOME Menu
	} {
		if got := TitleIDOf(c.ns, c.id); got != c.want {
			t.Errorf("TitleIDOf(%s, %x) = %q, want %q", c.ns, c.id, got, c.want)
		}
	}
}

// Friends live elsewhere get a line; friends on this platform, friends
// connected here and friends offline do not.
func TestForeignFor(t *testing.T) {
	core := newFakeCore()
	core.presence = []*accountv1.Presence{
		{AccountId: "a-switch", Namespace: "switch", TitleId: "01004D300C5AE000", Client: "ryujinx"},
		{AccountId: "a-3ds", Namespace: "3ds"},
		{AccountId: "a-wiiu", Namespace: "wiiu", TitleId: "000500001010EC00"},
		{AccountId: "a-here", Namespace: "switch"},
	}
	prev := Names
	Names = &TitleNames{Fetch: func(ns, id string) (string, bool) {
		if id == "01004D300C5AE000" {
			return "Kirby", true
		}
		return "", true
	}}
	t.Cleanup(func() { Names = prev })
	Names.Name("switch", "01004D300C5AE000")
	for deadline := time.Now().Add(2 * time.Second); Names.Name("switch", "01004D300C5AE000") == "" && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
	}

	byPID := map[uint32]string{1: "a-switch", 2: "a-3ds", 3: "a-wiiu", 4: "a-here", 5: "a-offline"}
	got := ForeignFor(context.Background(), core, "wiiu", byPID, func(pid uint32) bool { return pid == 4 })
	want := map[uint32]string{1: "Playing Kirby on Ryujinx", 2: "Online on 3DS"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for pid, line := range want {
		if got[pid] != line {
			t.Errorf("pid %d: %q, want %q", pid, got[pid], line)
		}
	}

	// Unknown name: said honestly.
	if d := Description(Live{Namespace: "switch", TitleID: "0100000000010000"}); d != "Playing a game on Switch" {
		t.Errorf("unnamed: %q", d)
	}
}

func TestSyntheticPresences(t *testing.T) {
	w := WiiUPresence(42, "Online on Switch")
	if !bool(w.Online) || uint64(w.GameKey.TitleID) != 0 || string(w.Message) != "Online on Switch" || uint64(w.PID) != 42 {
		t.Fatalf("wiiu presence: %+v", w)
	}
	c := ThreeDSPresence("Playing Kirby on Eden")
	if uint64(c.GameKey.TitleID) != 0 || string(c.Message) != "Playing Kirby on Eden" {
		t.Fatalf("3ds presence: %+v", c)
	}
}
