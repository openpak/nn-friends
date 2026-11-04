package erase

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const acct = "11111111-2222-3333-4444-555555555555"

func TestNoPIDsIsZeroRowsAndLogsOnce(t *testing.T) {
	logged := 0
	e := &Eraser{
		PIDs: func(context.Context, string) ([]uint32, error) { return nil, nil },
		Log:  func(string, int) { logged++ },
	}
	for i := 0; i < 2; i++ {
		n, err := e.Erase(context.Background(), acct)
		if err != nil || n != 0 {
			t.Fatalf("run %d: n=%d err=%v", i, n, err)
		}
	}
	if logged != 2 {
		t.Fatalf("one log line per erasure, got %d for 2 runs", logged)
	}
}

func TestUnknownPIDsIsAnErrorNotZero(t *testing.T) {
	e := &Eraser{PIDs: func(context.Context, string) ([]uint32, error) { return nil, errors.New("down") }}
	if _, err := e.Erase(context.Background(), acct); err == nil {
		t.Fatal("a failed PID lookup must fail the erasure so it is retried")
	}
}

func TestHandlerAuth(t *testing.T) {
	ran := 0
	run := func(r *http.Request, id string) (int, error) { ran++; return 3, nil }
	do := func(key, header, id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("DELETE", "/internal/accounts/"+id+"/data", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rr := httptest.NewRecorder()
		Handler(key, run).ServeHTTP(rr, req)
		return rr
	}
	if rr := do("", "Bearer k", acct); rr.Code != 503 {
		t.Fatalf("unset key: %d", rr.Code)
	}
	if rr := do("k", "", acct); rr.Code != 401 {
		t.Fatalf("missing key: %d", rr.Code)
	}
	if rr := do("k", "Bearer nope", acct); rr.Code != 401 {
		t.Fatalf("wrong key: %d", rr.Code)
	}
	if rr := do("k", "Bearer k", "not-a-uuid"); rr.Code != 400 {
		t.Fatalf("bad uuid: %d", rr.Code)
	}
	if ran != 0 {
		t.Fatal("run must not be reached without a valid key and id")
	}
	rr := do("k", "Bearer k", acct)
	if rr.Code != 200 || rr.Body.String() != "{\"rows\":3,\"service\":\"nn-friends\"}\n" {
		t.Fatalf("ok: %d %q", rr.Code, rr.Body.String())
	}
}

// The database test needs a Postgres (FRIENDS_TEST_DATABASE_URL) and is skipped
// without one, like the other integration surfaces.
func TestEraseAgainstPostgres(t *testing.T) {
	url := os.Getenv("FRIENDS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FRIENDS_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range []string{
		`CREATE SCHEMA IF NOT EXISTS wiiu`, `CREATE SCHEMA IF NOT EXISTS "3ds"`,
		`CREATE TABLE IF NOT EXISTS wiiu.user_data (pid integer PRIMARY KEY, comment text DEFAULT '')`,
		`CREATE TABLE IF NOT EXISTS wiiu.network_account_info (pid integer PRIMARY KEY, birthday bigint)`,
		`CREATE TABLE IF NOT EXISTS wiiu.principal_basic_info (pid integer PRIMARY KEY, username text)`,
		`CREATE TABLE IF NOT EXISTS wiiu.mii (pid integer PRIMARY KEY, name text, data bytea)`,
		`CREATE TABLE IF NOT EXISTS wiiu.friendships (id bigserial PRIMARY KEY, user1_pid integer, user2_pid integer, date bigint, active boolean, UNIQUE (user1_pid, user2_pid))`,
		`CREATE TABLE IF NOT EXISTS wiiu.blocks (id bigserial PRIMARY KEY, blocker_pid integer, blocked_pid integer, UNIQUE (blocker_pid, blocked_pid))`,
		`CREATE TABLE IF NOT EXISTS wiiu.friend_requests (id bigserial PRIMARY KEY, sender_pid integer, recipient_pid integer, message text)`,
		`CREATE TABLE IF NOT EXISTS "3ds".user_data (pid integer PRIMARY KEY, comment text DEFAULT '')`,
		`CREATE TABLE IF NOT EXISTS "3ds".friendships (id bigserial PRIMARY KEY, user1_pid integer, user2_pid integer, type integer, date bigint, UNIQUE (user1_pid, user2_pid))`,
		// Two accounts: pids 1900000001 (erased) and 1900000002 (bystander).
		`DELETE FROM wiiu.user_data WHERE pid IN (1900000001,1900000002)`,
		`DELETE FROM wiiu.friendships WHERE user1_pid IN (1900000001,1900000002) OR user2_pid IN (1900000001,1900000002)`,
		`DELETE FROM wiiu.blocks WHERE blocker_pid IN (1900000001,1900000002)`,
		`DELETE FROM wiiu.friend_requests WHERE sender_pid IN (1900000001,1900000002)`,
		`DELETE FROM "3ds".user_data WHERE pid IN (1900000001,1900000002)`,
		`INSERT INTO wiiu.user_data (pid) VALUES (1900000001),(1900000002)`,
		`INSERT INTO wiiu.friendships (user1_pid,user2_pid,active) VALUES (1900000002,1900000001,true)`,
		`INSERT INTO wiiu.blocks (blocker_pid,blocked_pid) VALUES (1900000002,1900000001)`,
		`INSERT INTO wiiu.friend_requests (sender_pid,recipient_pid) VALUES (1900000001,1900000002)`,
		`INSERT INTO "3ds".user_data (pid) VALUES (1900000001),(1900000002)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	e := &Eraser{DB: db, PIDs: func(context.Context, string) ([]uint32, error) { return []uint32{1900000001}, nil }}
	n, err := e.Erase(context.Background(), acct)
	if err != nil || n != 5 {
		t.Fatalf("first run: n=%d err=%v", n, err)
	}
	if n, err := e.Erase(context.Background(), acct); err != nil || n != 0 {
		t.Fatalf("second run must be 0 rows: n=%d err=%v", n, err)
	}
	var left int
	db.QueryRow(`SELECT count(*) FROM wiiu.user_data WHERE pid=1900000002`).Scan(&left)
	if left != 1 {
		t.Fatal("the other account's row must stay")
	}
	db.QueryRow(`SELECT count(*) FROM "3ds".user_data WHERE pid=1900000002`).Scan(&left)
	if left != 1 {
		t.Fatal("the other account's 3ds row must stay")
	}
}
