package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(contents), 0o600)
	require.NoError(t, err, "writing test config file")

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
	require.NoError(t, err)

	assert.Equal(t, defaultHTTPPublicURL, cfg.HTTPPublicURL)
	assert.Equal(t, defaultListenPort, cfg.ListenPort)
	assert.Equal(t, defaultRefreshIntervalSeconds, cfg.RefreshIntervalSeconds)
	assert.Equal(t, defaultNodeHostnameTemplate, cfg.Profiles[0].NodeHostnameTemplate)
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
    nodeHostnameTemplate: "{node}.example.internal"
`)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "https://unhoused.example.com", cfg.HTTPPublicURL)
	assert.Equal(t, 9000, cfg.ListenPort)
	assert.Equal(t, 10, cfg.RefreshIntervalSeconds)
	assert.Equal(t, "{node}.example.internal", cfg.Profiles[0].NodeHostnameTemplate)
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
		{
			name: "invalid nodeHostnameTemplate",
			yaml: `
profiles:
  - name: bad-template
    environment: staging
    region: us-west1
    nomadUrl: http://10.0.0.1:4646
    nodeHostnameTemplate: "static.example.com"
`,
			wantErr: "nodeHostnameTemplate must contain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.yaml)

			_, err := Load(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.Error(t, err)
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
		t.Run(string(tt.region), func(t *testing.T) {
			got, err := ShortRegion(tt.region)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
