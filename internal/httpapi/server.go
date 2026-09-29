// Package httpapi implements unhoused's REST API, described in specs/api.md.
package httpapi

import (
	"net/http"
	"time"

	"unhoused/internal/config"
	"unhoused/internal/nomadclient"
)

// Server serves unhoused's REST API.
type Server struct {
	cfg     *config.Config
	clients map[string]nomadclient.API
	now     func() time.Time
}

// NewServer builds a Server. clients must have one entry per profile in
// cfg.Profiles, keyed by profile name.
func NewServer(cfg *config.Config, clients map[string]nomadclient.API) *Server {
	return &Server{
		cfg:     cfg,
		clients: clients,
		now:     time.Now,
	}
}

// Handler returns the http.Handler serving the /api routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/profiles", s.handleListProfiles)
	mux.HandleFunc("GET /api/profiles/{profile}/jobs", s.handleListJobs)
	mux.HandleFunc("GET /api/profiles/{profile}/jobs/{jobId}", s.handleJobStatus)
	return mux
}

// profile looks up a configured profile and its Nomad client by its URI slug (see config.Profile.EffectiveSlug).
func (s *Server) profile(slug string) (config.Profile, nomadclient.API, bool) {
	for _, p := range s.cfg.Profiles {
		if p.EffectiveSlug() != slug {
			continue
		}

		client, ok := s.clients[p.Name]
		if !ok {
			return config.Profile{}, nil, false
		}

		return p, client, true
	}

	return config.Profile{}, nil, false
}
