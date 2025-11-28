package utils

import "math/big"

var base58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Base58Encode(number int64) string {
	if number == 0 {
		return string(base58Alphabet[0])
	}

	result := make([]byte, 0)
	bigNumber := big.NewInt(number)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := &big.Int{}

	for bigNumber.Cmp(zero) > 0 {
		bigNumber.DivMod(bigNumber, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// 反转结果
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
