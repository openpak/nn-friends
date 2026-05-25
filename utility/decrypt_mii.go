package utility

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"

	"github.com/PretendoNetwork/friends/globals"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

func ccmDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	L := 15 - len(nonce)

	counter := make([]byte, aes.BlockSize)
	counter[0] = byte(L - 1)
	copy(counter[1:1+len(nonce)], nonce)
	stream := cipher.NewCTR(block, counter)

	padded := make([]byte, aes.BlockSize+len(ciphertext))
	copy(padded[aes.BlockSize:], ciphertext)

	out := make([]byte, len(padded))
	stream.XORKeyStream(out, padded)

	return out[aes.BlockSize:], nil
}

func DecryptMiiData(mii_data types.Buffer) ([]byte, error) {
	if len(mii_data) < 96 {
		return nil, fmt.Errorf("Mii data length is incorrect: %d", len(mii_data))
	}
	nonce := mii_data[:8]
	ciphertext := mii_data[8 : 8+0x58]

	key, err := hex.DecodeString(globals.Config.MiiDecryptKey)
	if err != nil {
		return nil, err
	}
	content, err := ccmDecrypt(key, append(append([]byte{}, nonce...), 0, 0, 0, 0), ciphertext)
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, len(content)+len(nonce))
	result = append(result, content[:12]...)
	result = append(result, nonce...)
	result = append(result, content[12:]...)

	return result, nil
}
