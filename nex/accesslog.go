package nex

import (
	"log"

	nex "github.com/PretendoNetwork/nex-go/v2"
)

// logCalls writes one line per NEX call received: server, protocol id and method id. Never the call
// parameters, which carry user data, tickets and tokens. The reply is sent by the protocol handler,
// so the line records receipt, not the outcome.
func logCalls(server string, endpoint *nex.PRUDPEndPoint) {
	endpoint.OnData(func(packet nex.PacketInterface) {
		if m := packet.RMCMessage(); m != nil {
			log.Printf("nex %s call %#x/%d -> recv", server, m.ProtocolID, m.MethodID)
		}
	})
}
