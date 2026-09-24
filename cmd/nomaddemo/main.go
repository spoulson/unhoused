// Command nomaddemo simulates a subset of the Nomad HTTP API with fabricated
// jobs, versions, and allocations, so unhoused can be pointed at it as an
// ordinary profile without needing a real Nomad cluster. It's meant for repo
// maintainers to demo or manually test unhoused against — see
// internal/nomaddemo for what it simulates.
//
// Run alongside unhoused with a profile like:
//
//	profiles:
//	  - name: demo
//	    nomadUrl: http://127.0.0.1:4646
//	    nodeHostnameTemplate: "{node}.demo.local"
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"unhoused/internal/nomaddemo"
)

func main() {
	listenAddr := flag.String("listen", ":4646", "address to listen on")
	flag.Parse()

	server := nomaddemo.NewServer(nil)

	fmt.Printf("nomaddemo (simulated Nomad API) listening on %s\n", *listenAddr)

	err := http.ListenAndServe(*listenAddr, server.Handler())
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
