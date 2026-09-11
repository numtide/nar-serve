package unpack_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"testing"

	"github.com/numtide/nar-serve/pkg/nar"
	"github.com/numtide/nar-serve/pkg/nar/ls"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildListing writes the .ls a cache keeps next to the narinfo, from the
// same entries buildNAR gets.
func buildListing(t *testing.T, entries []entry) []byte {
	t.Helper()

	root := ls.Root{Version: 1}

	for _, e := range entries {
		node := &ls.Node{}

		switch {
		case e.dir:
			node.Type = nar.TypeDirectory
			node.Entries = map[string]*ls.Node{}
		case e.linkTo != "":
			node.Type = nar.TypeSymlink
			node.LinkTarget = e.linkTo
		default:
			node.Type = nar.TypeRegular
			node.Size = int64(len(e.contents))
			node.Executable = e.executable
		}

		if e.path == "/" {
			root.Root = *node

			continue
		}

		parent := root.Lookup(path.Dir(e.path))
		require.NotNil(t, parent, e.path)
		parent.Entries[path.Base(e.path)] = node
	}

	out, err := json.Marshal(root)
	require.NoError(t, err)

	return out
}

func newListingServer(t *testing.T) (http.Handler, *memCache) {
	t.Helper()

	return newServerWith(t, map[string][]byte{
		storeHash + ".ls": buildListing(t, testEntries()),
	})
}

func TestListingAnswersMissingFileWithoutTheNAR(t *testing.T) {
	srv, cache := newListingServer(t)

	rec := get(t, srv, "GET", storePath("/share/applications/absent.desktop"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Zero(t, cache.bytesRead(narURL))
}

func TestListingAnswersHeadWithoutTheNAR(t *testing.T) {
	srv, cache := newListingServer(t)

	rec := get(t, srv, "HEAD", storePath("/bin/example"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, fmt.Sprint(len("#!/bin/sh\n")), rec.Header().Get("Content-Length"))
	assert.Equal(t, "1", rec.Header().Get("NAR-Executable"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
	assert.Zero(t, cache.bytesRead(narURL))
}

func TestListingRedirectsSymlinkWithoutTheNAR(t *testing.T) {
	srv, cache := newListingServer(t)

	rec := get(t, srv, "GET", storePath("/share/link"))

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, storePath("/share/applications/example.desktop"),
		rec.Header().Get("Location"))
	assert.Zero(t, cache.bytesRead(narURL))
}

func TestListingServesDirectoryWithoutTheNAR(t *testing.T) {
	srv, cache := newListingServer(t)

	rec := get(t, srv, "GET", storePath("/lib"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), storePath("/lib/thing.so"))
	assert.NotContains(t, rec.Body.String(), storePath("/libexec"))
	assert.Zero(t, cache.bytesRead(narURL))
}

// A page made from the listing has to match one made by walking the archive,
// or a cache gaining listings would change what its users see.
func TestListingRendersTheSamePageAsTheScan(t *testing.T) {
	scan, _ := newServer(t)
	listed, _ := newListingServer(t)

	for _, dir := range []string{"", "/share", "/share/applications"} {
		want := get(t, scan, "GET", storePath(dir))
		got := get(t, listed, "GET", storePath(dir))

		require.Equal(t, http.StatusOK, want.Code, dir)
		assert.Equal(t, want.Body.String(), got.Body.String(), dir)
		assert.Equal(t, want.Header(), got.Header(), dir)
	}
}

func TestListingStillReadsFileBytesFromTheNAR(t *testing.T) {
	srv, cache := newListingServer(t)

	rec := get(t, srv, "GET", storePath("/share/applications/example.desktop"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, desktopEntry, rec.Body.String())
	assert.Positive(t, cache.bytesRead(narURL))
}

func TestBrokenListingFallsBackToTheNAR(t *testing.T) {
	srv, cache := newServerWith(t, map[string][]byte{
		storeHash + ".ls": []byte("not json"),
	})

	rec := get(t, srv, "GET", storePath("/share/applications/absent.desktop"))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Positive(t, cache.bytesRead(narURL))

	rec = get(t, srv, "GET", storePath("/bin/example"))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "#!/bin/sh\n", rec.Body.String())
}
