package compression_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/numtide/nar-serve/pkg/compression"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

const payload = "hello from the binary cache\n"

func compress(t *testing.T, name string) []byte {
	t.Helper()

	var buf bytes.Buffer

	var w io.WriteCloser

	switch name {
	case "gzip":
		w = gzip.NewWriter(&buf)
	case "zstd":
		var err error

		w, err = zstd.NewWriter(&buf)
		require.NoError(t, err)
	case "br":
		w = brotli.NewWriter(&buf)
	case "xz":
		var err error

		w, err = xz.NewWriter(&buf)
		require.NoError(t, err)
	default:
		t.Fatalf("no writer for %s", name)
	}

	_, err := io.WriteString(w, payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

func TestNewReaderRoundTrips(t *testing.T) {
	for _, name := range []string{"gzip", "zstd", "br", "xz"} {
		t.Run(name, func(t *testing.T) {
			r, err := compression.NewReader(name, bytes.NewReader(compress(t, name)))
			require.NoError(t, err)

			got, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, payload, string(got))
		})
	}
}

func TestNewReaderPassesPlainStreamsThrough(t *testing.T) {
	for _, name := range []string{"", "none", "identity"} {
		r, err := compression.NewReader(name, bytes.NewReader([]byte(payload)))
		require.NoError(t, err)

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, payload, string(got))
	}
}

func TestNewReaderRejectsUnknownNames(t *testing.T) {
	_, err := compression.NewReader("lzip", bytes.NewReader(nil))
	assert.Error(t, err)
}
