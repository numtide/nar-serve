package handler

import (
	"compress/bzip2"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"github.com/zimbatm/go-nix/src/libstore"
	"github.com/zimbatm/go-nix/src/nar"
)

var nixCache = libstore.HTTPBinaryCacheStore {
	CacheURI: getEnv("NAR_CACHE_URI", "https://cache.nixos.org"),
}

// TODO: consider keeping a LRU cache
func getNarInfo(key string) (*libstore.NarInfo, error) {
	path := fmt.Sprintf("%s.narinfo", key)
	fmt.Println("Fetching the narinfo:", path, "from:", nixCache.CacheURI)
	r, err := nixCache.GetFile(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return libstore.ParseNarInfo(r)
}

// MountPath is where this handler is supposed to be mounted
const MountPath = "/nix/store/"

// Handler is the entry-point for @now/go as well as the stub main.go net/http
func Handler(w http.ResponseWriter, req *http.Request) {
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
	narinfo, err := getNarInfo(narName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Println("narinfo", narinfo)

	// TODO: consider keeping a LRU cache
	narPATH := narinfo.URL
	fmt.Println("fetching the NAR:", narPATH)
	file, err := nixCache.GetFile(narPATH)
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

	narReader := nar.NewReader(r)
	newPath := strings.Join(components[1:], "/")

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
		if hdr.Name == newPath {
			switch hdr.Type {
			case nar.TypeDirectory:
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, "<p>%s is a directory:</p><ol>", hdr.Name)
				for {
					hdr2, err := narReader.Next()
					if err != nil {
						if err == io.EOF {
							break
						} else {
							http.Error(w, err.Error(), 500)
						}
					}

					if !strings.HasPrefix(hdr2.Name, hdr.Name) {
						break
					}

					var label string
					switch hdr2.Type {
					case nar.TypeDirectory:
						label = hdr2.Name + "/"
					case nar.TypeSymlink:
						label = hdr2.Name + " -> " + absSymlink(narinfo, hdr2)
					case nar.TypeRegular:
						label = hdr2.Name
					default:
						http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr.Type), 500)
					}

					fmt.Fprintf(w, "<li><a href='%s'>%s</a></li>", filepath.Join(narinfo.StorePath, hdr2.Name), label)
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
				ctype := mime.TypeByExtension(filepath.Ext(hdr.Name))
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

func absSymlink(narinfo *libstore.NarInfo, hdr *nar.Header) string {
	if filepath.IsAbs(hdr.Linkname) {
		return hdr.Linkname
	}

	return filepath.Join(narinfo.StorePath, filepath.Dir(hdr.Name), hdr.Linkname)
}
