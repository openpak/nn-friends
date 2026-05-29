package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/PretendoNetwork/friends/database"
	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/types"
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
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

	globals.AuthenticationServerAccount = nex.NewAccount(nex_types.NewPID(1), "Quazal Authentication", globals.KerberosPassword, false)
	globals.SecureServerAccount = nex.NewAccount(nex_types.NewPID(2), "Quazal Rendez-Vous", globals.KerberosPassword, false)
	globals.GuestAccount = nex.NewAccount(nex_types.NewPID(100), "guest", "MMQea3n!fsik", false)
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

	globals.GRPCAccountClientConnection, err = grpc.NewClient(fmt.Sprintf("%s:%d", globals.Config.AccountGRPCHost, globals.Config.AccountGRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		globals.Logger.Criticalf("Failed to connect to account gRPC server: %v", err)
		os.Exit(0)
	}

	globals.GRPCAccountClient = pb.NewAccountServiceClient(globals.GRPCAccountClientConnection)
	globals.GRPCAccountCommonMetadata = metadata.Pairs(
		"X-API-Key", globals.Config.AccountGRPCAPIKey,
	)

	if strings.TrimSpace(globals.Config.MiiDecryptKey) == "" {
		globals.Logger.Warning("PN_FRIENDS_CONFIG_MII_DECRYPT_KEY environment variable not set. 3DS Mii data cannot be decrypted")
	}

	miiKeyBytes, err := hex.DecodeString(globals.Config.MiiDecryptKey)
	if err != nil {
		globals.Logger.Criticalf("Failed to decode PN_FRIENDS_CONFIG_MII_DECRYPT_KEY %v", err)
		os.Exit(0)
	}

	miiMD5Hash := md5.Sum(miiKeyBytes)
	if hex.EncodeToString(miiMD5Hash[:]) != "aeb707b225ec0fcd8a503e26e3dcd596" {
		globals.Logger.Criticalf("PN_FRIENDS_CONFIG_MII_DECRYPT_KEY is incorrect! md5: %s", hex.EncodeToString(miiMD5Hash[:]))
		os.Exit(0)
	}

	database.ConnectPostgres()
}
