package crosspresence

import (
	"log"
	"os"
	"strconv"
	"strings"

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
	log.Printf("[presence] pid=%d title=%016x/%d joinFlagMask=%d matchmakeType=%d joinGameId=%d joinGameMode=%d hostPid=%d groupId=%d appdata=%x"+
		" changedFlags=%#x online=%t unknown1=%d unknown5=%d unknown6=%d unknown7=%d messageRunes=%d",
		pid, uint64(p.GameKey.TitleID), uint16(p.GameKey.TitleVersion), uint32(p.Unknown2), uint8(p.Unknown3), uint32(p.GameServerID),
		uint32(p.Unknown4), uint64(p.PID), uint32(p.GatheringID), []byte(p.ApplicationData),
		uint32(p.ChangedFlags), bool(p.Online), uint8(p.Unknown1), uint8(p.Unknown5), uint8(p.Unknown6), uint8(p.Unknown7), len([]rune(string(p.Message))))
}

// WorldLine is the line for a friend who is in the shared Minecraft world (common/minecraft publishes them to the core as the
// "minecraft" namespace). A Wii U viewer who is itself playing Minecraft: Wii U Edition gets a joinable presence instead
// (WiiUPresenceFor), which is what the game's Join tab is built from.
const WorldLine = "Playing Minecraft in the OpenPak world"

// minecraftTitles are the Minecraft: Wii U Edition title ids (USA, EUR, JPN).
var minecraftTitles = map[uint64]bool{0x00050000101D9D00: true, 0x00050000101D7500: true, 0x00050000101DBE00: true}

// What a Minecraft: Wii U Edition host publishes while hosting an online game (card #162). Read from nn-friends' own log of a
// real game (Minecraft EUR v688 hosting a new online world, 2026-11-09): joinFlagMask 1, matchmakeType 2, joinGameId the game's
// id (0x101D9D00, the same on the EUR title), joinGameMode 1, hostPid the gathering owner, groupId the gathering id, and 20 bytes of
// application data (net version 0x0881, game mode 15).
const (
	worldJoinFlagMask  = 1
	worldMatchmakeType = 2
	worldGameID        = 0x101D9D00
	worldJoinGameMode  = 1
)

var worldAppData = []byte{0, 0, 0, 0, 0x08, 0x81, 0, 0, 0x0f, 0, 0x5c, 0x70, 0, 0, 0, 0, 0, 0, 0, 0}

// hostingAppData is the application data a real Wii U host published while hosting an open game, read from nn-friends' log of the
// owner's console (WiiUTest, Minecraft EUR v688, 2026-11-11 15:54:52): 00000000 0881 0001 0f00 fa98 and zeros. The world's older reading
// (5c70 at bytes 10-11, 0000 at bytes 6-7) was a different session. Known: bytes 4-5 net version 0x0881, byte 8 game mode 15. NOT known: what
// bytes 6-7 (0 then 1) and 10-11 (5c70, fa98) mean; the first looks like a count or flag, the second like a per-session token. The newest
// reading is used for both, so the two values differ from the world's.
var hostingAppData = []byte{0, 0, 0, 0, 0x08, 0x81, 0, 0x01, 0x0f, 0, 0xfa, 0x98, 0, 0, 0, 0, 0, 0, 0, 0}

// worldPID is the gathering owner the service stands in for (MINECRAFT_WORLD_PID, nex's default 2000000001): the game checks the
// host PID of a friend's presence against the owner of the gathering it joins.
func worldPID() uint64 {
	if n, err := strconv.ParseUint(os.Getenv("MINECRAFT_WORLD_PID"), 10, 32); err == nil && n != 0 {
		return n
	}
	return 2000000001
}

// worldGathering is the world's gathering id in the Minecraft nex title (MINECRAFT_WORLD_GATHERING); 0 turns the join presence off.
func worldGathering() uint32 {
	n, _ := strconv.ParseUint(os.Getenv("MINECRAFT_WORLD_GATHERING"), 10, 32)
	return uint32(n)
}

// A friend who hosts a Minecraft game on another console (card #177): the session directory publishes them to the core as
// minecraft/g<gathering>, where the gathering is the one nex minecraft-wiiu lists for that game, owned by the friend's own Wii U PID. The
// gathering id travels in the line, so the line stays the only thing that crosses from the core to the console's presence.
const gameLinePrefix = "Playing Minecraft in a friend's game #"

// GameLine is the line for a friend whose own Minecraft game is gathering gid.
func GameLine(gid uint32) string { return gameLinePrefix + strconv.FormatUint(uint64(gid), 10) }

// bridgedGatheringOf reads the gathering id out of a minecraft title id of the form g<gathering>.
func bridgedGatheringOf(title string) (uint32, bool) {
	rest, ok := strings.CutPrefix(title, "g")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(rest, 10, 32)
	return uint32(n), err == nil && n != 0
}

func gameOfLine(line string) (uint32, bool) {
	rest, ok := strings.CutPrefix(line, gameLinePrefix)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(rest, 10, 32)
	return uint32(n), err == nil && n != 0
}

// WiiUPresenceFor is the presence a Wii U viewer sees for a friend whose core line is line. For a friend in the Minecraft world,
// and a viewer who runs Minecraft: Wii U Edition, it carries the viewer's own game key and the world's gathering as the game
// to join; every other case is the plain line.
func WiiUPresenceFor(viewer friends_wiiu_types.NintendoPresenceV2, friendPID uint32, line string) friends_wiiu_types.NintendoPresenceV2 {
	p := WiiUPresence(friendPID, line)
	gid, host := worldGathering(), worldPID()
	g, gameLine := gameOfLine(line)
	if inMinecraft := minecraftTitles[uint64(viewer.GameKey.TitleID)]; inMinecraft {
		// A console applies only the fields a notification flags as changed. The plain presence (offline of a game, or after one ended) must
		// clear the game key and the join game id, and the joinable one must set them, so a Minecraft viewer always gets both flagged.
		p.ChangedFlags |= friends_wiiu_constants.PresenceChangedFlagGameKey | friends_wiiu_constants.PresenceChangedFlagGameServerID
	}
	if gameLine {
		// the friend's own game: its gathering, owned by the friend's own PID
		gid, host = g, uint64(friendPID)
	} else if line != WorldLine {
		return p
	}
	if gid == 0 || !minecraftTitles[uint64(viewer.GameKey.TitleID)] {
		return p
	}
	p.GameKey = viewer.GameKey.Copy().(friends_wiiu_types.GameKey)
	p.Unknown2 = types.NewUInt32(worldJoinFlagMask)
	p.Unknown3 = types.NewUInt8(worldMatchmakeType)
	p.GameServerID = types.NewUInt32(worldGameID)
	p.Unknown4 = types.NewUInt32(worldJoinGameMode)
	p.GatheringID = types.NewUInt32(gid) // the group id is the gathering id
	p.PID = types.NewPID(host)           // the host PID; notifications are addressed by the friend, see SendPresenceTo
	p.ApplicationData = types.NewBuffer(worldAppData)
	if gameLine {
		p.ApplicationData = types.NewBuffer(hostingAppData)
	}
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
