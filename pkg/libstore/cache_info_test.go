package libstore

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCache is a BinaryCacheReader over a fixed set of files.
type stubCache map[string]string

func (c stubCache) FileExists(_ context.Context, path string) (bool, error) {
	_, ok := c[path]

	return ok, nil
}

func (c stubCache) GetFile(_ context.Context, path string) (io.ReadCloser, error) {
	contents, ok := c[path]
	if !ok {
		return nil, fmt.Errorf("no such file: %s", path)
	}

	return io.NopCloser(strings.NewReader(contents)), nil
}

func (c stubCache) URL() string {
	return "stub://"
}

func TestParseCacheInfo(t *testing.T) {
	info, err := ParseCacheInfo(strings.NewReader(
		"StoreDir: /nix/store\nWantMassQuery: 1\nPriority: 40\n",
	))
	require.NoError(t, err)

	assert.Equal(t, "/nix/store", info.StoreDir)
	assert.True(t, info.WantMassQuery)
	assert.Equal(t, 40, info.Priority)
}

func TestParseCacheInfoOtherStoreDir(t *testing.T) {
	info, err := ParseCacheInfo(strings.NewReader("StoreDir: /var/lib/nix/store\n"))
	require.NoError(t, err)

	assert.Equal(t, "/var/lib/nix/store", info.StoreDir)
}

func TestParseCacheInfoDefaults(t *testing.T) {
	// A cache that says nothing about its store holds the default one.
	info, err := ParseCacheInfo(strings.NewReader("Priority: 30\n"))
	require.NoError(t, err)

	assert.Equal(t, DefaultStoreDir, info.StoreDir)
	assert.False(t, info.WantMassQuery)
}

func TestParseCacheInfoIgnoresUnknownKeys(t *testing.T) {
	info, err := ParseCacheInfo(strings.NewReader("StoreDir: /nix/store\nSomethingElse: 1\n"))
	require.NoError(t, err)

	assert.Equal(t, "/nix/store", info.StoreDir)
}

func TestParseCacheInfoRejectsMalformed(t *testing.T) {
	_, err := ParseCacheInfo(strings.NewReader("StoreDir /nix/store\n"))
	assert.Error(t, err)

	_, err = ParseCacheInfo(strings.NewReader("Priority: soon\n"))
	assert.Error(t, err)
}

func TestGetCacheInfo(t *testing.T) {
	cache := stubCache{CacheInfoPath: "StoreDir: /var/lib/nix/store\nPriority: 41\n"}

	info, err := GetCacheInfo(context.Background(), cache)
	require.NoError(t, err)

	assert.Equal(t, "/var/lib/nix/store", info.StoreDir)
	assert.Equal(t, 41, info.Priority)
}

func TestGetCacheInfoMissing(t *testing.T) {
	_, err := GetCacheInfo(context.Background(), stubCache{})
	assert.Error(t, err)
}
