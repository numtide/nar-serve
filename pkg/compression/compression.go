// Package compression opens the encodings a binary cache stores files under.
package compression

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// NewReader decompresses r according to name, which is what a narinfo's
// Compression field and an HTTP Content-Encoding header both carry.
func NewReader(name string, r io.Reader) (io.Reader, error) {
	switch name {
	case "", "none", "identity":
		return r, nil
	case "xz":
		return xz.NewReader(r)
	case "bzip2":
		return bzip2.NewReader(r), nil
	case "gzip":
		return gzip.NewReader(r)
	case "zstd":
		return zstd.NewReader(r)
	case "br":
		return brotli.NewReader(r), nil
	default:
		return nil, fmt.Errorf("compression %s not handled", name)
	}
}

// Decode is NewReader for a stream that has to be closed once it is read.
func Decode(name string, rc io.ReadCloser) (io.ReadCloser, error) {
	r, err := NewReader(name, rc)
	if err != nil {
		rc.Close()

		return nil, err
	}

	return struct {
		io.Reader
		io.Closer
	}{r, rc}, nil
}
