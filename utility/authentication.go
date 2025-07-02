package utility

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	account_management_types "github.com/PretendoNetwork/nex-protocols-go/v2/account-management/types"

	"github.com/PretendoNetwork/friends/globals"
)

// ValidateNintendoCreateAccountToken validates the given Pretendo token for account creation
func ValidateNintendoCreateAccountToken(token types.DataHolder) (*common_globals.NEXToken, *nex.Error) {
	var tokenBase64 string

	tokenDataType := token.Object.DataObjectID().(types.String)

	switch tokenDataType {
	case "NintendoCreateAccountData": // * Wii U
		nintendoCreateAccountData := token.Object.Copy().(account_management_types.NintendoCreateAccountData)

		tokenBase64 = string(nintendoCreateAccountData.Token)
	case "AccountExtraInfo": // * 3DS
		accountExtraInfo := token.Object.Copy().(account_management_types.AccountExtraInfo)

		tokenBase64 = string(accountExtraInfo.NEXToken)
		tokenBase64 = strings.Replace(tokenBase64, ".", "+", -1)
		tokenBase64 = strings.Replace(tokenBase64, "-", "/", -1)
		tokenBase64 = strings.Replace(tokenBase64, "*", "=", -1)
	default:
		globals.Logger.Errorf("Invalid token data type %s!", tokenDataType)
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, fmt.Sprintf("Invalid token data type %s!", tokenDataType))
	}

	encryptedToken, err := base64.StdEncoding.DecodeString(tokenBase64)
	if err != nil {
		globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, err.Error())
	}

	decryptedToken, nexError := common_globals.DecryptToken(encryptedToken, globals.AESKey)
	if nexError != nil {
		return nil, nexError
	}

	// Check for NEX token type
	if decryptedToken.TokenType != 3 {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Invalid token type")
	}

	// Expire time is in milliseconds
	expireTime := time.Unix(int64(decryptedToken.ExpireTime / 1000), 0)

	if expireTime.Before(time.Now()) {
		return nil, nex.NewError(nex.ResultCodes.Authentication.TokenExpired, "Token expired")
	}

	// PID isn't checked since account creation is done with a guest account

	if decryptedToken.AccessLevel < 0 {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.AccountDisabled, fmt.Sprintf("Account %d is banned", decryptedToken.UserPID))
	}

	return decryptedToken, nil
}
