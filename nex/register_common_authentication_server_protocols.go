package nex

import (
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

func registerCommonAuthenticationServerProtocols() {
	ticketGrantingProtocol := ticket_granting.NewProtocol()
	commonTicketGrantingProtocol := common_ticket_granting.NewCommonProtocol(ticketGrantingProtocol)

	secureStationURL := types.NewStationURL("")
	secureStationURL.SetURLType(constants.StationURLPRUDPS)
	secureStationURL.SetAddress(globals.Config.SecureServerHost)
	secureStationURL.SetPortNumber(globals.Config.SecureServerPort)
	secureStationURL.SetConnectionID(1)
	secureStationURL.SetPrincipalID(types.NewPID(2))
	secureStationURL.SetStreamID(1)
	secureStationURL.SetStreamType(constants.StreamTypeRVSecure)
	secureStationURL.SetType(2)

	// The secure server starts in its own goroutine, so its endpoint may not
	// exist yet; the account it serves is set in init.
	commonTicketGrantingProtocol.SecureServerAccount = globals.SecureServerAccount
	commonTicketGrantingProtocol.SessionKeyLength = 16
	commonTicketGrantingProtocol.SecureStationURL = secureStationURL
	commonTicketGrantingProtocol.BuildName = types.NewString(serverBuildString)
	commonTicketGrantingProtocol.EnableInsecureLogin()

	globals.AuthenticationEndpoint.RegisterServiceProtocol(ticketGrantingProtocol)
}
