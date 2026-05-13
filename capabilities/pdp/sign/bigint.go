package sign

import (
	"errors"
	"math/big"
)

// ErrInvalidSignByte is returned when BigIntFromBytes encounters a wire-
// encoded big.Int whose leading sign byte is neither 0 (zero/positive)
// nor 1 (negative).
var ErrInvalidSignByte = errors.New("pdp/sign: big.Int wire encoding has invalid sign byte")

// BigIntToBytes encodes a *big.Int to its wire form: a single sign byte
// (0x00 for zero or positive, 0x01 for negative) followed by the
// big-endian magnitude. A nil input is treated as zero.
func BigIntToBytes(n *big.Int) []byte {
	if n == nil {
		return []byte{0}
	}
	switch n.Sign() {
	case -1:
		return append([]byte{1}, n.Bytes()...)
	case 0:
		return []byte{0}
	default:
		return append([]byte{0}, n.Bytes()...)
	}
}

// BigIntFromBytes decodes a wire-encoded big.Int (see [BigIntToBytes]).
// An empty slice is treated as zero.
func BigIntFromBytes(b []byte) (*big.Int, error) {
	if len(b) == 0 {
		return big.NewInt(0), nil
	}
	mag := new(big.Int).SetBytes(b[1:])
	switch b[0] {
	case 0:
		return mag, nil
	case 1:
		return mag.Neg(mag), nil
	default:
		return nil, ErrInvalidSignByte
	}
}
