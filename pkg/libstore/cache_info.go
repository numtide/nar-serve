package libstore

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DefaultStoreDir is the store a cache holds paths for unless it says otherwise.
const DefaultStoreDir = "/nix/store"

// CacheInfoPath is where a binary cache publishes its CacheInfo.
const CacheInfoPath = "nix-cache-info"

// CacheInfo is what a binary cache says about itself.
type CacheInfo struct {
	// StoreDir is the store the cached paths belong to. Paths from a store at
	// one prefix are not valid under another, so this decides where a client
	// may serve them from.
	StoreDir string

	// WantMassQuery is whether the cache is happy to be asked about paths it
	// may not have.
	WantMassQuery bool

	// Priority orders this cache against others; the lower one is consulted
	// first.
	Priority int
}

// GetCacheInfo reads the cache's `nix-cache-info`.
func GetCacheInfo(ctx context.Context, cache BinaryCacheReader) (*CacheInfo, error) {
	f, err := cache.GetFile(ctx, CacheInfoPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseCacheInfo(f)
}

// ParseCacheInfo reads the `Key: value` lines of a `nix-cache-info` file.
// Unknown keys are skipped, since a cache may state more than a reader knows
// to ask about.
func ParseCacheInfo(r io.Reader) (*CacheInfo, error) {
	info := &CacheInfo{StoreDir: DefaultStoreDir}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			return nil, fmt.Errorf("malformed %s line: %q", CacheInfoPath, line)
		}

		value = strings.TrimSpace(value)

		switch key {
		case "StoreDir":
			info.StoreDir = value
		case "WantMassQuery":
			info.WantMassQuery = value == "1"
		case "Priority":
			priority, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("%s Priority: %w", CacheInfoPath, err)
			}

			info.Priority = priority
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return info, nil
}
