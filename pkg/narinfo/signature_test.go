package narinfo_test

import (
	"testing"

	"github.com/numtide/nar-serve/pkg/narinfo"
	"github.com/stretchr/testify/assert"
)

const (
	dummySigLine = "cache.nixos.org-1" +
		":" + "rH4wxlNRbTbViQon40C15og5zlcFEphwoF26IQGHi2QCwVYyaLj6LOag+MeWcZ65SWzy6PnOlXjriLNcxE0hAQ=="
	expectedKeyName = "cache.nixos.org-1"
)

// nolint:gochecknoglobals
var (
	expectedDigest = []byte{
		0xac, 0x7e, 0x30, 0xc6, 0x53, 0x51, 0x6d, 0x36, 0xd5, 0x89, 0x0a, 0x27, 0xe3, 0x40, 0xb5, 0xe6,
		0x88, 0x39, 0xce, 0x57, 0x05, 0x12, 0x98, 0x70, 0xa0, 0x5d, 0xba, 0x21, 0x01, 0x87, 0x8b, 0x64,
		0x02, 0xc1, 0x56, 0x32, 0x68, 0xb8, 0xfa, 0x2c, 0xe6, 0xa0, 0xf8, 0xc7, 0x96, 0x71, 0x9e, 0xb9,
		0x49, 0x6c, 0xf2, 0xe8, 0xf9, 0xce, 0x95, 0x78, 0xeb, 0x88, 0xb3, 0x5c, 0xc4, 0x4d, 0x21, 0x01,
	}
)

func TestParseSignatureLine(t *testing.T) {
	signature, err := narinfo.ParseSignatureLine(dummySigLine)
	if assert.NoError(t, err) {
		assert.Equal(t, expectedKeyName, signature.KeyName)
		assert.Equal(t, expectedDigest, signature.Digest)
	}
}

func TestMustParseSignatureLine(t *testing.T) {
	signature := narinfo.MustParseSignatureLine(dummySigLine)
	assert.Equal(t, expectedKeyName, signature.KeyName)
	assert.Equal(t, expectedDigest, signature.Digest)

	assert.Panics(t, func() {
		_ = narinfo.MustParseSignatureLine(expectedKeyName)
	})
}

func BenchmarkParseSignatureLine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		narinfo.MustParseSignatureLine(dummySigLine)
	}
}
