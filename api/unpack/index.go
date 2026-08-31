package unpack

import (
	"compress/bzip2"
	"context"
	"fmt"
	"log"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/numtide/nar-serve/pkg/libstore"
	"github.com/numtide/nar-serve/pkg/nar"
	"github.com/numtide/nar-serve/pkg/narinfo"

	"github.com/go-chi/chi/v5"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type Handler struct {
	cache     libstore.BinaryCacheReader
	mountPath string
}

func NewHandler(cache libstore.BinaryCacheReader, mountPath string) *Handler {
	return &Handler{
		cache:     cache,
		mountPath: mountPath,
	}
}

// MountPath is where this handler is supposed to be mounted
func (h *Handler) MountPath() string {
	return h.mountPath
}

// Handler is the entry-point for @now/go as well as the stub main.go net/http
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	narDir := chi.URLParam(req, "narDir")
	if narDir == "" {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "store path missing", 404)
		return
	}

	narHash := strings.Split(narDir, "-")[0]

	h.ServeNAR(narHash, w, req)
}

// archivePath turns a request path into the path to look for inside the
// archive, by taking off the mount path and the store path's own directory.
// Trailing slashes are not significant.
func archivePath(mountPath, urlPath string) string {
	path := strings.TrimRight(urlPath, "/")

	if strings.HasPrefix(path, mountPath) {
		// The mount path is whatever store the cache holds paths for, which is
		// not always `/nix/store`, so count its components rather than assume
		// there are three. One more comes off for the store path itself.
		skip := len(strings.Split(strings.TrimSuffix(mountPath, "/"), "/")) + 1

		components := strings.Split(path, "/")
		if len(components) > skip {
			path = strings.Join(components[skip:], "/")
		} else {
			path = ""
		}
	}

	return "/" + strings.TrimLeft(path, "/")
}

func (h *Handler) ServeNAR(narHash string, w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	log.Println("narHash=", narHash)

	newPath := archivePath(h.mountPath, req.URL.Path)
	log.Println("newPath=", newPath)

	// Get the NAR info to find the NAR
	narinfo, err := getNarInfo(ctx, h.cache, narHash)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// TODO: consider keeping a LRU cache
	narPATH := narinfo.URL
	log.Println("fetching the NAR:", narPATH)
	file, err := h.cache.GetFile(ctx, narPATH)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	var r io.Reader
	r = file

	// decompress on the fly
	switch narinfo.Compression {
	case "none":
		// The NAR is stored verbatim. `narinfo.Parse` turns an absent
		// `Compression` field into `bzip2` rather than this, so `none` only
		// ever comes from a cache that states it.
	case "xz":
		r, err = xz.NewReader(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "bzip2":
		r = bzip2.NewReader(r)
	case "zstd":
		r, err = zstd.NewReader(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	default:
		http.Error(w, fmt.Sprintf("compression %s not handled", narinfo.Compression), 500)
		return
	}

	// TODO: try to load .ls files to speed-up the file lookups

	narReader, err := nar.NewReader(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer narReader.Close()

	for {
		hdr, err := narReader.Next()
		if err != nil {
			if err == io.EOF {
				http.Error(w, "file not found", 404)
			} else {
				http.Error(w, err.Error(), 500)
			}
			return
		}

		// we've got a match!
		if hdr.Path == newPath {
			switch hdr.Type {
			case nar.TypeDirectory:
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, "<p>%s is a directory:</p><ol>", hdr.Path)
				flush(w)

				// The directory's own path is a prefix of its siblings' paths
				// as well as its children's, so match against it with the
				// separator attached: `/libexec` is not inside `/lib`. The
				// root is already `/`, and trimming keeps it that way.
				prefix := strings.TrimSuffix(hdr.Path, "/") + "/"

				for {
					hdr2, err := narReader.Next()
					if err != nil {
						if err != io.EOF {
							http.Error(w, err.Error(), 500)
						}

						break
					}

					if !strings.HasPrefix(hdr2.Path, prefix) {
						break
					}

					var label string
					switch hdr2.Type {
					case nar.TypeDirectory:
						label = hdr2.Path + "/"
					case nar.TypeSymlink:
						label = hdr2.Path + " -> " + absSymlink(narinfo, hdr2)
					case nar.TypeRegular:
						label = hdr2.Path
					default:
						http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr2.Type), 500)

						return
					}

					fmt.Fprintf(w, "<li><a href='%s'>%s</a></li>", filepath.Join(narinfo.StorePath, hdr2.Path), label)
					flush(w)
				}
			case nar.TypeSymlink:
				redirectPath := absSymlink(narinfo, hdr)

				// Make sure the symlink is absolute

				if !strings.HasPrefix(redirectPath, h.mountPath) {
					fmt.Fprintf(w, "found symlink out of store: %s\n", redirectPath)
				} else {
					http.Redirect(w, req, redirectPath, http.StatusMovedPermanently)
				}
			case nar.TypeRegular:
				// TODO: ETag header matching. Use the NAR file name as the ETag
				// TODO: expose the executable flag somehow?
				ctype := mime.TypeByExtension(filepath.Ext(hdr.Path))
				if ctype == "" {
					ctype = "application/octet-stream"
					// TODO: use http.DetectContentType as a fallback
				}

				if hdr.Executable {
					w.Header().Set("NAR-Executable", "1")
				}

				w.Header().Set("Cache-Control", "immutable")
				w.Header().Set("Content-Type", ctype)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", hdr.Size))
				if req.Method != "HEAD" {
					_, _ = io.CopyN(w, narReader, hdr.Size)
				}
			default:
				http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr.Type), 500)
			}
			return
		}

		// NAR entries are ordered, so the wanted path can only appear while
		// the scan is still short of where its name would sort. Once an entry
		// comes back from beyond it, the archive does not contain the path and
		// there is nothing to gain from decompressing the remainder.
		if !nar.PathIsLexicographicallyOrdered(hdr.Path, newPath) {
			http.Error(w, "file not found", 404)

			return
		}
	}
}

// TODO: consider keeping a LRU cache
func getNarInfo(ctx context.Context, nixCache libstore.BinaryCacheReader, key string) (*narinfo.NarInfo, error) {
	path := fmt.Sprintf("%s.narinfo", key)
	fmt.Println("Fetching the narinfo:", path, "from:", nixCache.URL())
	r, err := nixCache.GetFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	ni, err := narinfo.Parse(r)
	if err != nil {
		return nil, err
	}
	return ni, err
}

func absSymlink(narinfo *narinfo.NarInfo, hdr *nar.Header) string {
	if filepath.IsAbs(hdr.LinkTarget) {
		return hdr.LinkTarget
	}

	return filepath.Join(narinfo.StorePath, filepath.Dir(hdr.Path), hdr.LinkTarget)
}

func flush(rw http.ResponseWriter) {
	f, ok := rw.(http.Flusher)
	if !ok {
		panic("ResponseWriter is not a Flusher")
	}
	f.Flush()
}
