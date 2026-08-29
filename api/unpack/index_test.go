package unpack_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/numtide/nar-serve/api/unpack"
	"github.com/numtide/nar-serve/pkg/nar"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	storeDir  = "/nix/store/"
	storeHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	storeName = storeHash + "-example-1.0"
	narURL    = "nar/example.nar"

	desktopEntry = "[Desktop Entry]\nName=Example\n"
)

// memCache is an in-memory BinaryCacheReader that records how much of each
// file was read, so a test can tell a full scan from one that stopped early.
type memCache struct {
	files map[string][]byte
	read  map[string]*int64
}

func newMemCache(files map[string][]byte) *memCache {
	cache := &memCache{files: files, read: map[string]*int64{}}
	for name := range files {
		var n int64

		cache.read[name] = &n
	}

	return cache
}

func (c *memCache) URL() string { return "mem://" }

func (c *memCache) FileExists(_ context.Context, path string) (bool, error) {
	_, ok := c.files[path]

	return ok, nil
}

func (c *memCache) GetFile(_ context.Context, path string) (io.ReadCloser, error) {
	body, ok := c.files[path]
	if !ok {
		return nil, fmt.Errorf("no such file: %s", path)
	}

	return &countingReader{r: bytes.NewReader(body), n: c.read[path]}, nil
}

// bytesRead reports how many bytes of path were pulled out of the cache.
func (c *memCache) bytesRead(path string) int64 {
	return atomic.LoadInt64(c.read[path])
}

type countingReader struct {
	r io.Reader
	n *int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	atomic.AddInt64(c.n, int64(n))

	return n, err
}

func (c *countingReader) Close() error { return nil }

// entry describes one NAR member. A zero linkTo and dir make it a regular
// file, whatever contents says.
type entry struct {
	path     string
	contents string
	linkTo   string
	dir      bool
}

func buildNAR(t *testing.T, entries []entry) []byte {
	t.Helper()

	var buf bytes.Buffer

	narWriter, err := nar.NewWriter(&buf)
	require.NoError(t, err)

	for _, e := range entries {
		hdr := &nar.Header{Path: e.path}

		switch {
		case e.dir:
			hdr.Type = nar.TypeDirectory
		case e.linkTo != "":
			hdr.Type = nar.TypeSymlink
			hdr.LinkTarget = e.linkTo
		default:
			hdr.Type = nar.TypeRegular
			hdr.Size = int64(len(e.contents))
		}

		require.NoError(t, narWriter.WriteHeader(hdr))

		if hdr.Type == nar.TypeRegular {
			_, err := io.WriteString(narWriter, e.contents)
			require.NoError(t, err)
		}
	}

	require.NoError(t, narWriter.Close())

	return buf.Bytes()
}

// The fixture deliberately contains `/lib` next to `/libexec` and
// `/share/applications` next to `/share/applications-extra`: both pairs are
// ones where treating a path as a plain string prefix gets the answer wrong.
// The large file at the end makes a full scan visibly more expensive than one
// that stops where it can.
func testNAR(t *testing.T) []byte {
	t.Helper()

	return buildNAR(t, []entry{
		{path: "/", dir: true},
		{path: "/bin", dir: true},
		{path: "/bin/example", contents: "#!/bin/sh\n"},
		{path: "/lib", dir: true},
		{path: "/lib/thing.so", contents: "lib"},
		{path: "/libexec", dir: true},
		{path: "/libexec/helper", contents: "helper"},
		{path: "/share", dir: true},
		{path: "/share/applications", dir: true},
		{path: "/share/applications/example.desktop", contents: desktopEntry},
		{path: "/share/applications-extra", dir: true},
		{path: "/share/applications-extra/other", contents: "other"},
		{path: "/share/doc", dir: true},
		{path: "/share/doc/big", contents: strings.Repeat("x", 1<<16)},
		{path: "/share/link", linkTo: "applications/example.desktop"},
	})
}

func newServer(t *testing.T) (http.Handler, *memCache) {
	t.Helper()

	narinfo := fmt.Sprintf("StorePath: %s%s\nURL: %s\nCompression: none\n",
		storeDir, storeName, narURL)

	cache := newMemCache(map[string][]byte{
		storeHash + ".narinfo": []byte(narinfo),
		narURL:                 testNAR(t),
	})

	handler := unpack.NewHandler(cache, storeDir)

	router := chi.NewRouter()
	router.Use(middleware.CleanPath)
	router.Use(middleware.GetHead)
	router.Method("GET", handler.MountPath()+"{narDir}", handler)
	router.Method("GET", handler.MountPath()+"{narDir}/*", handler)

	return router, cache
}

func get(t *testing.T, srv http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(method, path, nil))

	return rec
}

func storePath(sub string) string {
	return storeDir + storeName + sub
}

func TestServeRegularFile(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/share/applications/example.desktop"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, desktopEntry, rec.Body.String())
	assert.Equal(t, fmt.Sprint(len(desktopEntry)), rec.Header().Get("Content-Length"))
}

func TestServeHeadOmitsBody(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "HEAD", storePath("/share/applications/example.desktop"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, fmt.Sprint(len(desktopEntry)), rec.Header().Get("Content-Length"))
}

func TestServeFileInFirstDirectory(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/bin/example"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "#!/bin/sh\n", rec.Body.String())
}

func TestServeSymlinkRedirects(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/share/link"))

	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, storePath("/share/applications/example.desktop"),
		rec.Header().Get("Location"))
}

func TestServeMissingFile(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/share/applications/absent.desktop"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServeMissingStorePath(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storeDir+"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-absent/bin/x")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServeDirectoryListsOnlyItsChildren(t *testing.T) {
	srv, _ := newServer(t)

	rec := get(t, srv, "GET", storePath("/lib"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), storePath("/lib/thing.so"))
	assert.NotContains(t, rec.Body.String(), storePath("/libexec"),
		"/libexec is a sibling of /lib, not a child of it")
}
