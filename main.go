package main

import (
	"context"
	"log"
	_ "embed"
	"net/http"
	"text/template"
	"os"

	"github.com/numtide/nar-serve/pkg/libstore"
	"github.com/numtide/nar-serve/api/unpack"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed views/index.html
var indexHTML string

var indexHTMLTmpl = template.Must(template.New("index.html").Parse(indexHTML))

//go:embed views/robots.txt
var robotsTXT []byte

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(robotsTXT)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

func main() {
	var (
		port        = getEnv("PORT", "8383")
		addr        = getEnv("HTTP_ADDR", "")
		nixCacheURL = getEnv("NIX_CACHE_URL", getEnv("NAR_CACHE_URL", "https://cache.nixos.org"))
	)

	if addr == "" {
		addr = ":" + port
	}

	cache, err := libstore.NewBinaryCacheReader(context.Background(), nixCacheURL)
	if err != nil {
		panic(err)
	}

	// FIXME: get the mountPath from the binary cache /nix-cache-info file
	h := unpack.NewHandler(cache, "/nix/store/")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			NixCacheURL string
		}{ nixCacheURL }

		if err := indexHTMLTmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.Get("/healthz", healthzHandler)
	r.Get("/robots.txt", robotsHandler)
	r.Method("GET", h.MountPath()+"*", h)

	log.Println("nixCacheURL=", nixCacheURL)
	log.Println("addr=", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getEnv(name, def string) string {
	value := os.Getenv(name)
	if value == "" {
		return def
	}
	return value
}
