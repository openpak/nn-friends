package grpc

import (
	"context"
	"time"

	"github.com/PretendoNetwork/friends/database"
	database_3ds "github.com/PretendoNetwork/friends/database/3ds"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/utility"
	pb "github.com/PretendoNetwork/grpc/go/friends/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *gRPCFriendsV2Server) GetUserData3DS(ctx context.Context, in *pb.GetUserData3DSRequest) (*pb.GetUserData3DSResponse, error) {
	user, err := database_3ds.GetUserData(types.PID(in.GetPid()))
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
		TitleId:      uint64(user.GameKey.TitleID),
		TitleVersion: uint32(user.GameKey.TitleVersion),
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
	connectedUser, ok := globals.ConnectedUsers.Get(uint32(user.PID))
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
		Pid:              uint32(user.PID),
		Region:           uint32(user.Region),
		Country:          uint32(user.Country),
		Area:             uint32(user.Area),
		Language:         uint32(user.Language),
		Platform:         uint32(user.Platform),
		Presence:         presence,
		GameKey:          gameKey,
		Message:          string(user.Message),
		MessageUpdatedAt: timestamppb.New(user.MessageUpdatedAt.Standard()),
		MiiModifiedAt:    timestamppb.New(user.MiiModifiedAt.Standard()),
		LastOnline:       timestamppb.New(user.LastOnline.Standard()),
		Mii:              friendMii,
	}
	return &pb.GetUserData3DSResponse{
		User: info,
	}, nil
}
