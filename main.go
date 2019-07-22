package main

import (
	"fmt"
	"net/http"

	"github.com/urfave/negroni"
	"github.com/zimbatm/nar-serve/handler"
)

func main() {
	n := negroni.Classic() // Includes some default middlewares
	n.UseHandler(http.HandlerFunc(handler.Handler))

	addr := ":3000"
	fmt.Println("Starting server on address", addr)
	err := http.ListenAndServe(addr, n)
	if err != nil {
		panic(err)
	}
}
