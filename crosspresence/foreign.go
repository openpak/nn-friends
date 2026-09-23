package crosspresence

import (
	"context"
	"strings"
	"time"

	accountv1 "github.com/PretendoNetwork/friends/internal/accountpb"
)

// Live is what the core says about one account.
type Live struct {
	Namespace, TitleID, Client string
}

// Foreign answers which of the given accounts are live on another platform
// than viewerNamespace, according to the core. Accounts live on this platform
// are left out: those render from the connection they hold here, as they
// always have. A core that cannot answer means nobody, never a guess.
func Foreign(ctx context.Context, core accountv1.SessionsClient, viewerNamespace string, accountIDs []string) map[string]Live {
	if core == nil || len(accountIDs) == 0 {
		return nil
	}
	if len(accountIDs) > 200 {
		accountIDs = accountIDs[:200] // the core's cap; a friend list is at most 100 here
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	resp, err := core.GetPresence(ctx, &accountv1.GetPresenceRequest{AccountIds: accountIDs})
	if err != nil {
		return nil
	}
	out := map[string]Live{}
	for _, p := range resp.GetPresence() {
		if strings.EqualFold(p.GetNamespace(), viewerNamespace) {
			continue
		}
		out[p.GetAccountId()] = Live{Namespace: p.GetNamespace(), TitleID: p.GetTitleId(), Client: p.GetClient()}
	}
	return out
}
