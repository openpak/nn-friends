package types

import (
	"github.com/PretendoNetwork/nex-go/v2"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

type ConnectedUser struct {
	PID        uint32
	Platform   Platform
	Connection *nex.PRUDPConnection
	Presence   friends_3ds_types.NintendoPresence
	PresenceV2 friends_wiiu_types.NintendoPresenceV2
	// Client is what they logged in from ("wiiu", "3ds", "cemu", "azahar"),
	// as nn-account recorded it with their NEX token; "" when unknown.
	Client string
	// OS is what the emulator said it runs on ("windows", "macos", "linux",
	// "android", "ios"), recorded with the client; "" for a console.
	OS string
}

func NewConnectedUser() *ConnectedUser {
	return &ConnectedUser{
		Presence:   friends_3ds_types.NewNintendoPresence(),
		PresenceV2: friends_wiiu_types.NewNintendoPresenceV2(),
	}
}
