package nex_account_management

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"

	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/friends/utility"
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	account_management "github.com/PretendoNetwork/nex-protocols-go/v2/account-management"
)

func NintendoCreateAccount(err error, packet nex.PacketInterface, callID uint32, strPrincipalName types.String, strKey types.String, uiGroups types.UInt32, strEmail types.String, oAuthData types.DataHolder) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	decryptedToken, nexError := utility.ValidateNintendoCreateAccountToken(oAuthData)
	if nexError != nil {
		globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	pid := types.NewPID(uint64(decryptedToken.UserPID))

	pidByteArray := make([]byte, 4)
	binary.LittleEndian.PutUint32(pidByteArray, uint32(pid))

	mac := hmac.New(md5.New, []byte(strKey))
	_, err = mac.Write(pidByteArray)
	if err != nil {
		globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Authentication.Unknown, err.Error())
	}

	pidHmac := types.NewString(hex.EncodeToString(mac.Sum(nil)))

	rmcResponseStream := nex.NewByteStreamOut(globals.SecureEndpoint.LibraryVersions(), globals.SecureEndpoint.ByteStreamSettings())

	pid.WriteTo(rmcResponseStream)
	pidHmac.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(globals.SecureEndpoint, rmcResponseBody)
	rmcResponse.ProtocolID = account_management.ProtocolID
	rmcResponse.MethodID = account_management.MethodNintendoCreateAccount
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
