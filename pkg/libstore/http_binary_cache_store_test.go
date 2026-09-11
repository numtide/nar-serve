package libstore_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/numtide/nar-serve/pkg/libstore"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPGetFileUndoesTheContentEncoding(t *testing.T) {
	var body bytes.Buffer

	w, err := zstd.NewWriter(&body)
	require.NoError(t, err)
	_, err = io.WriteString(w, `{"version":1}`)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/x.ls":
			w.Header().Set("Content-Encoding", "zstd")
			_, _ = w.Write(body.Bytes())
		case "/x.narinfo":
			_, _ = io.WriteString(w, "plain")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	store := libstore.NewHTTPBinaryCacheStore(u)

	r, err := store.GetFile(context.Background(), "x.ls")
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, `{"version":1}`, string(got))

	r, err = store.GetFile(context.Background(), "x.narinfo")
	require.NoError(t, err)
	got, err = io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "plain", string(got))

	_, err = store.GetFile(context.Background(), "absent")
	assert.Error(t, err)
}
