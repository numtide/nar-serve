package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/numtide/nar-serve/api/unpack"
	"github.com/numtide/nar-serve/pkg/libstore"
	"github.com/numtide/nar-serve/pkg/nixhash"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/hostrouter"
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

// limitConcurrency bounds how many requests may be walking an archive at once.
// Decompressing up to the wanted file is the whole cost of a request, so this
// is what keeps a burst of lookups from taking over the machine it runs on.
// Requests over the limit wait their turn rather than being refused, and give
// up only if the client does. A limit of zero or less does not restrict
// anything.
func limitConcurrency(limit int) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	slots := make(chan struct{}, limit)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-r.Context().Done():
				http.Error(w, "gave up waiting for a free slot", http.StatusRequestTimeout)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	var (
		port        = getEnv("PORT", "8383")
		addr        = getEnv("HTTP_ADDR", "")
		nixCacheURL = getEnv("NIX_CACHE_URL", getEnv("NAR_CACHE_URL", "https://cache.nixos.org"))
		domain      = getEnv("DOMAIN", "")
	)

	maxConcurrency, err := strconv.Atoi(getEnv("MAX_CONCURRENCY", "0"))
	if err != nil {
		log.Fatalf("MAX_CONCURRENCY: %v", err)
	}

	if addr == "" {
		addr = ":" + port
	}

	cache, err := libstore.NewBinaryCacheReader(context.Background(), nixCacheURL)
	if err != nil {
		panic(err)
	}

	// FIXME: get the mountPath from the binary cache /nix-cache-info file
	storeHandler := unpack.NewHandler(cache, "/nix/store/")

	// Only the archive walk is worth limiting; `/healthz` and the index stay
	// answerable however busy the rest of it is. One limiter, shared by both
	// routes into the store, so the budget is for the process rather than per
	// route.
	limit := limitConcurrency(maxConcurrency)
	limitedStoreHandler := limit(storeHandler)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)

	defaultRouter := chi.NewRouter()
	defaultRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			NixCacheURL string
		}{nixCacheURL}

		if err := indexHTMLTmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	defaultRouter.Get("/healthz", healthzHandler)
	defaultRouter.Get("/robots.txt", robotsHandler)
	defaultRouter.Method("GET", storeHandler.MountPath()+"{narDir}", limitedStoreHandler)
	defaultRouter.Method("GET", storeHandler.MountPath()+"{narDir}/*", limitedStoreHandler)

	if domain != "" {
		narRouter := chi.NewRouter()
		narRouter.Use(limit)
		narRouter.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// First try to find a nix hash in a subdomain.
			narHash := getSubdomain(r.Host)
			algo := nixhash.SHA1
			_, err := nixhash.ParseAny(narHash, &algo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Println("subdomain narHash", narHash)
			storeHandler.ServeNAR(narHash, w, r)
		})

		hr := hostrouter.New()
		hr.Map("*", defaultRouter) // default
		hr.Map("*."+domain, narRouter)

		r.Mount("/", hr)
	} else {
		r.Mount("/", defaultRouter)
	}

	// Front the naked muxer with one that matches sub-domains

	log.Println("domain=", domain)
	log.Println("nixCacheURL=", nixCacheURL)
	log.Println("addr=", addr)
	log.Println("maxConcurrency=", maxConcurrency)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getEnv(name, def string) string {
	value := os.Getenv(name)
	if value == "" {
		return def
	}
	return value
}

func getSubdomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}
