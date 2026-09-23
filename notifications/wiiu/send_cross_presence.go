package notifications_wiiu

import (
	"github.com/PretendoNetwork/friends/globals"
	friends_types "github.com/PretendoNetwork/friends/types"
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
	nintendo_notifications "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications"
	nintendo_notifications_constants "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications/constants"
	nintendo_notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/nintendo-notifications/types"
)

// Cross-platform presence: a friend live on another platform (per the account
// core) is told to one viewer at a time, with the same events a Wii U friend
// going online or offline produces.

// SendPresenceTo tells viewer that presence.PID's presence is presence.
func SendPresenceTo(viewer *friends_types.ConnectedUser, presence friends_wiiu_types.NintendoPresenceV2) {
	eventObject := nintendo_notifications_types.NewNintendoNotificationEvent()
	eventObject.Type = nintendo_notifications_constants.NotificationTypeFriendStartedTitleWiiU
	eventObject.SenderPID = presence.PID.Copy().(types.PID)
	eventObject.DataHolder = types.NewDataHolder()
	eventObject.DataHolder.Object = presence.Copy().(friends_wiiu_types.NintendoPresenceV2)
	sendEventTo(viewer, eventObject, nintendo_notifications.MethodProcessNintendoNotificationEvent2)
}

// SendWentOfflineTo tells viewer that friendPID went offline.
func SendWentOfflineTo(viewer *friends_types.ConnectedUser, friendPID uint32) {
	general := nintendo_notifications_types.NewNintendoNotificationEventGeneral()
	general.U32Param = types.NewUInt32(0)
	general.U64Param1 = types.NewUInt64(0)
	general.U64Param2 = types.NewUInt64(uint64(types.NewDateTime(0).Now()))
	general.StrParam = types.NewString("")
	eventObject := nintendo_notifications_types.NewNintendoNotificationEvent()
	eventObject.Type = nintendo_notifications_constants.NotificationTypeFriendOffline
	eventObject.SenderPID = types.NewPID(uint64(friendPID))
	eventObject.DataHolder = types.NewDataHolder()
	eventObject.DataHolder.Object = general.Copy().(nintendo_notifications_types.NintendoNotificationEventGeneral)
	sendEventTo(viewer, eventObject, nintendo_notifications.MethodProcessNintendoNotificationEvent1)
}

func sendEventTo(viewer *friends_types.ConnectedUser, event nintendo_notifications_types.NintendoNotificationEvent, method uint32) {
	if viewer == nil || viewer.Connection == nil {
		return
	}
	stream := nex.NewByteStreamOut(globals.SecureEndpoint.LibraryVersions(), globals.SecureEndpoint.ByteStreamSettings())
	event.WriteTo(stream)

	request := nex.NewRMCRequest(globals.SecureEndpoint)
	request.ProtocolID = nintendo_notifications.ProtocolID
	request.CallID = 3810693103
	request.MethodID = method
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
