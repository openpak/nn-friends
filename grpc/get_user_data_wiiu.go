package grpc

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/database"
	database_wiiu "github.com/PretendoNetwork/friends/database/wiiu"
	"github.com/PretendoNetwork/friends/globals"
	pb "github.com/PretendoNetwork/grpc/go/friends/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *gRPCFriendsV2Server) GetUserDataWiiU(ctx context.Context, in *pb.GetUserDataWiiURequest) (*pb.GetUserDataWiiUResponse, error) {
	user, err := database_wiiu.GetUserData(types.PID(in.GetPid()))
	if err != nil {
		if err == database.ErrPIDNotFound {
			return nil, status.Errorf(codes.NotFound, "PID was not found")
		} else {
			globals.Logger.Critical(err.Error())
			return nil, status.Errorf(codes.Internal, "Internal error")
		}
	}

	comment := &pb.Comment{
		Contents:    string(user.Status.Contents),
		LastChanged: timestamppb.New(time.Unix(int64(user.Status.LastChanged.Second()), 0)),
	}

	mii := &pb.MiiV2{
		Name:     string(user.NNAInfo.PrincipalBasicInfo.Mii.Name),
		MiiData:  user.NNAInfo.PrincipalBasicInfo.Mii.MiiData,
		Datetime: timestamppb.New(time.Unix(int64(user.NNAInfo.PrincipalBasicInfo.Mii.Datetime.Second()), 0)),
	}

	principal := &pb.PrincipalBasicInfo{
		Pid:  uint32(user.NNAInfo.PrincipalBasicInfo.PID),
		Nnid: string(user.NNAInfo.PrincipalBasicInfo.NNID),
		Mii:  mii,
	}

	nnaInfo := &pb.NNAInfo{
		PrincipalBasicInfo: principal,
	}

	gameKey := &pb.GameKey{
		TitleId:      uint64(user.Presence.GameKey.TitleID),
		TitleVersion: uint32(user.Presence.GameKey.TitleVersion),
	}

	presence := &pb.NintendoPresenceV2{
		ChangedFlags:    uint32(user.Presence.ChangedFlags),
		Online:          bool(user.Presence.Online),
		GameKey:         gameKey,
		Message:         string(user.Presence.Message),
		GameServerId:    uint32(user.Presence.GameServerID),
		Pid:             uint32(user.Presence.PID),
		GatheringId:     uint32(user.Presence.GatheringID),
		ApplicationData: user.Presence.ApplicationData,
	}

	info := &pb.FriendInfoWiiU{
		NnaInfo:      nnaInfo,
		Presence:     presence,
		Status:       comment,
		BecameFriend: timestamppb.New(user.BecameFriend.Standard()),
		LastOnline:   timestamppb.New(user.LastOnline.Standard()),
	}
	return &pb.GetUserDataWiiUResponse{
		User: info,
	}, nil
}
