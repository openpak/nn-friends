package nex

import (
	"github.com/PretendoNetwork/friends/globals"
	nex_secure_connection "github.com/PretendoNetwork/friends/nex/secure-connection"
	common_secure_connection "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
)

func registerCommonSecureServerProtocols() {
	secureConnectionProtocol := secure_connection.NewProtocol()
	commonSecureConnectionProtocol := common_secure_connection.NewCommonProtocol(secureConnectionProtocol)

	// * On account creation the console logs in with a guest account and uses the Register method
	commonSecureConnectionProtocol.EnableInsecureRegister()

	// * Override RegisterEx so that we can register the user into the ConnectedUsers map
	secureConnectionProtocol.RegisterEx = nex_secure_connection.RegisterEx

	globals.SecureEndpoint.RegisterServiceProtocol(secureConnectionProtocol)
}
