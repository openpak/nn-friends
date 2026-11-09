package crosspresence

import (
	"log"
	"os"
	"strconv"

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

// LogJoin logs the join fields of a presence a console published (hosting a game): ids and flags only, never names.
func LogJoin(pid uint32, p friends_wiiu_types.NintendoPresenceV2) {
	if p.GameServerID == 0 && p.GatheringID == 0 {
		return
	}
	log.Printf("[presence] pid=%d title=%016x/%d joinFlagMask=%d matchmakeType=%d joinGameId=%d joinGameMode=%d hostPid=%d groupId=%d appdata=%x",
		pid, uint64(p.GameKey.TitleID), uint16(p.GameKey.TitleVersion), uint32(p.Unknown2), uint8(p.Unknown3), uint32(p.GameServerID),
		uint32(p.Unknown4), uint64(p.PID), uint32(p.GatheringID), []byte(p.ApplicationData))
}

// WorldLine is the line for a friend who is in the shared Minecraft world (common/minecraft publishes them to the core as the
// "minecraft" namespace). A Wii U viewer who is itself playing Minecraft: Wii U Edition gets a joinable presence instead
// (WiiUPresenceFor), which is what the game's Join tab is built from.
const WorldLine = "Playing Minecraft in the OpenPak world"

// minecraftTitles are the Minecraft: Wii U Edition title ids (USA, EUR, JPN).
var minecraftTitles = map[uint64]bool{0x00050000101D9D00: true, 0x00050000101D7500: true, 0x00050000101DBE00: true}

// What a Minecraft: Wii U Edition host publishes while hosting an online game (card #162). Read from nn-friends' own log of a
// real game (Minecraft EUR v688 hosting a new online world, 2026-11-09): joinFlagMask 1, matchmakeType 2, joinGameId the game's
// id (0x101D9D00, the same on the EUR title), joinGameMode 1, hostPid the host's own PID (the presence PID, which the friend notification is addressed by too), groupId the gathering id, and 20 bytes of
// application data (net version 0x0881, game mode 15).
const (
	worldJoinFlagMask  = 1
	worldMatchmakeType = 2
	worldGameID        = 0x101D9D00
	worldJoinGameMode  = 1
)

var worldAppData = []byte{0, 0, 0, 0, 0x08, 0x81, 0, 0, 0x0f, 0, 0x5c, 0x70, 0, 0, 0, 0, 0, 0, 0, 0}

// worldGathering is the world's gathering id in the Minecraft nex title (MINECRAFT_WORLD_GATHERING); 0 turns the join presence off.
func worldGathering() uint32 {
	n, _ := strconv.ParseUint(os.Getenv("MINECRAFT_WORLD_GATHERING"), 10, 32)
	return uint32(n)
}

// WiiUPresenceFor is the presence a Wii U viewer sees for a friend whose core line is line. For a friend in the Minecraft world,
// and a viewer who runs Minecraft: Wii U Edition, it carries the viewer's own game key and the world's gathering as the game
// to join; every other case is the plain line.
func WiiUPresenceFor(viewer friends_wiiu_types.NintendoPresenceV2, friendPID uint32, line string) friends_wiiu_types.NintendoPresenceV2 {
	p := WiiUPresence(friendPID, line)
	gid := worldGathering()
	if line != WorldLine || gid == 0 || !minecraftTitles[uint64(viewer.GameKey.TitleID)] {
		return p
	}
	p.GameKey = viewer.GameKey.Copy().(friends_wiiu_types.GameKey)
	p.Unknown2 = types.NewUInt32(worldJoinFlagMask)
	p.Unknown3 = types.NewUInt8(worldMatchmakeType)
	p.GameServerID = types.NewUInt32(worldGameID)
	p.Unknown4 = types.NewUInt32(worldJoinGameMode)
	p.GatheringID = types.NewUInt32(gid) // the group id is the gathering id
	p.ApplicationData = types.NewBuffer(worldAppData)
	return p
}

// InWorldKey is the viewer's game key when it runs Minecraft: Wii U Edition (0 otherwise); a change of it means the viewer's
// friends in the world must be sent again, joinable or not.
func InWorldKey(viewer friends_wiiu_types.NintendoPresenceV2) uint64 {
	if t := uint64(viewer.GameKey.TitleID); minecraftTitles[t] {
		return t
	}
	return 0
}
