//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package main

import (
	"log"
	"net/http"

	"github.com/r0wbrt/riot/pkg/riotserver"
)

func main() {
	halServer := &riotserver.Server{
		Name:        "Empty Server",
		Description: "This server has no streams for clients to consume.",
		GUID:        "FFFFFFFF00000002",
		PathPrefix:  "/",
	}

	httpServer := http.Server{Handler: halServer, Addr: ":7468"}

	err := httpServer.ListenAndServe()
	log.Fatal(err)
}
