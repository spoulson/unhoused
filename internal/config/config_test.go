package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("writing test config file: %v", err)
	}

	return path
}

func TestLoadAppliesDefaults(t *testing.T) {
	path := writeConfigFile(t, `
profiles:
  - name: prod-usw1
    environment: production
    region: us-west1
    nomadUrl: http://10.0.0.1:4646
    nomadToken: secret
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.HTTPPublicURL != defaultHTTPPublicURL {
		t.Errorf("HTTPPublicURL = %q, want %q", cfg.HTTPPublicURL, defaultHTTPPublicURL)
	}
	if cfg.ListenPort != defaultListenPort {
		t.Errorf("ListenPort = %d, want %d", cfg.ListenPort, defaultListenPort)
	}
	if cfg.RefreshIntervalSeconds != defaultRefreshIntervalSeconds {
		t.Errorf("RefreshIntervalSeconds = %d, want %d", cfg.RefreshIntervalSeconds, defaultRefreshIntervalSeconds)
	}
}

func TestLoadHonorsExplicitValues(t *testing.T) {
	path := writeConfigFile(t, `
httpPublicUrl: https://unhoused.example.com
listenPort: 9000
refreshIntervalSeconds: 10
profiles:
  - name: staging-euw1
    environment: staging
    region: europe-west1
    nomadUrl: http://10.0.0.2:4646
    nomadToken: secret
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.HTTPPublicURL != "https://unhoused.example.com" {
		t.Errorf("HTTPPublicURL = %q", cfg.HTTPPublicURL)
	}
	if cfg.ListenPort != 9000 {
		t.Errorf("ListenPort = %d", cfg.ListenPort)
	}
	if cfg.RefreshIntervalSeconds != 10 {
		t.Errorf("RefreshIntervalSeconds = %d", cfg.RefreshIntervalSeconds)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "no profiles",
			yaml:    "profiles: []",
			wantErr: "at least one profile is required",
		},
		{
			name: "missing profile name",
			yaml: `
profiles:
  - environment: staging
    region: us-west1
    nomadUrl: http://10.0.0.1:4646
`,
			wantErr: "name is required",
		},
		{
			name: "duplicate profile name",
			yaml: `
profiles:
  - name: dup
    environment: staging
    region: us-west1
    nomadUrl: http://10.0.0.1:4646
  - name: dup
    environment: production
    region: us-east4
    nomadUrl: http://10.0.0.2:4646
`,
			wantErr: "duplicate profile name",
		},
		{
			name: "invalid environment",
			yaml: `
profiles:
  - name: bad-env
    environment: dev
    region: us-west1
    nomadUrl: http://10.0.0.1:4646
`,
			wantErr: "invalid environment",
		},
		{
			name: "invalid region",
			yaml: `
profiles:
  - name: bad-region
    environment: staging
    region: mars
    nomadUrl: http://10.0.0.1:4646
`,
			wantErr: "invalid region",
		},
		{
			name: "missing nomad url",
			yaml: `
profiles:
  - name: no-url
    environment: staging
    region: us-west1
`,
			wantErr: "nomadUrl is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.yaml)

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load returned no error, want error containing %q", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load returned no error for missing file")
	}
}

func TestShortRegion(t *testing.T) {
	tests := []struct {
		region  Region
		want    string
		wantErr bool
	}{
		{RegionUSWest1, "usw1", false},
		{RegionUSEast4, "use4", false},
		{RegionEuropeWest1, "euw1", false},
		{RegionAUSE1, "ause1", false},
		{Region("mars"), "", true},
	}

	for _, tt := range tests {
		got, err := ShortRegion(tt.region)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ShortRegion(%q) returned no error, want error", tt.region)
			}
			continue
		}
		if err != nil {
			t.Errorf("ShortRegion(%q) returned error: %v", tt.region, err)
		}
		if got != tt.want {
			t.Errorf("ShortRegion(%q) = %q, want %q", tt.region, got, tt.want)
		}
	}
}
