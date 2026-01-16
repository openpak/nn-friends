package nex

import (
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2"
)

var serverBuildString string

func StartAuthenticationServer() {

	if globals.Config.HealthCheckPort != 0 {
		go nex.EnableBasicUDPHealthCheck(int(globals.Config.HealthCheckPort))
	}

	globals.AuthenticationServer = nex.NewPRUDPServer()
	globals.AuthenticationEndpoint = nex.NewPRUDPEndPoint(1)

	globals.AuthenticationEndpoint.ServerAccount = globals.AuthenticationServerAccount
	globals.AuthenticationEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	globals.AuthenticationEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername

	registerCommonAuthenticationServerProtocols()

	globals.AuthenticationServer.SetFragmentSize(962)
	globals.AuthenticationServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(1, 1, 0))
	globals.AuthenticationServer.SessionKeyLength = 16
	globals.AuthenticationServer.AccessKey = "ridfebb9"
	globals.AuthenticationServer.BindPRUDPEndPoint(globals.AuthenticationEndpoint)
	globals.AuthenticationServer.Listen(int(globals.Config.AuthenticationServerPort))
}
