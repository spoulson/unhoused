// Command unhoused serves a REST API for monitoring HashiCorp Nomad job and
// allocation status, per specs/architecture.md and specs/api.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"unhoused/internal/config"
	"unhoused/internal/httpapi"
	"unhoused/internal/nomadclient"
)

func main() {
	configPath := flag.String("c", "", "path to YAML configuration file (required)")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("missing required -c <file> flag")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	clients := make(map[string]nomadclient.API, len(cfg.Profiles))
	for _, p := range cfg.Profiles {
		client, err := nomadclient.New(p.NomadURL, p.NomadToken)
		if err != nil {
			log.Fatalf("building Nomad client for profile %q: %v", p.Name, err)
		}
		clients[p.Name] = client
	}

	server := httpapi.NewServer(cfg, clients)
	addr := fmt.Sprintf(":%d", cfg.ListenPort)

	log.Printf("unhoused listening on %s", addr)

	err = http.ListenAndServe(addr, server.Handler())
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
