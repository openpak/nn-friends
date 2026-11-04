// Package erase removes everything nn-friends holds about one OpenPak account
// (card openpak/prds#120). The website's erasure finaliser calls it over HTTP
// (DELETE /internal/accounts/{account_id}/data) before the core deletes the
// account's console links; the core's account_deleted event calls it again as
// a backstop. Both paths are the same function and it is idempotent.
package erase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// Service is the name reported in the response and the log line.
const Service = "nn-friends"

// Eraser deletes an account's friends data.
type Eraser struct {
	DB *sql.DB
	// PIDs returns every console PID (Wii U and 3DS, shadow included) the
	// account has. It must return an error rather than an empty list when it
	// could not find out, so a failed lookup is retried and not mistaken for
	// "nothing to erase".
	PIDs func(ctx context.Context, accountID string) ([]uint32, error)
	// Drop ends the live sessions of these PIDs. Optional.
	Drop func(pids []uint32)
	// Log writes the single "account erased" line. Optional.
	Log func(accountID string, rows int)
}

// Every statement takes the PID list as $1. A friendship, block or request
// row belongs to both people in it; it is removed whichever side the erased
// account is on.
var statements = []string{
	`DELETE FROM wiiu.user_data WHERE pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.network_account_info WHERE pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.principal_basic_info WHERE pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.mii WHERE pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.friendships WHERE user1_pid = ANY($1::bigint[]) OR user2_pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.blocks WHERE blocker_pid = ANY($1::bigint[]) OR blocked_pid = ANY($1::bigint[])`,
	`DELETE FROM wiiu.friend_requests WHERE sender_pid = ANY($1::bigint[]) OR recipient_pid = ANY($1::bigint[])`,
	`DELETE FROM "3ds".user_data WHERE pid = ANY($1::bigint[])`,
	`DELETE FROM "3ds".friendships WHERE user1_pid = ANY($1::bigint[]) OR user2_pid = ANY($1::bigint[])`,
}

// Erase removes the account's rows in one transaction and returns how many.
func (e *Eraser) Erase(ctx context.Context, accountID string) (int, error) {
	pids, err := e.PIDs(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("resolve pids: %w", err)
	}
	if len(pids) == 0 {
		return e.done(accountID, 0), nil
	}
	// Sessions first: a connection that ends after the delete would write its
	// last-online time back as a fresh row.
	if e.Drop != nil {
		e.Drop(pids)
	}
	ids := make([]int64, len(pids))
	for i, p := range pids {
		ids[i] = int64(p)
	}
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	total := 0
	for _, stmt := range statements {
		res, err := tx.ExecContext(ctx, stmt, pq.Array(ids))
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		total += int(n)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return e.done(accountID, total), nil
}

func (e *Eraser) done(accountID string, rows int) int {
	if e.Log != nil {
		e.Log(accountID, rows)
	}
	return rows
}

// ErrNotConfigured is returned by Default users before Start ran.
var ErrNotConfigured = errors.New("erase: not configured")

// Default is the process-wide eraser, set by Start.
var Default *Eraser

// Run erases through Default.
func Run(ctx context.Context, accountID string) (int, error) {
	if Default == nil {
		return 0, ErrNotConfigured
	}
	return Default.Erase(ctx, accountID)
}
