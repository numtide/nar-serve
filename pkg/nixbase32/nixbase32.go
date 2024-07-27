package nixbase32

import (
	"fmt"
	"strings"
)

// Alphabet contains the list of valid characters for the Nix base32 alphabet.
const Alphabet = "0123456789abcdfghijklmnpqrsvwxyz"

func decodeString(s string, dst []byte) error {
	var dstLen int
	if dst != nil {
		dstLen = len(dst)
	} else {
		dstLen = DecodedLen(len(s))
	}

	for n := 0; n < len(s); n++ {
		c := s[len(s)-n-1]

		digit := strings.IndexByte(Alphabet, c)
		if digit == -1 {
			return fmt.Errorf("character %v not in alphabet", c)
		}

		b := uint(n * 5)
		i := b / 8
		j := b % 8

		// OR the main pattern
		if dst != nil {
			dst[i] |= byte(digit) << j
		}

		// calculate the "carry pattern"
		carry := byte(digit) >> (8 - j)

		// if we're at the end of dst…
		if i == uint(dstLen-1) {
			// but have a nonzero carry, the encoding is invalid.
			if carry != 0 {
				return fmt.Errorf("invalid encoding")
			}
		} else if dst != nil {
			dst[i+1] |= carry
		}
	}

	return nil
}

// ValidateBytes validates if a byte slice is valid nixbase32.
func ValidateBytes(b []byte) error {
	return ValidateString(string(b))
}

// ValidateString validates if a string is valid nixbase32.
func ValidateString(s string) error {
	return decodeString(s, nil)
}

// EncodedLen returns the length in bytes of the base32 encoding of an input
// buffer of length n.
func EncodedLen(n int) int {
	if n == 0 {
		return 0
	}

	return (n*8-1)/5 + 1
}

// DecodedLen returns the length in bytes of the decoded data
// corresponding to n bytes of base32-encoded data.
// If we have bits that don't fit into here, they are padding and must
// be 0.
func DecodedLen(n int) int {
	return (n * 5) / 8
}

// EncodeToString returns the nixbase32 encoding of src.
func EncodeToString(src []byte) string {
	l := EncodedLen(len(src))

	var dst strings.Builder

	dst.Grow(l)

	for n := l - 1; n >= 0; n-- {
		b := uint(n * 5)
		i := b / 8
		j := b % 8

		c := src[i] >> j

		if i+1 < uint(len(src)) {
			c |= src[i+1] << (8 - j)
		}

		dst.WriteByte(Alphabet[c&0x1f])
	}

	return dst.String()
}

// DecodeString returns the bytes represented by the nixbase32 string s or
// returns an error.
func DecodeString(s string) ([]byte, error) {
	dst := make([]byte, DecodedLen(len(s)))

	return dst, decodeString(s, dst)
}

// MustDecodeString returns the bytes represented by the nixbase32 string s or
// panics on error.
func MustDecodeString(s string) []byte {
	b, err := DecodeString(s)
	if err != nil {
		panic(err)
	}

	return b
}
