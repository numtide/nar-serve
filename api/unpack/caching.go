package unpack

import (
	"net/http"
	"strings"
)

// store paths are content addressed, so a year cannot go stale
const immutableCacheControl = "public, max-age=31536000, immutable"

// the hash alone determines the content; the path is in the tag so it stays
// meaningful away from the url that produced it
func etagFor(narHash, archivePath string) string {
	return `"` + narHash + ":" + archivePath + `"`
}

// only for bytes read verbatim out of the nar.
//
// not for anything this code synthesizes: a directory listing changes when the
// html does, and a 404 only means the cache had nothing today, which a cache
// that gains content can outlive. neither can be revalidated once a client has
// been told to keep it for a year.
func setImmutable(w http.ResponseWriter, etag string) {
	w.Header().Set("Cache-Control", immutableCacheControl)
	w.Header().Set("ETag", etag)
}

// weak comparison, as If-None-Match requires.
//
// `*` is not honoured: it asks whether the path exists, which is unknown until
// the archive is walked. falling through is slower, never wrong.
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}

	want := strings.TrimPrefix(etag, "W/")

	for _, candidate := range strings.Split(header, ",") {
		if strings.TrimPrefix(strings.TrimSpace(candidate), "W/") == want {
			return true
		}
	}

	return false
}
