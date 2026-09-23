package crosspresence

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/coregraph"
	"github.com/PretendoNetwork/nex-go/v2/types"
	account_management_types "github.com/PretendoNetwork/nex-protocols-go/v2/account-management/types"
	ticket_granting_types "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting/types"
)

// LoginToken is the NEX token a console or emulator logged in with: the one
// nn-account issued (NNAS nex_token for a Wii U, NASC LOGIN for a 3DS), or ""
// for any other login data.
func LoginToken(loginData types.DataHolder) string {
	switch d := loginData.Object.(type) {
	case ticket_granting_types.NintendoLoginData:
		return string(d.Token)
	case *ticket_granting_types.NintendoLoginData:
		return string(d.Token)
	case account_management_types.AccountExtraInfo:
		return string(d.NEXToken)
	case *account_management_types.AccountExtraInfo:
		return string(d.NEXToken)
	}
	return ""
}

// ClientOf is what somebody logged in from, as nn-account recorded it when it
// issued their NEX token: "wiiu"/"3ds" for a console, "cemu"/"azahar" for an
// emulator. "" when nobody can tell (no core, an old token, nn-account
// unreachable): the core then shows the platform, as it did before.
func ClientOf(loginData types.DataHolder) string {
	token := LoginToken(loginData)
	if token == "" || !coregraph.Configured() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := coregraph.C().ClientOfNEXToken(ctx, token)
	if err != nil {
		return ""
	}
	return client
}
