package grpc

import (
	"context"
	"database/sql"

	"github.com/PretendoNetwork/friends/database"
	database_3ds "github.com/PretendoNetwork/friends/database/3ds"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	pb "github.com/PretendoNetwork/grpc/go/friends/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	friends_3ds_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-3ds/types"
	friends_wiiu_types "github.com/PretendoNetwork/nex-protocols-go/v2/friends-wiiu/types"
)

func (s *gRPCFriendsV2Server) GetUserFriendsDataWiiU(ctx context.Context, in *pb.GetUserFriendsDataRequest) (*pb.GetUserFriendsDataWiiUResponse, error) {
	var friends []*pb.FriendInfoWiiU
	friendList, err := database_wiiu.GetUserFriendList(in.Pid)
	if err != nil && err != database.ErrEmptyList {
		return &pb.GetUserFriendsDataWiiUResponse{
			Friends: friends,
		}, nil
	}

	if globals.Config.EnableBella {
		bella := friends_wiiu_types.NewFriendInfo()

		bella.NNAInfo = friends_wiiu_types.NewNNAInfo()
		bella.Presence = friends_wiiu_types.NewNintendoPresenceV2()
		bella.Status = friends_wiiu_types.NewComment()
		bella.BecameFriend = types.NewDateTime(0)
		bella.LastOnline = types.NewDateTime(0)
		bella.Unknown = types.NewUInt64(0)

		bella.NNAInfo.PrincipalBasicInfo = friends_wiiu_types.NewPrincipalBasicInfo()
		bella.NNAInfo.Unknown1 = types.NewUInt8(0)
		bella.NNAInfo.Unknown2 = types.NewUInt8(0)

		bella.NNAInfo.PrincipalBasicInfo.PID = types.NewPID(1743126339)
		bella.NNAInfo.PrincipalBasicInfo.NNID = types.NewString("PN_Testing")
		bella.NNAInfo.PrincipalBasicInfo.Mii = friends_wiiu_types.NewMiiV2()
		bella.NNAInfo.PrincipalBasicInfo.Unknown = types.NewUInt8(0)

		bella.NNAInfo.PrincipalBasicInfo.Mii.Name = types.NewString("Bandwidth")
		bella.NNAInfo.PrincipalBasicInfo.Mii.Unknown1 = types.NewUInt8(0)
		bella.NNAInfo.PrincipalBasicInfo.Mii.Unknown2 = types.NewUInt8(0)
		bella.NNAInfo.PrincipalBasicInfo.Mii.MiiData = types.NewBuffer([]byte{
			0x03, 0x00, 0x00, 0x40, 0xE9, 0x55, 0xA2, 0x09,
			0xE7, 0xC7, 0x41, 0x82, 0xD9, 0x7D, 0x0B, 0x2D,
			0x03, 0xB3, 0xB8, 0x8D, 0x27, 0xD9, 0x00, 0x00,
			0x01, 0x40, 0x62, 0x00, 0x65, 0x00, 0x6C, 0x00,
			0x6C, 0x00, 0x61, 0x00, 0x00, 0x00, 0x45, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x40,
			0x12, 0x00, 0x81, 0x01, 0x04, 0x68, 0x43, 0x18,
			0x20, 0x34, 0x46, 0x14, 0x81, 0x12, 0x17, 0x68,
			0x0D, 0x00, 0x00, 0x29, 0x03, 0x52, 0x48, 0x50,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFE, 0x86,
		})
		bella.NNAInfo.PrincipalBasicInfo.Mii.Datetime = types.NewDateTime(0)

		bella.Presence.ChangedFlags = types.NewUInt32(0x1EE)
		bella.Presence.Online = types.NewBool(true)
		bella.Presence.GameKey = friends_wiiu_types.NewGameKey()
		bella.Presence.Unknown1 = types.NewUInt8(0)
		bella.Presence.Message = types.NewString("Testing")
		bella.Presence.Unknown2 = types.NewUInt32(0)
		bella.Presence.Unknown3 = types.NewUInt8(0)
		bella.Presence.GameServerID = types.NewUInt32(0)
		bella.Presence.Unknown4 = types.NewUInt32(0)
		bella.Presence.PID = types.NewPID(1743126339)
		bella.Presence.GatheringID = types.NewUInt32(0)
		bella.Presence.ApplicationData = types.NewBuffer([]byte{0x0})
		bella.Presence.Unknown5 = types.NewUInt8(0)
		bella.Presence.Unknown6 = types.NewUInt8(0)
		bella.Presence.Unknown7 = types.NewUInt8(0)

		bella.Presence.GameKey.TitleID = 0x0005000010176900
		bella.Presence.GameKey.TitleVersion = types.NewUInt16(0)

		bella.Status.Unknown = types.NewUInt8(0)
		bella.Status.Contents = types.NewString("Howdy hey!")
		bella.Status.LastChanged = types.NewDateTime(0)

		friendList = append(friendList, bella)
	}

	for _, friend := range friendList {
		// TODO: Is there a better way to do this? I really don't know what I'm doing here
		var comment = &pb.Comment{
			Contents:    string(friend.Status.Contents),
			LastChanged: uint64(friend.Status.LastChanged),
		}
		var mii = &pb.MiiV2{
			Name:     string(friend.NNAInfo.PrincipalBasicInfo.Mii.Name),
			MiiData:  friend.NNAInfo.PrincipalBasicInfo.Mii.MiiData,
			Datetime: uint64(friend.NNAInfo.PrincipalBasicInfo.Mii.Datetime),
		}
		var principal = &pb.PrincipalBasicInfo{
			Pid:  uint32(friend.NNAInfo.PrincipalBasicInfo.PID),
			Nnid: string(friend.NNAInfo.PrincipalBasicInfo.NNID),
			Mii:  mii,
		}
		var nnaInfo = &pb.NNAInfo{
			PrincipalBasicInfo: principal,
		}
		var gameKey = &pb.GameKey{
			TitleId:      uint64(friend.Presence.GameKey.TitleID),
			TitleVersion: uint32(friend.Presence.GameKey.TitleVersion),
		}
		var presence = &pb.NintendoPresenceV2{
			ChangedFlags:    uint32(friend.Presence.ChangedFlags),
			Online:          bool(friend.Presence.Online),
			GameKey:         gameKey,
			Message:         string(friend.Presence.Message),
			GameServerId:    uint32(friend.Presence.GameServerID),
			Pid:             uint32(friend.Presence.PID),
			GatheringId:     uint32(friend.Presence.GatheringID),
			ApplicationData: friend.Presence.ApplicationData,
		}
		var info = &pb.FriendInfoWiiU{
			NnaInfo:      nnaInfo,
			Presence:     presence,
			Status:       comment,
			BecameFriend: uint64(friend.BecameFriend),
			LastOnline:   uint64(friend.LastOnline),
		}
		friends = append(friends, info)
	}

	return &pb.GetUserFriendsDataWiiUResponse{
		Friends: friends,
	}, nil
}

func (s *gRPCFriendsV2Server) GetUserFriendsData3DS(ctx context.Context, in *pb.GetUserFriendsDataRequest) (*pb.GetUserFriendsData3DSResponse, error) {
	var friends []*pb.FriendInfo3DS
	friendList, err := database_3ds.GetUserFriends(in.Pid)
	if err != nil && err != database.ErrEmptyList {
		return &pb.GetUserFriendsData3DSResponse{
			Friends: friends,
		}, nil
	}

	friendPIDs := make([]uint32, len(friendList))

	for _, friend := range friendList {
		friendPIDs = append(friendPIDs, uint32(friend.PID))
	}

	friendInfoList, err := database_3ds.GetFriendPersistentInfos(uint32(in.Pid), friendPIDs)
	if err != nil && err != sql.ErrNoRows {
		globals.Logger.Critical(err.Error())
		return &pb.GetUserFriendsData3DSResponse{
			Friends: friends,
		}, nil
	}

	miiList, err := database_3ds.GetFriendMiis(friendPIDs)
	if err != nil && err != sql.ErrNoRows {
		globals.Logger.Critical(err.Error())
		return &pb.GetUserFriendsData3DSResponse{
			Friends: friends,
		}, nil
	}

	if globals.Config.EnableBella {
		bella := friends_3ds_types.NewFriendPersistentInfo()

		bella.PID = types.NewPID(1743126339)
		bella.Region = types.NewUInt8(0)
		bella.Country = types.NewUInt8(0)
		bella.Area = types.NewUInt8(0)
		bella.Language = types.NewUInt8(0)
		bella.Platform = types.NewUInt8(0)
		bella.GameKey.TitleID = 0x0005000010176900
		bella.GameKey.TitleVersion = types.NewUInt16(0)
		bella.Message = "Howdy Hey!"
		bella.MessageUpdatedAt = types.NewDateTime(0)
		bella.MiiModifiedAt = types.NewDateTime(0)
		bella.LastOnline = types.NewDateTime(0)

		mii := friends_3ds_types.NewMii()
		mii.Name = types.NewString("Bandwidth")
		mii.ProfanityFlag = types.NewBool(false)
		mii.CharacterSet = types.NewUInt8(0)
		mii.MiiData = types.NewBuffer([]byte{
			0x03, 0x00, 0x00, 0x40, 0xE9, 0x55, 0xA2, 0x09,
			0xE7, 0xC7, 0x41, 0x82, 0xD9, 0x7D, 0x0B, 0x2D,
			0x03, 0xB3, 0xB8, 0x8D, 0x27, 0xD9, 0x00, 0x00,
			0x01, 0x40, 0x62, 0x00, 0x65, 0x00, 0x6C, 0x00,
			0x6C, 0x00, 0x61, 0x00, 0x00, 0x00, 0x45, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x40,
			0x12, 0x00, 0x81, 0x01, 0x04, 0x68, 0x43, 0x18,
			0x20, 0x34, 0x46, 0x14, 0x81, 0x12, 0x17, 0x68,
			0x0D, 0x00, 0x00, 0x29, 0x03, 0x52, 0x48, 0x50,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFE, 0x86,
		})

		friendMii := friends_3ds_types.NewFriendMii()
		friendMii.PID = types.NewPID(uint64(bella.PID))
		friendMii.Mii = mii
		friendMii.ModifiedAt = types.NewDateTime(0)

		friendInfoList = append(friendInfoList, bella)
		miiList = append(miiList, friendMii)
	}

	for _, friend := range friendInfoList {
		// TODO: Is there a better way to do this? I really don't know what I'm doing here
		var gameKey = &pb.GameKey{
			TitleId:      uint64(friend.GameKey.TitleID),
			TitleVersion: uint32(friend.GameKey.TitleVersion),
		}
		var miiIndex = -1
		for index, mii := range miiList {
			if mii.PID == friend.PID {
				miiIndex = index
				break
			}
		}
		if miiIndex == -1 {
			continue
		}
		var miiData = miiList[miiIndex]
		var mii = &pb.MiiV2{
			Name:     string(miiData.Mii.Name),
			MiiData:  miiData.Mii.MiiData,
			Datetime: uint64(miiData.ModifiedAt),
		}
		var info = &pb.FriendInfo3DS{
			Pid:              uint32(friend.PID),
			Region:           uint32(friend.Region),
			Country:          uint32(friend.Country),
			Area:             uint32(friend.Area),
			Language:         uint32(friend.Language),
			Platform:         uint32(friend.Platform),
			GameKey:          gameKey,
			Message:          string(friend.Message),
			MessageUpdatedAt: uint64(friend.MessageUpdatedAt),
			MiiModifiedAt:    uint64(friend.MiiModifiedAt),
			LastOnline:       uint64(friend.LastOnline),
			Mii:              mii,
		}
		friends = append(friends, info)
	}

	return &pb.GetUserFriendsData3DSResponse{
		Friends: friends,
	}, nil
}
