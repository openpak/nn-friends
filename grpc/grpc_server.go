package grpc

import (
	"fmt"
	"log"
	"net"

	"github.com/PretendoNetwork/friends/globals"
	pb "github.com/PretendoNetwork/grpc/go/friends"
	pbv2 "github.com/PretendoNetwork/grpc/go/friends/v2"
	"google.golang.org/grpc"
)

type gRPCFriendsServer struct {
	pb.UnimplementedFriendsServer
}

type gRPCFriendsV2Server struct {
	pbv2.UnimplementedFriendsServiceServer
}

func StartGRPCServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", globals.Config.GRPCServerPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(apiKeyInterceptor),
	)

	pb.RegisterFriendsServer(server, &gRPCFriendsServer{})
	pbv2.RegisterFriendsServiceServer(server, &gRPCFriendsV2Server{})

	log.Printf("server listening at %v", listener.Addr())

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
