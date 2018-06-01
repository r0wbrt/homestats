//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

//Quick and dirty go routine to do a stress test on a HTTP server.
package main

import (
	"fmt"
	"net/http"
	"time"
)

var path = "http://localhost:7468/FFFFFFFF01000001/"

func main() {
	threads := 100

	for i := 0; i < threads; i++ {
		go connectSite()
	}

	for {
		time.Sleep(time.Minute * 1)
	}
}

func connectSite() {
	for {
		resp, err := http.Get(path)

		if err == nil {
			bytes := make([]byte, 128)

			for err == nil {
				_, err = resp.Body.Read(bytes)
			}

			resp.Body.Close()
		} else {
			fmt.Println(err.Error())
		}
	}
}
