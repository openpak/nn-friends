package erase

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/friends/globals"
)

// Start builds Default and serves the erase endpoint on ERASE_ADDR (default
// :9099, container network only). Without ACCOUNT_ERASE_KEY the endpoint is
// not served; the account_deleted event path still works.
func Start() {
	db, err := sql.Open("postgres", globals.Config.PostgresURI)
	if err != nil {
		globals.Logger.Criticalf("erase: postgres: %v", err)
		return
	}
	db.SetMaxOpenConns(2)
	Default = &Eraser{
		DB: db,
		PIDs: func(ctx context.Context, accountID string) ([]uint32, error) {
			return coregraph.C().PIDsOfAccount(ctx, accountID)
		},
		Drop: dropSessions,
		Log: func(accountID string, rows int) {
			globals.Logger.Infof("account erased account=%s rows=%d service=%s", accountID, rows, Service)
		},
	}
	key := os.Getenv("ACCOUNT_ERASE_KEY")
	if key == "" {
		globals.Logger.Warning("ACCOUNT_ERASE_KEY not set: the erase endpoint is not served")
		return
	}
	addr := os.Getenv("ERASE_ADDR")
	if addr == "" {
		addr = ":9099"
	}
	h := Handler(key, func(r *http.Request, accountID string) (int, error) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		n, err := Default.Erase(ctx, accountID)
		if err != nil {
			globals.Logger.Errorf("erase: account=%s: %v", accountID, err)
		}
		return n, err
	})
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		globals.Logger.Criticalf("erase: listener: %v", err)
	}
}

func dropSessions(pids []uint32) {
	for _, pid := range pids {
		user, ok := globals.ConnectedUsers.Get(pid)
		if !ok || user == nil || user.Connection == nil {
			continue
		}
		globals.SecureEndpoint.CleanupConnection(user.Connection)
	}
}
