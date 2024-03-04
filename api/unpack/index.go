package handler

import (
	"compress/bzip2"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/numtide/nar-serve/libstore"

	"github.com/ulikunitz/xz"
)

// MountPath is where this handler is supposed to be mounted
const MountPath = "/nix/store/"

var nixCache = mustBinaryCacheReader()

func mustBinaryCacheReader() libstore.BinaryCacheReader {
	r, err := libstore.NewBinaryCacheReader(context.Background(), getEnv("NAR_CACHE_URL", "https://cache.nixos.org"))
	if err != nil {
		panic(err)
	}
	return r
}

// Handler is the entry-point for @now/go as well as the stub main.go net/http
func Handler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	// remove the mount path from the path
	path := strings.TrimPrefix(req.URL.Path, MountPath)
	// ignore trailing slashes
	path = strings.TrimRight(path, "/")

	components := strings.Split(path, "/")
	if len(components) == 0 {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "store path missing", 404)
		return
	}
	fmt.Println(len(components), components)

	narDir := components[0]
	narName := strings.Split(narDir, "-")[0]

	// Get the NAR info to find the NAR
	narinfo, err := getNarInfo(ctx, narName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Println("narinfo", narinfo)

	// TODO: consider keeping a LRU cache
	narPATH := narinfo.URL
	fmt.Println("fetching the NAR:", narPATH)
	file, err := nixCache.GetFile(ctx, narPATH)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	var r io.Reader
	r = file

	// decompress on the fly
	switch narinfo.Compression {
	case "xz":
		r, err = xz.NewReader(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "bzip2":
		r = bzip2.NewReader(r)
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

	newPath := "/" + strings.Join(components[1:], "/")

	fmt.Println("newPath", newPath)

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
				for {
					hdr2, err := narReader.Next()
					if err != nil {
						if err == io.EOF {
							break
						} else {
							http.Error(w, err.Error(), 500)
						}
					}

					if !strings.HasPrefix(hdr2.Path, hdr.Path) {
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
						http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr.Type), 500)
					}

					fmt.Fprintf(w, "<li><a href='%s'>%s</a></li>", filepath.Join(narinfo.StorePath, hdr2.Path), label)
					flush(w)
				}
			case nar.TypeSymlink:
				redirectPath := absSymlink(narinfo, hdr)

				// Make sure the symlink is absolute

				if !strings.HasPrefix(redirectPath, MountPath) {
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

				w.Header().Set("Cache-Control", "immutable")
				w.Header().Set("Content-Type", ctype)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", hdr.Size))
				if req.Method != "HEAD" {
					io.CopyN(w, narReader, hdr.Size)
				}
			default:
				http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr.Type), 500)
			}
			return
		}

		// TODO: since the nar entries are sorted it's possible to abort early by
		//       comparing the paths
	}
}

func getEnv(name, def string) string {
	value := os.Getenv(name)
	if value == "" {
		return def
	}
	return value
}

// TODO: consider keeping a LRU cache
func getNarInfo(ctx context.Context, key string) (*narinfo.NarInfo, error) {
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
