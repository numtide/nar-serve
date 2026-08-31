package unpack_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const immutableCacheControl = "public, max-age=31536000, immutable"

// conditionalGet claims to hold etag already
func conditionalGet(
	t *testing.T, srv http.Handler, path, etag string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("If-None-Match", etag)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	return rec
}

func TestRegularFileIsCacheable(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/share/applications/example.desktop"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, immutableCacheControl, rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
}

// a listing is this code's html, not the store path's bytes, so it changes
// when the template does
func TestListingAndRedirectAreNotImmutable(t *testing.T) {
	srv, _ := newServer(t)

	for _, path := range []string{"/share/applications", "/share/link"} {
		rec := get(t, srv, "GET", storePath(path))

		require.Contains(t, []int{http.StatusOK, http.StatusMovedPermanently}, rec.Code)
		assert.Empty(t, rec.Header().Get("Cache-Control"), path)
		assert.Empty(t, rec.Header().Get("ETag"), path)
	}
}

// a 404 means the cache had nothing today, and a cache gains content
func TestNotFoundIsNotImmutable(t *testing.T) {
	srv, _ := newServer(t)

	for _, path := range []string{"/share/applications/missing", "/zzz-past-the-end"} {
		rec := get(t, srv, "GET", storePath(path))

		require.Equal(t, http.StatusNotFound, rec.Code, path)
		assert.Empty(t, rec.Header().Get("Cache-Control"), path)
		assert.Empty(t, rec.Header().Get("ETag"), path)
	}
}

// the validator comes from the url, so this costs no upstream read at all
func TestConditionalRequestCostsNothing(t *testing.T) {
	path := storePath("/share/applications/example.desktop")

	served, _ := newServer(t)

	first := get(t, served, "GET", path)
	require.Equal(t, http.StatusOK, first.Code)

	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// measured against nothing, not against a parser still winding down
	srv, cache := newServer(t)

	rec := conditionalGet(t, srv, path, etag)

	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, etag, rec.Header().Get("ETag"))
	assert.Zero(t, cache.bytesRead(narURL),
		"a revalidated request should not read the archive at all")
	assert.Zero(t, cache.bytesRead(storeHash+".narinfo"),
		"nor should it need the narinfo")
}

func TestConditionalRequestAcceptsWeakAndLists(t *testing.T) {
	srv, _ := newServer(t)

	path := storePath("/share/applications/example.desktop")
	etag := get(t, srv, "GET", path).Header().Get("ETag")
	require.NotEmpty(t, etag)

	for _, header := range []string{
		"W/" + etag,
		`"something-else", ` + etag,
	} {
		rec := conditionalGet(t, srv, path, header)
		assert.Equal(t, http.StatusNotModified, rec.Code, header)
	}
}

// `*` is not honoured: existence is unknown before the archive is walked
func TestConditionalRequestDoesNotOvermatch(t *testing.T) {
	srv, _ := newServer(t)

	path := storePath("/share/applications/example.desktop")

	for _, header := range []string{`"nope"`, "*"} {
		rec := conditionalGet(t, srv, path, header)

		assert.Equal(t, http.StatusOK, rec.Code, header)
		assert.Equal(t, desktopEntry, rec.Body.String(), header)
	}
}

// or a client holding one would be told it already has the other
func TestEtagsDifferPerPath(t *testing.T) {
	srv, _ := newServer(t)

	first := get(t, srv, "GET", storePath("/share/applications/example.desktop"))
	second := get(t, srv, "GET", storePath("/share/doc/readme"))

	assert.NotEqual(t, first.Header().Get("ETag"), second.Header().Get("ETag"))
}
