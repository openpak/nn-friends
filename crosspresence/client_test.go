package crosspresence

import (
	"context"
	"net"
	"testing"

	"github.com/PretendoNetwork/friends/coregraph"
	resolutionv1 "github.com/PretendoNetwork/friends/internal/resolutionpb"
	"github.com/PretendoNetwork/nex-go/v2/types"
	account_management_types "github.com/PretendoNetwork/nex-protocols-go/v2/account-management/types"
	ticket_granting_types "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func wiiuLogin(token string) types.DataHolder {
	d := types.NewDataHolder()
	l := ticket_granting_types.NewNintendoLoginData()
	l.Token = types.NewString(token)
	d.Object = l
	return d
}

func ctrLogin(token string) types.DataHolder {
	d := types.NewDataHolder()
	a := account_management_types.NewAccountExtraInfo()
	a.NEXToken = types.NewString(token)
	d.Object = a
	return d
}

func TestLoginToken(t *testing.T) {
	if got := LoginToken(wiiuLogin("wiiu-token")); got != "wiiu-token" {
		t.Fatalf("Wii U login data: %q", got)
	}
	if got := LoginToken(ctrLogin("ctr-token")); got != "ctr-token" {
		t.Fatalf("3DS login data: %q", got)
	}
	if got := LoginToken(types.NewDataHolder()); got != "" {
		t.Fatalf("empty holder: %q", got)
	}
}

// fakeResolution is nn-account's Resolution service with a token table.
type fakeResolution struct {
	resolutionv1.UnimplementedResolutionServer
	clients map[string]string
}

func (f *fakeResolution) ResolveNexTokenClient(ctx context.Context, r *resolutionv1.ResolveNexTokenClientRequest) (*resolutionv1.ResolveNexTokenClientResponse, error) {
	if md, _ := metadata.FromIncomingContext(ctx); len(md.Get("x-api-key")) == 0 || md.Get("x-api-key")[0] != "adapter-key" {
		return &resolutionv1.ResolveNexTokenClientResponse{}, nil
	}
	c, ok := f.clients[r.GetToken()]
	return &resolutionv1.ResolveNexTokenClientResponse{Found: ok, Client: c}, nil
}

// The whole path on this side: login data -> token -> nn-account -> the
// client the publisher sends to the core.
func TestClientOfAsksNNAccount(t *testing.T) {
	if !coregraph.Configured() {
		if got := ClientOf(wiiuLogin("anything")); got != "" {
			t.Fatalf("no core configured: %q", got)
		}
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	resolutionv1.RegisterResolutionServer(srv, &fakeResolution{clients: map[string]string{
		"console-token": "wiiu", "cemu-token": "cemu", "azahar-token": "azahar", "old-token": "",
	}})
	go srv.Serve(lis)
	defer srv.Stop()
	if err := coregraph.Init("127.0.0.1:1", "core-key", lis.Addr().String(), "adapter-key"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		login types.DataHolder
		want  string
	}{
		{wiiuLogin("console-token"), "wiiu"},
		{wiiuLogin("cemu-token"), "cemu"},
		{ctrLogin("azahar-token"), "azahar"},
		{wiiuLogin("old-token"), ""},
		{wiiuLogin("unknown-token"), ""},
	}
	for _, c := range cases {
		if got := ClientOf(c.login); got != c.want {
			t.Errorf("ClientOf(%s) = %q, want %q", LoginToken(c.login), got, c.want)
		}
	}
}

func TestPublisherCarriesClient(t *testing.T) {
	core := newFakeCore()
	p := &Publisher{Core: core, AccountOf: accountOf}
	ctx := context.Background()

	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu", Client: "cemu"}})
	if r := core.only(t); r.GetClient() != "cemu" || r.GetNamespace() != "wiiu" {
		t.Fatalf("client not carried: %+v", r)
	}
	// Same person, now from the console: the session is replaced.
	p.Reconcile(ctx, []Online{{PID: 7, Namespace: "wiiu", Client: "wiiu"}})
	if r := core.only(t); r.GetClient() != "wiiu" {
		t.Fatalf("client change not carried: %+v", r)
	}
}
