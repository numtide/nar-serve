package main
// this file is being used for local development. It reproduces more
// or less the behaviour of now.sh but all compiled into a single binary.

import (
	"fmt"
	"net/http"

	"github.com/urfave/negroni"
	unpack "github.com/zimbatm/nar-serve/api/unpack"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(unpack.MountPath, unpack.Handler)

	// Includes some default middlewares
	// Serve static files from ./public
	n := negroni.Classic()
	n.UseHandler(mux)

	addr := ":3000"
	fmt.Println("Starting server on address", addr)
	err := http.ListenAndServe(addr, n)
	if err != nil {
		panic(err)
	}
}
