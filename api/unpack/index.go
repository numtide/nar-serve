package unpack

import (
	"context"
	"fmt"
	"io"
	"log"
	"maps"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/numtide/nar-serve/pkg/compression"
	"github.com/numtide/nar-serve/pkg/libstore"
	"github.com/numtide/nar-serve/pkg/metrics"
	"github.com/numtide/nar-serve/pkg/nar"
	"github.com/numtide/nar-serve/pkg/nar/ls"
	"github.com/numtide/nar-serve/pkg/narinfo"

	"github.com/go-chi/chi/v5"
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

	// the url carries the hash, so this answers before anything is fetched.
	// the entry type is still unknown here, but only file bytes are ever given
	// an etag, so a match can only be a client holding those bytes.
	etag := etagFor(narHash, newPath)
	if etagMatches(req.Header.Get("If-None-Match"), etag) {
		setImmutable(w, etag)
		w.WriteHeader(http.StatusNotModified)

		return
	}

	// Get the NAR info to find the NAR
	narinfo, err := getNarInfo(ctx, h.cache, narHash)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// a listing answers everything but a file's bytes without touching the nar
	if listing := getListing(ctx, h.cache, narHash); listing != nil {
		node := listing.Lookup(newPath)
		if node == nil {
			http.Error(w, "file not found", 404)

			return
		}

		switch node.Type {
		case nar.TypeDirectory:
			writeDirectoryHeader(w, newPath)

			if err := writeListedEntries(w, narinfo.StorePath, newPath, node); err != nil {
				http.Error(w, err.Error(), 500)
			}

			return
		case nar.TypeSymlink:
			redirectSymlink(w, req, h.mountPath, absSymlink(narinfo.StorePath, newPath, node.LinkTarget))

			return
		case nar.TypeRegular:
			if req.Method == "HEAD" {
				setFileHeaders(w, etag, newPath, node.Size, node.Executable)

				return
			}
		}
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

	r, err := compression.NewReader(narinfo.Compression, metrics.Count(file, metrics.UpstreamBytes))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	narReader, err := nar.NewReader(metrics.Count(r, metrics.ArchiveBytes))
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
				writeDirectoryHeader(w, hdr.Path)

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

					if err := writeDirectoryEntry(w, narinfo.StorePath, hdr2.Path, hdr2.Type, hdr2.LinkTarget); err != nil {
						http.Error(w, err.Error(), 500)

						return
					}
				}
			case nar.TypeSymlink:
				redirectSymlink(w, req, h.mountPath, absSymlink(narinfo.StorePath, hdr.Path, hdr.LinkTarget))
			case nar.TypeRegular:
				setFileHeaders(w, etag, hdr.Path, hdr.Size, hdr.Executable)

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

// getListing fetches the .ls that a cache written with write-nar-listing
// keeps beside each narinfo. Most caches have none, so whatever goes wrong
// here only means the archive has to be walked.
func getListing(ctx context.Context, nixCache libstore.BinaryCacheReader, key string) *ls.Root {
	r, err := nixCache.GetFile(ctx, key+".ls")
	if err != nil {
		return nil
	}
	defer r.Close()

	listing, err := ls.ParseLS(metrics.Count(r, metrics.UpstreamBytes))
	if err != nil {
		log.Println("ignoring listing:", err)

		return nil
	}

	return listing
}

func absSymlink(storePath, path, target string) string {
	if filepath.IsAbs(target) {
		return target
	}

	return filepath.Join(storePath, filepath.Dir(path), target)
}

func redirectSymlink(w http.ResponseWriter, req *http.Request, mountPath, target string) {
	if !strings.HasPrefix(target, mountPath) {
		fmt.Fprintf(w, "found symlink out of store: %s\n", target)

		return
	}

	http.Redirect(w, req, target, http.StatusMovedPermanently)
}

func setFileHeaders(w http.ResponseWriter, etag, path string, size int64, executable bool) {
	ctype := mime.TypeByExtension(filepath.Ext(path))
	if ctype == "" {
		ctype = "application/octet-stream"
		// TODO: use http.DetectContentType as a fallback
	}

	if executable {
		w.Header().Set("NAR-Executable", "1")
	}

	setImmutable(w, etag)
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
}

func writeDirectoryHeader(w http.ResponseWriter, path string) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<p>%s is a directory:</p><ol>", path)
	flush(w)
}

func writeDirectoryEntry(w http.ResponseWriter, storePath, path string, nodeType nar.NodeType, linkTarget string) error {
	var label string

	switch nodeType {
	case nar.TypeDirectory:
		label = path + "/"
	case nar.TypeSymlink:
		label = path + " -> " + absSymlink(storePath, path, linkTarget)
	case nar.TypeRegular:
		label = path
	default:
		return fmt.Errorf("BUG: unknown NAR header type: %s", nodeType)
	}

	fmt.Fprintf(w, "<li><a href='%s'>%s</a></li>", filepath.Join(storePath, path), label)
	flush(w)

	return nil
}

// writeListedEntries walks a listing the way the archive lays its entries
// out, names in byte order and a directory's children right behind it, so
// the page is the same whichever way it was made.
func writeListedEntries(w http.ResponseWriter, storePath, dir string, node *ls.Node) error {
	for _, name := range slices.Sorted(maps.Keys(node.Entries)) {
		child := node.Entries[name]
		path := filepath.Join(dir, name)

		if err := writeDirectoryEntry(w, storePath, path, child.Type, child.LinkTarget); err != nil {
			return err
		}

		if child.Type == nar.TypeDirectory {
			if err := writeListedEntries(w, storePath, path, child); err != nil {
				return err
			}
		}
	}

	return nil
}

func flush(rw http.ResponseWriter) {
	f, ok := rw.(http.Flusher)
	if !ok {
		panic("ResponseWriter is not a Flusher")
	}
	f.Flush()
}
