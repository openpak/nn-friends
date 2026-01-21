package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/types"
	pb "github.com/PretendoNetwork/grpc/go/account"
	"github.com/PretendoNetwork/nex-go/v2"
	nex_types "github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/plogger-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/joho/godotenv"
)

func init() {
	globals.Logger = plogger.NewLogger()
	globals.ConnectedUsers = nex.NewMutexMap[uint32, *types.ConnectedUser]()

	var err error

	err = godotenv.Load()
	if err != nil {
		globals.Logger.Warningf("Error loading .env file: %s", err.Error())
	}

	globals.Config = globals.NewConfigParser(globals.Config).SetPrefix("PN_FRIENDS_CONFIG").ParseFromEnv()

	kerberosPassword := make([]byte, 0x10)
	_, err = rand.Read(kerberosPassword)
	if err != nil {
		globals.Logger.Error("Error generating Kerberos password")
		os.Exit(0)
	}

	globals.KerberosPassword = string(kerberosPassword)

	globals.AuthenticationServerAccount = nex.NewAccount(nex_types.NewPID(1), "Quazal Authentication", globals.KerberosPassword, true) // * Is "true" correct here?
	globals.SecureServerAccount = nex.NewAccount(nex_types.NewPID(2), "Quazal Rendez-Vous", globals.KerberosPassword, true)            // * Is "true" correct here?
	globals.GuestAccount = nex.NewAccount(nex_types.NewPID(100), "guest", "MMQea3n!fsik", true)                                        // * Is "true" correct here? Guest account password is always the same, known to all consoles. Only allow on the friends server
	globals.AESKey, err = hex.DecodeString(globals.Config.AESKey)
	if err != nil {
		globals.Logger.Criticalf("Failed to decode AES key: %v", err)
		os.Exit(0)
	}

	if strings.TrimSpace(globals.Config.GRPCAPIKey) == "" {
		globals.Logger.Warning("Insecure gRPC server detected. PN_FRIENDS_CONFIG_GRPC_API_KEY environment variable not set")
	}

	if strings.TrimSpace(globals.Config.AccountGRPCAPIKey) == "" {
		globals.Logger.Warning("Insecure gRPC server detected. PN_FRIENDS_CONFIG_ACCOUNT_GRPC_API_KEY environment variable not set")
	}

	globals.GRPCAccountClientConnection, err = grpc.Dial(fmt.Sprintf("%s:%s", globals.Config.AccountGRPCHost, globals.Config.AccountGRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		globals.Logger.Criticalf("Failed to connect to account gRPC server: %v", err)
		os.Exit(0)
	}

	globals.GRPCAccountClient = pb.NewAccountClient(globals.GRPCAccountClientConnection)
	globals.GRPCAccountCommonMetadata = metadata.Pairs(
		"X-API-Key", globals.Config.AccountGRPCAPIKey,
	)

	database.ConnectPostgres()
}
