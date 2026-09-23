package globals

import (
	"strings"

	"github.com/PretendoNetwork/nex-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NEXPasswordError is the NEX error for a failed GetNEXPassword. nn-account
// refuses a banned (or deleting) owner with InvalidArgument "Account is banned
// or deleted"; that becomes RendezVous::AccountDisabled (0x00030067), the NEX
// result for a disabled account, so the console is told its account is
// banned. Anything else stays RendezVous::InvalidPID, the upstream answer.
func NEXPasswordError(err error) *nex.Error {
	if s, ok := status.FromError(err); ok && s.Code() == codes.InvalidArgument &&
		strings.Contains(strings.ToLower(s.Message()), "banned") {
		return nex.NewError(nex.ResultCodes.RendezVous.AccountDisabled, "Account is banned")
	}
	return nex.NewError(nex.ResultCodes.RendezVous.InvalidPID, "Invalid PID")
}
