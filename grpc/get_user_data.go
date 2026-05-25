package grpc

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/database"
	database_3ds "github.com/PretendoNetwork/friends/database/3ds"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/utility"
	pb "github.com/PretendoNetwork/grpc/go/friends/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *gRPCFriendsV2Server) GetUserDataWiiU(ctx context.Context, in *pb.GetUserDataWiiURequest) (*pb.GetUserDataWiiUResponse, error) {
	friend, err := database_wiiu.GetUserData(types.PID(in.GetPid()))
	if err != nil {
		globals.Logger.Critical(err.Error())
		if err == database.ErrPIDNotFound {
			return nil, status.Errorf(codes.NotFound, "PID was not found")
		} else {
			return nil, status.Errorf(codes.Internal, "Internal error")
		}
	}

	comment := &pb.Comment{
		Contents:    string(friend.Status.Contents),
		LastChanged: timestamppb.New(time.Unix(int64(friend.Status.LastChanged.Second()), 0)),
	}

	mii := &pb.MiiV2{
		Name:     string(friend.NNAInfo.PrincipalBasicInfo.Mii.Name),
		MiiData:  friend.NNAInfo.PrincipalBasicInfo.Mii.MiiData,
		Datetime: timestamppb.New(time.Unix(int64(friend.NNAInfo.PrincipalBasicInfo.Mii.Datetime.Second()), 0)),
	}

	principal := &pb.PrincipalBasicInfo{
		Pid:  uint32(friend.NNAInfo.PrincipalBasicInfo.PID),
		Nnid: string(friend.NNAInfo.PrincipalBasicInfo.NNID),
		Mii:  mii,
	}

	nnaInfo := &pb.NNAInfo{
		PrincipalBasicInfo: principal,
	}

	gameKey := &pb.GameKey{
		TitleId:      uint64(friend.Presence.GameKey.TitleID),
		TitleVersion: uint32(friend.Presence.GameKey.TitleVersion),
	}

	presence := &pb.NintendoPresenceV2{
		ChangedFlags:    uint32(friend.Presence.ChangedFlags),
		Online:          bool(friend.Presence.Online),
		GameKey:         gameKey,
		Message:         string(friend.Presence.Message),
		GameServerId:    uint32(friend.Presence.GameServerID),
		Pid:             uint32(friend.Presence.PID),
		GatheringId:     uint32(friend.Presence.GatheringID),
		ApplicationData: friend.Presence.ApplicationData,
	}

	info := &pb.FriendInfoWiiU{
		NnaInfo:      nnaInfo,
		Presence:     presence,
		Status:       comment,
		BecameFriend: timestamppb.New(friend.BecameFriend.Standard()),
		LastOnline:   timestamppb.New(friend.LastOnline.Standard()),
	}
	return &pb.GetUserDataWiiUResponse{
		User: info,
	}, nil
}

func (s *gRPCFriendsV2Server) GetUserData3DS(ctx context.Context, in *pb.GetUserData3DSRequest) (*pb.GetUserData3DSResponse, error) {
	friend, err := database_3ds.GetUserData(types.PID(in.GetPid()))
	if err != nil {
		globals.Logger.Critical(err.Error())
		if err == database.ErrPIDNotFound {
			return nil, status.Errorf(codes.NotFound, "PID was not found")
		} else {
			return nil, status.Errorf(codes.Internal, "Internal error")
		}
	}

	miiData, err := database_3ds.GetMii(types.PID(in.GetPid()))
	if err != nil {
		globals.Logger.Critical(err.Error())
		if err == database.ErrPIDNotFound {
			return nil, status.Errorf(codes.NotFound, "PID was not found")
		} else {
			return nil, status.Errorf(codes.Internal, "Internal error")
		}
	}

	gameKey := &pb.GameKey{
		TitleId:      uint64(friend.GameKey.TitleID),
		TitleVersion: uint32(friend.GameKey.TitleVersion),
	}

	mii := &pb.Mii{
		Name:             string(miiData.Mii.Name),
		ProfanityFlag:    bool(miiData.Mii.ProfanityFlag),
		CharacterSet:     uint32(miiData.Mii.CharacterSet),
		MiiDataEncrypted: miiData.Mii.MiiData,
	}

	mii_data, err := utility.DecryptMiiData(miiData.Mii.MiiData)
	if err == nil {
		mii.MiiData = mii_data
	}
	friendMii := &pb.FriendMii{
		Pid:        uint32(miiData.PID),
		Mii:        mii,
		ModifiedAt: timestamppb.New(time.Unix(int64(miiData.ModifiedAt.Second()), 0)),
	}

	presence := &pb.NintendoPresence{}
	connectedUser, ok := globals.ConnectedUsers.Get(uint32(friend.PID))
	if ok && connectedUser != nil {
		presence.ChangedFlags = uint32(connectedUser.Presence.ChangedFlags)
		presence.GameKey = &pb.GameKey{
			TitleId:      uint64(connectedUser.Presence.GameKey.TitleID),
			TitleVersion: uint32(connectedUser.Presence.GameKey.TitleVersion),
		}
		presence.Message = string(connectedUser.Presence.Message)
		presence.JoinAvailableFlag = uint32(connectedUser.Presence.JoinAvailableFlag)
		presence.MatchmakeType = uint32(connectedUser.Presence.MatchmakeType)
		presence.JoinGameId = uint32(connectedUser.Presence.JoinGameID)
		presence.JoinGameMode = uint32(connectedUser.Presence.JoinGameMode)
		presence.OwnerPid = uint32(connectedUser.Presence.OwnerPID)
		presence.JoinGroupId = uint32(connectedUser.Presence.JoinGroupID)
		presence.ApplicationArg = connectedUser.Presence.ApplicationArg
	}

	info := &pb.FriendInfo3DS{
		Pid:              uint32(friend.PID),
		Region:           uint32(friend.Region),
		Country:          uint32(friend.Country),
		Area:             uint32(friend.Area),
		Language:         uint32(friend.Language),
		Platform:         uint32(friend.Platform),
		Presence:         presence,
		GameKey:          gameKey,
		Message:          string(friend.Message),
		MessageUpdatedAt: timestamppb.New(friend.MessageUpdatedAt.Standard()),
		MiiModifiedAt:    timestamppb.New(friend.MiiModifiedAt.Standard()),
		LastOnline:       timestamppb.New(friend.LastOnline.Standard()),
		Mii:              friendMii,
	}
	return &pb.GetUserData3DSResponse{
		User: info,
	}, nil
}
