package unpack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchivePath(t *testing.T) {
	for _, tt := range []struct {
		mountPath string
		urlPath   string
		expected  string
	}{
		// The store path's own directory is not part of the path inside it.
		{"/nix/store/", "/nix/store/00000000000000000000000000000000-example", "/"},
		{"/nix/store/", "/nix/store/00000000000000000000000000000000-example/", "/"},
		{"/nix/store/", "/nix/store/00000000000000000000000000000000-example/bin/sh", "/bin/sh"},
		{"/nix/store/", "/nix/store/00000000000000000000000000000000-example/bin/sh/", "/bin/sh"},

		// A store somewhere other than `/nix/store` is one component deeper,
		// so a fixed count would cut the path in the wrong place.
		{"/var/lib/nix/store/", "/var/lib/nix/store/00000000000000000000000000000000-example/bin/sh", "/bin/sh"},
		{"/var/lib/nix/store/", "/var/lib/nix/store/00000000000000000000000000000000-example", "/"},

		// Served by hash sub-domain, the request path is already the one
		// inside the archive.
		{"/nix/store/", "/bin/sh", "/bin/sh"},
		{"/nix/store/", "/", "/"},
	} {
		assert.Equal(t, tt.expected, archivePath(tt.mountPath, tt.urlPath),
			"archivePath(%q, %q)", tt.mountPath, tt.urlPath)
	}
}
