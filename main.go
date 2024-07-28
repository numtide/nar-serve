package main

import (
	"embed"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/numtide/nar-serve/api/unpack"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed views/*
var viewsFS embed.FS

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	f, _ := viewsFS.Open("views/index.html")
	_, _ = io.Copy(w, f)
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	f, _ := viewsFS.Open("views/robots.txt")
	_, _ = io.Copy(w, f)
	f.Close()
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

func main() {
	var (
		port = getEnv("PORT", "8383")
		addr = getEnv("HTTP_ADDR", "")
	)

	if addr == "" {
		addr = ":" + port
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.GetHead)

	r.Get("/", indexHandler)
	r.Get("/healthz", healthzHandler)
	r.Get("/robots.txt", robotsHandler)
	r.Get(unpack.MountPath+"*", unpack.Handler)

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
