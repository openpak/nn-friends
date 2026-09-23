package notifications_3ds

import (
	"github.com/PretendoNetwork/friends/globals"
	friends_types "github.com/PretendoNetwork/friends/types"
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
	nintendo_notifications "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications"
	nintendo_notifications_constants "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications/constants"
	nintendo_notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications/types"
)

// Cross-platform presence, one viewer at a time: the events a 3DS friend's
// presence change or departure produces, sent on behalf of a friend who is
// live on another platform per the account core.

// SendPresenceTo tells viewer that friendPID's presence is presence.
func SendPresenceTo(viewer *friends_types.ConnectedUser, friendPID uint32, presence friends_3ds_types.NintendoPresence) {
	eventObject := nintendo_notifications_types.NewNintendoNotificationEvent()
	eventObject.Type = nintendo_notifications_constants.NotificationTypeFriendPresenceUpdated3DS
	eventObject.SenderPID = types.NewPID(uint64(friendPID))
	eventObject.DataHolder = types.NewDataHolder()
	eventObject.DataHolder.Object = presence.Copy().(friends_3ds_types.NintendoPresence)
	sendEventTo(viewer, eventObject)
}

// SendWentOfflineTo tells viewer that friendPID went offline.
func SendWentOfflineTo(viewer *friends_types.ConnectedUser, friendPID uint32) {
	eventObject := nintendo_notifications_types.NewNintendoNotificationEvent()
	eventObject.Type = nintendo_notifications_constants.NotificationTypeFriendOffline
	eventObject.SenderPID = types.NewPID(uint64(friendPID))
	eventObject.DataHolder = types.NewDataHolder()
	eventObject.DataHolder.Object = nintendo_notifications_types.NewNintendoNotificationEventGeneral()
	sendEventTo(viewer, eventObject)
}

func sendEventTo(viewer *friends_types.ConnectedUser, event nintendo_notifications_types.NintendoNotificationEvent) {
	if viewer == nil || viewer.Connection == nil {
		return
	}
	stream := nex.NewByteStreamOut(globals.SecureEndpoint.LibraryVersions(), globals.SecureEndpoint.ByteStreamSettings())
	event.WriteTo(stream)

	request := nex.NewRMCRequest(globals.SecureEndpoint)
	request.ProtocolID = nintendo_notifications.ProtocolID
	request.CallID = 3810693103
	request.MethodID = nintendo_notifications.MethodProcessNintendoNotificationEvent1
	request.Parameters = stream.Bytes()

	packet, _ := nex.NewPRUDPPacketV0(globals.SecureEndpoint.Server, viewer.Connection, nil)
	packet.SetType(constants.DataPacket)
	packet.AddFlag(constants.PacketFlagNeedsAck)
	packet.AddFlag(constants.PacketFlagReliable)
	packet.SetSourceVirtualPortStreamType(viewer.Connection.StreamType)
	packet.SetSourceVirtualPortStreamID(globals.SecureEndpoint.StreamID)
	packet.SetDestinationVirtualPortStreamType(viewer.Connection.StreamType)
	packet.SetDestinationVirtualPortStreamID(viewer.Connection.StreamID)
	packet.SetPayload(request.Bytes())
	globals.SecureServer.Send(packet)
}
