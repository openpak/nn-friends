// Package coregraph routes friends' canonical graph operations to the
// OpenPak account core (PRD FR-4 / §7). Friends keeps presence and
// protocol presentation locally; pending/accepted/removed state and
// blocks live in the core. PID ↔ account resolution goes through the
// nn-account adapter's Resolution service — never the core directly.
package coregraph

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	accountv1 "github.com/PretendoNetwork/friends/internal/accountpb"
	resolutionv1 "github.com/PretendoNetwork/friends/internal/resolutionpb"
)

var (
	ErrNotConfigured      = errors.New("coregraph: not configured")
	ErrResolutionNotFound = errors.New("coregraph: pid/account not resolvable")
)

const (
	cacheTTL    = 5 * time.Minute
	negativeTTL = 30 * time.Second
)

// Client owns the core Social client and adapter Resolution client.
type Client struct {
	core       accountv1.SocialClient
	events     accountv1.EventsClient
	coreKey    string
	res        resolutionv1.ResolutionClient
	adapterKey string

	mu  sync.Mutex
	fwd map[string]cacheEntry[string] // "ns|pid" -> account id
	rev map[string]cacheEntry[uint32] // "ns|account" -> pid
}

type cacheEntry[T any] struct {
	value T
	load  bool
	exp   time.Time
}

var (
	defaultClient     *Client
	defaultClientOnce sync.Once
)

// Init configures the process-wide client. Safe to call once at startup.
func Init(coreAddr, coreKey, adapterAddr, adapterKey string) error {
	coreConn, err := grpc.NewClient(coreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("coregraph: core dial: %w", err)
	}
	adapterConn, err := grpc.NewClient(adapterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("coregraph: adapter dial: %w", err)
	}
	defaultClient = &Client{
		core:       accountv1.NewSocialClient(coreConn),
		events:     accountv1.NewEventsClient(coreConn),
		coreKey:    coreKey,
		res:        resolutionv1.NewResolutionClient(adapterConn),
		adapterKey: adapterKey,
		fwd:        make(map[string]cacheEntry[string]),
		rev:        make(map[string]cacheEntry[uint32]),
	}
	return nil
}

func C() *Client {
	return defaultClient
}

// Configured reports whether Init succeeded.
func Configured() bool { return defaultClient != nil }

// --- resolution ---

func (c *Client) AccountOfPID(ctx context.Context, namespace string, pid uint32) (string, error) {
	if c == nil {
		return "", ErrNotConfigured
	}
	key := namespace + "|" + fmt.Sprint(pid)
	c.mu.Lock()
	if e, ok := c.fwd[key]; ok && time.Now().Before(e.exp) {
		c.mu.Unlock()
		if !e.load {
			return "", ErrResolutionNotFound
		}
		return e.value, nil
	}
	c.mu.Unlock()

	rctx, cancel := context.WithTimeout(metadata.AppendToOutgoingContext(ctx,
		"X-API-Key", c.adapterKey), 5*time.Second)
	defer cancel()
	resp, err := c.res.ResolvePid(rctx, &resolutionv1.ResolvePidRequest{Namespace: namespace, Pid: pid})
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	if resp.GetFound() {
		c.fwd[key] = cacheEntry[string]{value: resp.GetAccountId(), load: true, exp: time.Now().Add(cacheTTL)}
	} else {
		c.fwd[key] = cacheEntry[string]{load: false, exp: time.Now().Add(negativeTTL)}
	}
	c.mu.Unlock()
	if !resp.GetFound() {
		return "", ErrResolutionNotFound
	}
	return resp.GetAccountId(), nil
}

func (c *Client) PIDOfAccount(ctx context.Context, namespace, accountID string) (uint32, error) {
	if c == nil {
		return 0, ErrNotConfigured
	}
	key := namespace + "|" + accountID
	c.mu.Lock()
	if e, ok := c.rev[key]; ok && time.Now().Before(e.exp) {
		c.mu.Unlock()
		if !e.load {
			return 0, ErrResolutionNotFound
		}
		return e.value, nil
	}
	c.mu.Unlock()

	rctx, cancel := context.WithTimeout(metadata.AppendToOutgoingContext(ctx,
		"X-API-Key", c.adapterKey), 5*time.Second)
	defer cancel()
	resp, err := c.res.ResolveAccount(rctx, &resolutionv1.ResolveAccountRequest{Namespace: namespace, AccountId: accountID})
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	if resp.GetFound() {
		c.rev[key] = cacheEntry[uint32]{value: resp.GetPid(), load: true, exp: time.Now().Add(cacheTTL)}
	} else {
		c.rev[key] = cacheEntry[uint32]{load: false, exp: time.Now().Add(negativeTTL)}
	}
	c.mu.Unlock()
	if !resp.GetFound() {
		return 0, ErrResolutionNotFound
	}
	return resp.GetPid(), nil
}

// --- social ops (core account IDs; caller = friends service) ---

func (c *Client) coreCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-key", c.coreKey)
}

func (c *Client) FriendAccounts(ctx context.Context, namespace string, pid uint32) ([]string, error) {
	account, err := c.AccountOfPID(ctx, namespace, pid)
	if err != nil {
		return nil, err
	}
	resp, err := c.core.ListFriends(c.coreCtx(ctx), &accountv1.ListFriendsRequest{AccountId: account})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.GetFriends()))
	for _, f := range resp.GetFriends() {
		out = append(out, f.GetAccountId())
	}
	return out, nil
}

// FriendPIDs returns the pid's friends' PIDs. Unresolvable accounts (e.g.
// users who have never connected) are skipped, matching upstream behavior
// of only listing known principals.
func (c *Client) FriendPIDs(ctx context.Context, namespace string, pid uint32) ([]uint32, error) {
	accounts, err := c.FriendAccounts(ctx, namespace, pid)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, 0, len(accounts))
	for _, a := range accounts {
		if friendPID, err := c.PIDOfAccount(ctx, namespace, a); err == nil {
			out = append(out, friendPID)
		}
	}
	return out, nil
}

// RequestState mirrors the 3DS friendship model.
type RequestState int

const (
	StateInvalid RequestState = iota
	StateIncomplete
	StateComplete
)

// Request sends A→B. Completes immediately when B already has an active
// pending request toward A (mutual add) or they are already friends.
func (c *Client) Request(ctx context.Context, namespace string, fromPID, toPID uint32) (RequestState, error) {
	from, err := c.AccountOfPID(ctx, namespace, fromPID)
	if err != nil {
		return StateInvalid, err
	}
	to, err := c.AccountOfPID(ctx, namespace, toPID)
	if err != nil {
		return StateInvalid, err
	}
	resp, err := c.core.RequestFriend(c.coreCtx(ctx), &accountv1.RequestFriendRequest{
		RequesterId: from, AddresseeId: to,
		IdempotencyKey: fmt.Sprintf("%s-%d-%d", namespace, fromPID, toPID),
	})
	if err != nil {
		// 3DS mutual-add: when a reverse pending request exists the core
		// refuses the duplicate; the second request COMPLETES the friendship
		// instead of failing (protocol semantics).
		if status.Code(err) == codes.FailedPrecondition {
			if err := c.AcceptRequest(ctx, namespace, fromPID, toPID); err == nil {
				return StateComplete, nil
			}
		}
		return StateInvalid, err
	}
	if resp.GetState() == accountv1.FriendState_FRIEND_STATE_ACCEPTED {
		return StateComplete, nil
	}
	// Pending: complete when the OTHER side also requested (mutual add).
	rel, err := c.core.GetRelationship(c.coreCtx(ctx), &accountv1.GetRelationshipRequest{
		AccountId: to, OtherId: from})
	if err == nil && rel.GetRelationship() == accountv1.Relationship_RELATIONSHIP_PENDING_INCOMING {
		// from->to is outgoing from `to`'s viewpoint? No: incoming to `to`
		// means `from` sent it — one-sided. Incomplete is correct.
		return StateIncomplete, nil
	}
	return StateIncomplete, nil
}

// AcceptRequest accepts a pending request from requester toward addressee.
func (c *Client) AcceptRequest(ctx context.Context, namespace string, addresseePID, requesterPID uint32) error {
	addressee, err := c.AccountOfPID(ctx, namespace, addresseePID)
	if err != nil {
		return err
	}
	requester, err := c.AccountOfPID(ctx, namespace, requesterPID)
	if err != nil {
		return err
	}
	_, err = c.core.AcceptFriend(c.coreCtx(ctx), &accountv1.AcceptFriendRequest{
		AddresseeId: addressee, RequesterId: requester,
	})
	return err
}

// Remove ends the relationship between the two accounts (either side).
func (c *Client) Remove(ctx context.Context, namespace string, pidA, pidB uint32) error {
	a, err := c.AccountOfPID(ctx, namespace, pidA)
	if err != nil {
		return err
	}
	b, err := c.AccountOfPID(ctx, namespace, pidB)
	if err != nil {
		return err
	}
	_, err = c.core.RemoveFriend(c.coreCtx(ctx), &accountv1.RemoveFriendRequest{AccountId: a, OtherId: b})
	return err
}

// Block records an account-level block.
func (c *Client) Block(ctx context.Context, namespace string, blockerPID, blockedPID uint32) error {
	blocker, err := c.AccountOfPID(ctx, namespace, blockerPID)
	if err != nil {
		return err
	}
	blocked, err := c.AccountOfPID(ctx, namespace, blockedPID)
	if err != nil {
		return err
	}
	_, err = c.core.Block(c.coreCtx(ctx), &accountv1.BlockRequest{BlockerId: blocker, BlockedId: blocked})
	return err
}

// Unblock removes an account-level block.
func (c *Client) Unblock(ctx context.Context, namespace string, blockerPID, blockedPID uint32) error {
	blocker, err := c.AccountOfPID(ctx, namespace, blockerPID)
	if err != nil {
		return err
	}
	blocked, err := c.AccountOfPID(ctx, namespace, blockedPID)
	if err != nil {
		return err
	}
	_, err = c.core.Unblock(c.coreCtx(ctx), &accountv1.UnblockRequest{BlockerId: blocker, BlockedId: blocked})
	return err
}

// IncomingRequesterPIDs returns PIDs with pending requests toward pid.
func (c *Client) IncomingRequesterPIDs(ctx context.Context, namespace string, pid uint32) ([]uint32, error) {
	account, err := c.AccountOfPID(ctx, namespace, pid)
	if err != nil {
		return nil, err
	}
	resp, err := c.core.ListIncomingRequests(c.coreCtx(ctx), &accountv1.ListIncomingRequestsRequest{AccountId: account})
	if err != nil {
		return nil, err
	}
	out := make([]uint32, 0, len(resp.GetRequesters()))
	for _, f := range resp.GetRequesters() {
		if p, err := c.PIDOfAccount(ctx, namespace, f.GetAccountId()); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

// OutgoingAddresseePIDs returns PIDs pid has pending requests toward.
func (c *Client) OutgoingAddresseePIDs(ctx context.Context, namespace string, pid uint32) ([]uint32, error) {
	account, err := c.AccountOfPID(ctx, namespace, pid)
	if err != nil {
		return nil, err
	}
	resp, err := c.core.ListOutgoingRequests(c.coreCtx(ctx), &accountv1.ListOutgoingRequestsRequest{AccountId: account})
	if err != nil {
		return nil, err
	}
	out := make([]uint32, 0, len(resp.GetAddressees()))
	for _, f := range resp.GetAddressees() {
		if p, err := c.PIDOfAccount(ctx, namespace, f.GetAccountId()); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

// IsBlocked reports whether either account blocks the other.
func (c *Client) IsBlocked(ctx context.Context, namespace string, pidA, pidB uint32) (bool, error) {
	a, err := c.AccountOfPID(ctx, namespace, pidA)
	if err != nil {
		return false, err
	}
	b, err := c.AccountOfPID(ctx, namespace, pidB)
	if err != nil {
		return false, err
	}
	rel, err := c.core.GetRelationship(c.coreCtx(ctx), &accountv1.GetRelationshipRequest{
		AccountId: a, OtherId: b})
	if err != nil {
		return false, err
	}
	r := rel.GetRelationship()
	return r == accountv1.Relationship_RELATIONSHIP_BLOCKED_BY_SELF ||
		r == accountv1.Relationship_RELATIONSHIP_BLOCKED_BY_OTHER, nil
}

// PollEvents pages the core's event stream from sinceVersion (exclusive).
func (c *Client) PollEvents(ctx context.Context, sinceVersion uint64) (*accountv1.PollEventsResponse, error) {
	rctx, cancel := context.WithTimeout(c.coreCtx(ctx), 10*time.Second)
	defer cancel()
	return c.events.PollEvents(rctx, &accountv1.PollEventsRequest{SinceVersion: sinceVersion, Limit: 500})
}
