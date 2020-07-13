package main

// this file is being used for local development. It reproduces more
// or less the behaviour of now.sh but all compiled into a single binary.

import (
	"net/http"

	unpack "github.com/numtide/nar-serve/api/unpack"
	"github.com/urfave/negroni"
)

const robotsTxt = `
User-agent: *
Disallow: /nix/store
`

const indexPage = `
<!DOCTYPE html>
<html>
<head>
  <meta content="text/html;charset=utf-8" http-equiv="Content-Type"/>
  <title>nar-serve</title>
  <link rel=stylesheet
        href="https://cdnjs.cloudflare.com/ajax/libs/Primer/11.0.0/build.css">
</head>
<body>
  <div class="container-lg px-3 my-5 markdown-body">
    <h1>nar-serve</h1>

    <p>All the files in <a href="https://cache.nixos.org">cache.nixos.org</a> are packed in NAR files which makes them not directly accessible. This service allows to dowload, decompress, unpack and serve any file in the cache on the fly.</p>

    <h2>Use cases</h2>

    <ul>
      <li>Avoid publishing build artifacts to both the binary cache and
        another service.</li>
      <li>Allows to share build results easily.</li>
      <li>Inspect the content of a NAR file.</li>
    </ul>

    <h2>Usage</h2>
    <ol>
      <li>Pick a full store path in your filesystem.</li>
      <li>Paste it in the form below.</li>
      <li>Click submit. TADA!</li>
    </ol>

    <dl class="form-group">
      <dt><label for=reformat-input>Store path</label></dt>
      <dd>
        <input type=text id=store-path class="form-control input-block">
      </dd>
    </dl>

    <button class="btn btn-primary" id=store-path-submit>Load</button>

    <h2>Examples</h2>
    <ul>
      <li><a href="/nix/store/zk5crljigizl5snkfyaijja89bb6228x-rake-12.3.1/bin/rake">readlink -f $(which rake)</a></li>
    <li><a href="/nix/store/barxv95b8arrlh97s6axj8k7ljn7aky1-go-1.12/share/go/doc/effective_go.html">/nix/store/barxv95b8arrlh97s6axj8k7ljn7aky1-go-1.12/share/go/doc/effective_go.html</a></li>
    </ul>


<hr>
<p>
Like this project? Star it on <a href="https://github.com/numtide/nar-serve">GitHub</a>.

<script>
const storePathEl = document.getElementById("store-path");
const storePathSubmitEl = document.getElementById("store-path-submit");

storePathSubmitEl.onclick = function() {
  document.location.pathname = storePathEl.value;
}
</script>
`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(indexPage))
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(robotsTxt))
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {
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
	n.Run()
}
