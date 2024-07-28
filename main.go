package main

import (
	"embed"
	"net/http"
	"io"
	"os"

	unpack "github.com/numtide/nar-serve/api/unpack"
	"github.com/urfave/negroni"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/robots.txt", robotsHandler)
	mux.HandleFunc(unpack.MountPath, unpack.Handler)

	// Includes some default middlewares
	// Serve static files from ./public
	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
	)
	n.UseHandler(mux)
	n.Run(addr)
}

func getEnv(name, def string) string {
	value := os.Getenv(name)
	if value == "" {
		return def
	}
	return value
}

