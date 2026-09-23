package globals

import (
	"errors"
	"testing"

	"github.com/PretendoNetwork/nex-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNEXPasswordError(t *testing.T) {
	const errorBit = 0x80000000
	cases := []struct {
		err  error
		want uint32
	}{
		{status.Error(codes.InvalidArgument, "Account is banned or deleted"), nex.ResultCodes.RendezVous.AccountDisabled},
		{status.Error(codes.InvalidArgument, "No NEX account found"), nex.ResultCodes.RendezVous.InvalidPID},
		{status.Error(codes.Internal, "Core account service unavailable"), nex.ResultCodes.RendezVous.InvalidPID},
		{errors.New("dial tcp: refused"), nex.ResultCodes.RendezVous.InvalidPID},
	}
	for _, c := range cases {
		if got := NEXPasswordError(c.err).ResultCode; got != c.want|errorBit {
			t.Errorf("%v: got %#x, want %#x", c.err, got, c.want|errorBit)
		}
	}
}

func TestOnlinePIDsOfAccount(t *testing.T) {
	connectedAccountsMu.Lock()
	connectedAccounts["acct-1"] = 20
	connectedPIDs[10], connectedPIDs[20], connectedPIDs[30] = "acct-1", "acct-1", "acct-2"
	connectedAccountsMu.Unlock()
	t.Cleanup(func() {
		for _, pid := range []uint32{10, 20, 30} {
			MarkAccountOffline(pid)
		}
	})
	got := OnlinePIDsOfAccount("acct-1")
	if len(got) != 2 || (got[0] != 10 && got[1] != 10) {
		t.Fatalf("got %v, want both of acct-1's consoles", got)
	}
}
