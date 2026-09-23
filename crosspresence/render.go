package crosspresence

import (
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_constants "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/constants"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
	friends_wiiu_constants "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/constants"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

// WiiUPresence is a friend live on another platform, as a Wii U draws it:
// online, no game key (a foreign title id would name a game the console does
// not have, or the wrong one), and the words in the game-mode message. The
// shape follows the fixed friend upstream shipped for testing ("bella"):
// online, zero game key, a message, the same changed flags. Whether the Wii U
// friend list shows the message for a friend with no game is NOT verified on
// hardware.
func WiiUPresence(pid uint32, message string) friends_wiiu_types.NintendoPresenceV2 {
	p := friends_wiiu_types.NewNintendoPresenceV2()
	p.ChangedFlags = friends_wiiu_constants.PresenceChangedFlagApplicationData |
		friends_wiiu_constants.PresenceChangedFlagGatheringID |
		friends_wiiu_constants.PresenceChangedFlagOwnerPID |
		friends_wiiu_constants.PresenceChangedFlagJoinGameMode |
		friends_wiiu_constants.PresenceChangedFlagMatchmakeSystemType |
		friends_wiiu_constants.PresenceChangedFlagJoinAvailabilityFlag |
		friends_wiiu_constants.PresenceChangedFlagGameModeDescription
	p.Online = types.NewBool(true)
	p.Message = types.NewString(message)
	p.PID = types.NewPID(uint64(pid))
	p.ApplicationData = types.NewBuffer([]byte{0x0})
	return p
}

// ThreeDSPresence is the 3DS form: an empty game key with every flag set (the
// shape UpdatePresence already sends for "not showing my game"), and the words
// in the game-mode description. Unverified on hardware whether the 3DS friend
// list shows that description without a game.
func ThreeDSPresence(message string) friends_3ds_types.NintendoPresence {
	p := friends_3ds_types.NewNintendoPresence()
	p.ChangedFlags = friends_3ds_constants.PresenceChangedFlag(0xFFFFFFFF)
	p.Message = types.NewString(message)
	return p
}
