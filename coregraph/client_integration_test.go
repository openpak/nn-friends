//go:build integration

// End-to-end: friends' coregraph against live account core + nn-account
// adapter subprocesses (M3 shared-graph milestone).
package coregraph_test

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
)

const (
	coreDB     = "postgres://postgres:test@127.0.0.1:54329/friends_core_test?sslmode=disable"
	adapterDB  = "postgres://postgres:test@127.0.0.1:54329/friends_adapter_test?sslmode=disable"
	coreKey    = "friends-core-key-0123456789abcdef0123"
	adapterKey = "friends-adapter-key-0123456789abcdef012"
)

func resetDB(t *testing.T, url string) {
	t.Helper()
	// psql is unavailable here; use the adapter/core binaries' own startup
	// on a freshly created database instead.
}

func startStack(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(thisFile)) // friends repo root

	reset := func(db string) {
		drop := exec.Command("docker", "exec", "openpak-pg-test", "psql", "-U", "postgres",
			"-c", "DROP DATABASE IF EXISTS "+db, "-c", "CREATE DATABASE "+db)
		if out, err := drop.CombinedOutput(); err != nil {
			t.Fatalf("reset %s: %v\n%s", db, err, out)
		}
	}
	reset("friends_core_test")
	reset("friends_adapter_test")

	// Build both binaries.
	binDir := t.TempDir()
	coreBin := filepath.Join(binDir, "account-core")
	coreBuild := exec.Command("go", "build", "-buildvcs=false", "-o", coreBin, "openpak/account/cmd/account")
	coreBuild.Dir = filepath.Join(root, "..", "account")
	if out, err := coreBuild.CombinedOutput(); err != nil {
		t.Fatalf("build core: %v\n%s", err, out)
	}
	build := exec.Command("go", "build", "-buildvcs=false", "-o", filepath.Join(binDir, "nn-account"), "./cmd/nn-account")
	build.Dir = filepath.Join(root, "..", "nn-account")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v\n%s", err, out)
	}

	// Start core.
	core := exec.Command(coreBin, "x")
	core = exec.Command(coreBin)
	core.Env = append(os.Environ(),
		"ACCOUNT_DATABASE_URL="+coreDB,
		"ACCOUNT_SESSION_SECRET=0123456789abcdef0123456789abcdef",
		"ACCOUNT_INTERNAL_KEY="+coreKey,
		"ACCOUNT_ENVIRONMENT=development",
		"ACCOUNT_GRPC_ADDR=127.0.0.1:17171",
		"ACCOUNT_HTTP_ADDR=127.0.0.1:17181",
	)
	core.Stderr = os.Stderr
	if err := core.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = core.Process.Kill(); _, _ = core.Process.Wait() })

	// Wait for the core BEFORE starting the adapter: the adapter dials the
	// core lazily, and a startup race surfaces as "connection refused" on
	// the first registration.
	waitHTTP(t, "http://127.0.0.1:17181/readyz", 30*time.Second)

	// Start adapter.
	adapter := exec.Command(filepath.Join(binDir, "nn-account"))
	adapter.Env = append(os.Environ(),
		"NN_ACCOUNT_DATABASE_URL="+adapterDB,
		"NN_ACCOUNT_CORE_ADDR=127.0.0.1:17171",
		"NN_ACCOUNT_CORE_KEY="+coreKey,
		"NN_ACCOUNT_GRPC_API_KEY="+adapterKey,
		"NN_ACCOUNT_GRPC_ADDR=127.0.0.1:17191",
		"NN_ACCOUNT_HTTP_ADDR=127.0.0.1:17201",
	)
	adapter.Stderr = os.Stderr
	if err := adapter.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adapter.Process.Kill(); _, _ = adapter.Process.Wait() })

	waitHTTP(t, "http://127.0.0.1:17201/healthz", 15*time.Second)
}

func waitHTTP(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("service at %s did not become ready within %v", url, timeout)
}

func registerConsoleUser(t *testing.T, username string) uint32 {
	t.Helper()
	body := strings.NewReader(strings.Join([]string{
		"user_id=" + username,
		"password=consoleSecret99",
		"email.address=" + username + "@example.com",
		"mii.name=" + username,
		"mii.data=AAAA",
		"country=US", "language=en", "region=1", "tz_name=EST5EDT",
	}, "&"))
	req, _ := http.NewRequest("POST", "http://127.0.0.1:17201/v1/api/people", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	xml := string(buf[:n])
	start := strings.Index(xml, "<pid>") + len("<pid>")
	end := strings.Index(xml, "</pid>")
	if start < len("<pid>") || end < 0 {
		t.Fatalf("registration failed: %s", xml)
	}
	var pid uint32
	for _, c := range xml[start:end] {
		pid = pid*10 + uint32(c-'0')
	}
	return pid
}

func TestCoreGraphIntegration(t *testing.T) {
	startStack(t)

	if err := coregraph.Init("127.0.0.1:17171", coreKey, "127.0.0.1:17191", adapterKey); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	alice := registerConsoleUser(t, "graphalice")
	bob := registerConsoleUser(t, "graphbob")

	// Resolution both ways.
	aAccount, err := coregraph.C().AccountOfPID(ctx, "wiiu", alice)
	if err != nil || aAccount == "" {
		t.Fatalf("resolve alice: %v", err)
	}
	bPID, err := coregraph.C().PIDOfAccount(ctx, "wiiu", aAccount)
	if err != nil || bPID != alice {
		t.Fatalf("reverse resolve: %d %v", bPID, err)
	}

	// Unknown PID does not resolve.
	if _, err := coregraph.C().AccountOfPID(ctx, "wiiu", 1799999998); err == nil {
		t.Fatal("unknown PID must not resolve")
	}

	// One-sided request → incomplete; mutual → complete.
	state, err := coregraph.C().Request(ctx, "wiiu", alice, bob)
	if err != nil || state != coregraph.StateIncomplete {
		t.Fatalf("request: %v %v", state, err)
	}
	state, err = coregraph.C().Request(ctx, "wiiu", bob, alice)
	if err != nil || state != coregraph.StateComplete {
		t.Fatalf("mutual request: %v %v", state, err)
	}

	// Friend lists agree both directions.
	aliceFriends, err := coregraph.C().FriendPIDs(ctx, "wiiu", alice)
	if err != nil || len(aliceFriends) != 1 || aliceFriends[0] != bob {
		t.Fatalf("alice friends: %v %v", aliceFriends, err)
	}
	bobFriends, err := coregraph.C().FriendPIDs(ctx, "wiiu", bob)
	if err != nil || len(bobFriends) != 1 || bobFriends[0] != alice {
		t.Fatalf("bob friends: %v %v", bobFriends, err)
	}

	// Incoming/outgoing pending sets.
	in, err := coregraph.C().IncomingRequesterPIDs(ctx, "wiiu", bob)
	_ = in
	out, err := coregraph.C().OutgoingAddresseePIDs(ctx, "wiiu", alice)
	_ = out
	// (After completion there are no pendings; send a fresh pair to check.)
	carol := registerConsoleUser(t, "graphcarol")
	if _, err := coregraph.C().Request(ctx, "wiiu", carol, alice); err != nil {
		t.Fatal(err)
	}
	in, err = coregraph.C().IncomingRequesterPIDs(ctx, "wiiu", alice)
	if err != nil || len(in) != 1 || in[0] != carol {
		t.Fatalf("incoming: %v %v", in, err)
	}

	// Blocks are core-authoritative.
	if err := coregraph.C().Block(ctx, "wiiu", alice, bob); err != nil {
		t.Fatal(err)
	}
	blocked, err := coregraph.C().IsBlocked(ctx, "wiiu", alice, bob)
	if err != nil || !blocked {
		t.Fatalf("block: %v %v", blocked, err)
	}
	if err := coregraph.C().Unblock(ctx, "wiiu", alice, bob); err != nil {
		t.Fatal(err)
	}
	if blocked, _ := coregraph.C().IsBlocked(ctx, "wiiu", alice, bob); blocked {
		t.Fatal("unblock failed")
	}

	// Removal ends the friendship from either side.
	if err := coregraph.C().Remove(ctx, "wiiu", bob, alice); err != nil {
		t.Fatal(err)
	}
	aliceFriends, _ = coregraph.C().FriendPIDs(ctx, "wiiu", alice)
	if len(aliceFriends) != 0 {
		t.Fatalf("removal failed: %v", aliceFriends)
	}
}
